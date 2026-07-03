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
// directly, without TF_ACC or any API calls, to guard against the two
// regressions this hook fixes:
//   - a purely Computed attribute like updated_at defaults to unknown on
//     every plan, even a genuine no-op, unless something pins it back;
//   - naively pinning it (stringplanmodifier.UseStateForUnknown) instead
//     breaks once a real update legitimately changes the value, and also
//     defeats the per-engine Update logic that relies on ip_filter going
//     unknown to detect "omitted from config, clear it".
//
// It's run once per engine block (mysql, pg, opensearch, kafka, grafana),
// since ModifyPlan has a separate, hand-written branch neutralizing
// ip_filter for each one.
func TestServiceResourceModifyPlan(t *testing.T) {
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

			stateValue := buildObject(baseAttrs, map[string]tftypes.Value{
				"ip_filter": tftypes.NewValue(ipFilterSetType, []tftypes.Value{
					tftypes.NewValue(tftypes.String, "1.2.3.4/32"),
				}),
			})
			state := tfsdk.State{Raw: stateValue, Schema: sch}

			ipFilterPath := path.Root(engine).AtName("ip_filter")

			t.Run("no-op plan pins recomputed attributes back to state", func(t *testing.T) {
				planAttrs := map[string]tftypes.Value{}
				for k, v := range baseAttrs {
					planAttrs[k] = v
				}
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

				req := resource.ModifyPlanRequest{
					State: state,
					Plan:  tfsdk.Plan{Raw: planValue, Schema: sch},
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
				if ipFilter.IsUnknown() {
					t.Errorf("expected %s.ip_filter to be pinned to prior state, got unknown", engine)
				}
			})

			t.Run("genuine update leaves recomputed attributes unknown", func(t *testing.T) {
				planAttrs := map[string]tftypes.Value{}
				for k, v := range baseAttrs {
					planAttrs[k] = v
				}
				// A real config change elsewhere in the resource.
				planAttrs["maintenance_dow"] = tftypes.NewValue(tftypes.String, "tuesday")
				planAttrs["updated_at"] = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)

				planValue := buildObject(planAttrs, map[string]tftypes.Value{
					"ip_filter": tftypes.NewValue(ipFilterSetType, tftypes.UnknownValue),
				})

				req := resource.ModifyPlanRequest{
					State: state,
					Plan:  tfsdk.Plan{Raw: planValue, Schema: sch},
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
				if !ipFilter.IsUnknown() {
					t.Errorf("expected %s.ip_filter to remain unknown so the Update logic still treats an omitted ip_filter as cleared", engine)
				}
			})
		})
	}
}
