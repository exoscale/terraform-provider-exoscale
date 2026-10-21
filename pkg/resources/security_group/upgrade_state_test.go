package security_group

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
)

func TestUpgradeStateV0(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	a := require.New(t)
	r := &Resource{}

	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	a.False(schemaResp.Diagnostics.HasError())

	upgrader := r.UpgradeState(ctx)[0]

	prior := tfsdk.State{
		Schema: *upgrader.PriorSchema,
		Raw:    tftypes.NewValue(upgrader.PriorSchema.Type().TerraformType(ctx), nil),
	}
	for attr, value := range map[string]any{
		"id":               "8a5f6f2e-0000-4000-8000-000000000001",
		"name":             "sg",
		"description":      "test",
		"external_sources": []string{"10.0.0.0/8"},
	} {
		a.False(prior.SetAttribute(ctx, path.Root(attr), value).HasError(), attr)
	}

	resp := resource.UpgradeStateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
		},
	}

	upgrader.StateUpgrader(ctx, resource.UpgradeStateRequest{State: &prior}, &resp)
	a.False(resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

	var state ResourceModel
	a.False(resp.State.Get(ctx, &state).HasError())
	a.Equal(types.StringValue("8a5f6f2e-0000-4000-8000-000000000001"), state.ID)
	a.Equal(types.StringValue("sg"), state.Name)
	a.Equal(types.StringValue("test"), state.Description)
	a.Equal(types.SetValueMust(types.StringType, []attr.Value{types.StringValue("10.0.0.0/8")}), state.ExternalSources)
}
