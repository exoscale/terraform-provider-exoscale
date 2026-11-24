package instance_pool

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
)

func resourceSchema(t *testing.T) schema.Schema {
	t.Helper()

	var resp resource.SchemaResponse
	(&Resource{}).Schema(context.Background(), resource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	return resp.Schema
}

func TestUpgradeStateV0Private(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	a := require.New(t)
	s := resourceSchema(t)
	upgrader := (&Resource{}).UpgradeState(ctx)[0]

	prior := tfsdk.State{
		Schema: *upgrader.PriorSchema,
		Raw:    tftypes.NewValue(upgrader.PriorSchema.Type().TerraformType(ctx), nil),
	}
	for attr, value := range map[string]any{
		"id":               "8a5f6f2e-0000-4000-8000-000000000001",
		"name":             "pool",
		"zone":             "ch-gva-2",
		"template_id":      "8a5f6f2e-0000-4000-8000-000000000002",
		"size":             int64(1),
		"ipv6":             false,
		"service_offering": "standard.tiny",
	} {
		a.False(prior.SetAttribute(ctx, path.Root(attr), value).HasError(), attr)
	}

	req := resource.UpgradeStateRequest{State: &prior}
	resp := resource.UpgradeStateResponse{
		State: tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(ctx), nil)},
	}

	upgrader.StateUpgrader(ctx, req, &resp)
	a.False(resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var private types.Bool
	a.False(resp.State.GetAttribute(ctx, path.Root("private"), &private).HasError())
	a.Equal(types.BoolValue(false), private)
}

func TestPrivateRequiresReplace(t *testing.T) {
	t.Parallel()

	modifiers := resourceSchema(t).Attributes["private"].(schema.BoolAttribute).PlanModifiers

	for name, tc := range map[string]struct {
		state, plan types.Bool
		expected    bool
	}{
		// A pool created before private existed, planned with -refresh=false.
		"null to false": {state: types.BoolNull(), plan: types.BoolValue(false), expected: false},
		"null to true":  {state: types.BoolNull(), plan: types.BoolValue(true), expected: true},
		"false to true": {state: types.BoolValue(false), plan: types.BoolValue(true), expected: true},
		"true to false": {state: types.BoolValue(true), plan: types.BoolValue(false), expected: true},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			req := planmodifier.BoolRequest{
				Path:        path.Root("private"),
				StateValue:  tc.state,
				PlanValue:   tc.plan,
				ConfigValue: tc.plan,
				// RequiresReplaceIf skips resource creation and destruction.
				State: tfsdk.State{Raw: tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})},
				Plan:  tfsdk.Plan{Raw: tftypes.NewValue(tftypes.Object{}, map[string]tftypes.Value{})},
			}
			resp := &planmodifier.BoolResponse{PlanValue: tc.plan}

			for _, m := range modifiers {
				m.PlanModifyBool(ctx, req, resp)
			}

			require.False(t, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
			require.Equal(t, tc.expected, resp.RequiresReplace)
		})
	}
}
