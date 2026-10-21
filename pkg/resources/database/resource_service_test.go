package database_test

import (
	"context"
	"maps"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/exoscale/terraform-provider-exoscale/pkg/resources/database"
)

// nullAttrs builds a map with every attribute of objType set to its own
// null value, so callers only need to override the attributes they care
// about instead of enumerating the entire schema by hand.
func nullAttrs(objType tftypes.Object) map[string]tftypes.Value {
	attrs := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, at := range objType.AttributeTypes {
		attrs[name] = tftypes.NewValue(at, nil)
	}
	return attrs
}

// TestServiceResourceModifyPlan exercises ServiceResource.ModifyPlan
// directly, without TF_ACC or any API calls, to guard against the
// regressions this hook fixes:
//   - a purely Computed attribute like updated_at defaults to unknown on
//     every plan, even a genuine no-op, unless something pins it back;
//   - naively pinning it (stringplanmodifier.UseStateForUnknown) instead
//     breaks once a real update legitimately changes the value (e.g.
//     node_memory/state during a resize, or updated_at whenever ip_filter
//     is genuinely cleared);
//   - ip_filter's own outcome is resolved directly from config (the
//     Update logic always clears it when config omits it), not inferred
//     by comparing plan to state - see the "clear-only" case below, which
//     regression-tests https://github.com/exoscale/terraform-provider-exoscale/pull/563#discussion_r3537167709
//     where comparison-based inference silently dropped a real clear.
//
// It's run once per engine block (mysql, pg, opensearch, kafka, grafana),
// since ModifyPlan has a separate, hand-written branch resolving ip_filter
// for each one.
func TestServiceResourceModifyPlan(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	res := database.NewServiceResource()

	modifier, ok := res.(resource.ResourceWithModifyPlan)
	if !ok {
		t.Fatal("exoscale_dbaas resource does not implement ResourceWithModifyPlan")
	}

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("unexpected schema diagnostics: %s", schemaResp.Diagnostics)
	}
	sch := schemaResp.Schema

	topType, ok := sch.Type().TerraformType(ctx).(tftypes.Object)
	if !ok {
		t.Fatalf("resource schema type is not an object: %T", sch.Type().TerraformType(ctx))
	}

	ipFilterSetType := tftypes.Set{ElementType: tftypes.String}

	engines := []string{"mysql", "pg", "opensearch", "kafka", "grafana"}

	for _, engine := range engines {
		t.Run(engine, func(t *testing.T) {
			engineType, ok := topType.AttributeTypes[engine].(tftypes.Object)
			if !ok {
				t.Fatalf("%s attribute type is not an object: %T", engine, topType.AttributeTypes[engine])
			}

			// buildObject starts from an all-null top-level object, applies
			// the given overrides, and does the same one level down for the
			// engine block under test (populated so it looks like a
			// resource of that type; every other engine block is left
			// entirely null, as it would be in practice).
			buildObject := func(overrides, engineOverrides map[string]tftypes.Value) tftypes.Value {
				attrs := nullAttrs(topType)
				maps.Copy(attrs, overrides)

				engineAttrs := nullAttrs(engineType)
				maps.Copy(engineAttrs, engineOverrides)
				attrs[engine] = tftypes.NewValue(engineType, engineAttrs)

				return tftypes.NewValue(topType, attrs)
			}

			baseAttrs := map[string]tftypes.Value{
				"id":                     tftypes.NewValue(tftypes.String, "test-id"),
				"name":                   tftypes.NewValue(tftypes.String, "test"),
				"zone":                   tftypes.NewValue(tftypes.String, "bg-sof-1"),
				"type":                   tftypes.NewValue(tftypes.String, engine),
				"plan":                   tftypes.NewValue(tftypes.String, "hobbyist-2"),
				"termination_protection": tftypes.NewValue(tftypes.Bool, false),
				"created_at":             tftypes.NewValue(tftypes.String, "2026-01-01 00:00:00 +0000 UTC"),
				"disk_size":              tftypes.NewValue(tftypes.Number, 10),
				"node_cpus":              tftypes.NewValue(tftypes.Number, 2),
				"node_memory":            tftypes.NewValue(tftypes.Number, 2147483648),
				"nodes":                  tftypes.NewValue(tftypes.Number, 1),
				"state":                  tftypes.NewValue(tftypes.String, "running"),
				"ca_certificate":         tftypes.NewValue(tftypes.String, "ca-cert"),
				"updated_at":             tftypes.NewValue(tftypes.String, "2026-01-01 00:00:00 +0000 UTC"),
			}

			// Used by the "genuine update" and "clear-only" cases below,
			// which need something non-empty to actually clear.
			stateValue := buildObject(baseAttrs, map[string]tftypes.Value{
				"ip_filter": tftypes.NewValue(ipFilterSetType, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "1.2.3.4/32"),
				}),
			})
			state := tfsdk.State{Raw: stateValue, Schema: sch}

			// Used by the "no-op" case: config omits ip_filter, and state
			// already reflects that (a known empty list, matching what the
			// API reports back for a cleared filter), so omitting it truly
			// implies no change - unlike the non-empty state above, where
			// omitting it means a real clear.
			noOpStateValue := buildObject(baseAttrs, map[string]tftypes.Value{
				"ip_filter": tftypes.NewValue(ipFilterSetType, []tftypes.Value{}),
			})
			noOpState := tfsdk.State{Raw: noOpStateValue, Schema: sch}

			ipFilterPath := path.Root(engine).AtName("ip_filter")

			t.Run("no-op plan pins recomputed attributes back to state", func(t *testing.T) {
				planAttrs := map[string]tftypes.Value{}
				maps.Copy(planAttrs, baseAttrs)
				// Nothing in config actually changed, but Terraform's
				// default behavior still marks unconfigured Computed
				// attributes unknown.
				planAttrs["updated_at"] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
				planAttrs["disk_size"] = tftypes.NewValue(tftypes.Number, tftypes.UnknownValue)
				planAttrs["node_cpus"] = tftypes.NewValue(tftypes.Number, tftypes.UnknownValue)
				planAttrs["node_memory"] = tftypes.NewValue(tftypes.Number, tftypes.UnknownValue)
				planAttrs["nodes"] = tftypes.NewValue(tftypes.Number, tftypes.UnknownValue)
				planAttrs["state"] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)

				planValue := buildObject(planAttrs, map[string]tftypes.Value{
					// ip_filter omitted from config -> unknown by default,
					// exactly as if the practitioner never set it.
					"ip_filter": tftypes.NewValue(ipFilterSetType, tftypes.UnknownValue),
				})
				// Config mirrors what the practitioner actually wrote: no
				// ip_filter, same as the plan.
				configValue := buildObject(planAttrs, map[string]tftypes.Value{})

				req := resource.ModifyPlanRequest{
					Config: tfsdk.Config{Raw: configValue, Schema: sch},
					State:  noOpState,
					Plan:   tfsdk.Plan{Raw: planValue, Schema: sch},
				}
				resp := &resource.ModifyPlanResponse{Plan: tfsdk.Plan{Raw: planValue, Schema: sch}}

				modifier.ModifyPlan(ctx, req, resp)
				if resp.Diagnostics.HasError() {
					t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
				}

				var updatedAt types.String
				if diags := resp.Plan.GetAttribute(ctx, path.Root("updated_at"), &updatedAt); diags.HasError() {
					t.Fatalf("unexpected diagnostics reading updated_at: %s", diags)
				}
				if updatedAt.IsUnknown() || updatedAt.ValueString() != "2026-01-01 00:00:00 +0000 UTC" {
					t.Errorf("expected updated_at to be pinned to prior state, got %q (unknown=%v)", updatedAt.ValueString(), updatedAt.IsUnknown())
				}

				var nodeMemory types.Int64
				if diags := resp.Plan.GetAttribute(ctx, path.Root("node_memory"), &nodeMemory); diags.HasError() {
					t.Fatalf("unexpected diagnostics reading node_memory: %s", diags)
				}
				if nodeMemory.IsUnknown() || nodeMemory.ValueInt64() != 2147483648 {
					t.Errorf("expected node_memory to be pinned to prior state, got %v (unknown=%v)", nodeMemory.ValueInt64(), nodeMemory.IsUnknown())
				}

				var stateAttr types.String
				if diags := resp.Plan.GetAttribute(ctx, path.Root("state"), &stateAttr); diags.HasError() {
					t.Fatalf("unexpected diagnostics reading state: %s", diags)
				}
				if stateAttr.IsUnknown() || stateAttr.ValueString() != "running" {
					t.Errorf("expected state to be pinned to prior state, got %q (unknown=%v)", stateAttr.ValueString(), stateAttr.IsUnknown())
				}

				var ipFilter types.Set
				if diags := resp.Plan.GetAttribute(ctx, ipFilterPath, &ipFilter); diags.HasError() {
					t.Fatalf("unexpected diagnostics reading %s.ip_filter: %s", engine, diags)
				}
				if ipFilter.IsUnknown() || len(ipFilter.Elements()) != 0 {
					t.Errorf("expected %s.ip_filter to resolve to an empty set matching the already-cleared state, got %v", engine, ipFilter)
				}
			})

			t.Run("genuine update leaves recomputed attributes unknown", func(t *testing.T) {
				planAttrs := map[string]tftypes.Value{}
				maps.Copy(planAttrs, baseAttrs)
				// A real config change elsewhere in the resource.
				planAttrs["maintenance_dow"] = tftypes.NewValue(tftypes.String, "tuesday")
				planAttrs["updated_at"] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)

				planValue := buildObject(planAttrs, map[string]tftypes.Value{
					"ip_filter": tftypes.NewValue(ipFilterSetType, tftypes.UnknownValue),
				})
				// Config mirrors the plan: maintenance_dow really changed,
				// ip_filter is still omitted.
				configValue := buildObject(planAttrs, map[string]tftypes.Value{})

				req := resource.ModifyPlanRequest{
					Config: tfsdk.Config{Raw: configValue, Schema: sch},
					State:  state,
					Plan:   tfsdk.Plan{Raw: planValue, Schema: sch},
				}
				resp := &resource.ModifyPlanResponse{Plan: tfsdk.Plan{Raw: planValue, Schema: sch}}

				modifier.ModifyPlan(ctx, req, resp)
				if resp.Diagnostics.HasError() {
					t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
				}

				var updatedAt types.String
				if diags := resp.Plan.GetAttribute(ctx, path.Root("updated_at"), &updatedAt); diags.HasError() {
					t.Fatalf("unexpected diagnostics reading updated_at: %s", diags)
				}
				if !updatedAt.IsUnknown() {
					t.Errorf("expected updated_at to remain unknown during a real update, got %q", updatedAt.ValueString())
				}

				var ipFilter types.Set
				if diags := resp.Plan.GetAttribute(ctx, ipFilterPath, &ipFilter); diags.HasError() {
					t.Fatalf("unexpected diagnostics reading %s.ip_filter: %s", engine, diags)
				}
				if ipFilter.IsUnknown() || len(ipFilter.Elements()) != 0 {
					t.Errorf("expected %s.ip_filter to resolve to an empty set since config omits it, got %v", engine, ipFilter)
				}
			})

			// Regression test for https://github.com/exoscale/terraform-provider-exoscale/pull/563#discussion_r3537167709:
			// when clearing ip_filter is the *only* real change (state has a
			// non-empty ip_filter, config now omits it, nothing else
			// differs), ModifyPlan must not pin it back to the old value -
			// that would silently drop the clear, since Terraform would see
			// zero changes anywhere and never even call Update.
			t.Run("clear-only ip_filter removal is resolved from config, not pinned", func(t *testing.T) {
				planAttrs := map[string]tftypes.Value{}
				maps.Copy(planAttrs, baseAttrs)
				// Nothing else changes; only ip_filter is omitted.
				planAttrs["updated_at"] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)

				planValue := buildObject(planAttrs, map[string]tftypes.Value{
					"ip_filter": tftypes.NewValue(ipFilterSetType, tftypes.UnknownValue),
				})
				configValue := buildObject(planAttrs, map[string]tftypes.Value{})

				req := resource.ModifyPlanRequest{
					Config: tfsdk.Config{Raw: configValue, Schema: sch},
					State:  state,
					Plan:   tfsdk.Plan{Raw: planValue, Schema: sch},
				}
				resp := &resource.ModifyPlanResponse{Plan: tfsdk.Plan{Raw: planValue, Schema: sch}}

				modifier.ModifyPlan(ctx, req, resp)
				if resp.Diagnostics.HasError() {
					t.Fatalf("unexpected diagnostics: %s", resp.Diagnostics)
				}

				var ipFilter types.Set
				if diags := resp.Plan.GetAttribute(ctx, ipFilterPath, &ipFilter); diags.HasError() {
					t.Fatalf("unexpected diagnostics reading %s.ip_filter: %s", engine, diags)
				}
				if ipFilter.IsUnknown() || len(ipFilter.Elements()) != 0 {
					t.Errorf("expected %s.ip_filter to resolve to an empty set (the clear), got %v", engine, ipFilter)
				}

				// Since a real clear is happening, updated_at will really
				// bump - it must not be pinned to the old value either.
				var updatedAt types.String
				if diags := resp.Plan.GetAttribute(ctx, path.Root("updated_at"), &updatedAt); diags.HasError() {
					t.Fatalf("unexpected diagnostics reading updated_at: %s", diags)
				}
				if !updatedAt.IsUnknown() {
					t.Errorf("expected updated_at to remain unknown since clearing %s.ip_filter is a real update, got %q", engine, updatedAt.ValueString())
				}
			})
		})
	}
}
