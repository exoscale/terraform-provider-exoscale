package instance_pool

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
)

// The forked SDKv2 implementation wrote private in the v0 state, upstream did
// not: both must upgrade.
func TestUpgradeStateV0Private(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		raw      string
		expected bool
	}{
		"fork private": {
			raw:      `{"id":"8a5f6f2e-0000-4000-8000-000000000001","name":"pool","zone":"de-fra-1","template_id":"8a5f6f2e-0000-4000-8000-000000000002","size":1,"ipv6":false,"private":true,"description":"","labels":{},"service_offering":"standard.tiny"}`,
			expected: true,
		},
		"fork public": {
			raw:      `{"id":"8a5f6f2e-0000-4000-8000-000000000001","name":"pool","zone":"de-fra-1","template_id":"8a5f6f2e-0000-4000-8000-000000000002","size":1,"ipv6":true,"private":false}`,
			expected: false,
		},
		"upstream": {
			raw:      `{"id":"8a5f6f2e-0000-4000-8000-000000000001","name":"pool","zone":"de-fra-1","template_id":"8a5f6f2e-0000-4000-8000-000000000002","size":1,"ipv6":false}`,
			expected: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			a := require.New(t)
			r := &Resource{}

			var schemaResp resource.SchemaResponse
			r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
			a.False(schemaResp.Diagnostics.HasError())

			upgrader := r.UpgradeState(ctx)[0]

			prior, err := tftypes.ValueFromJSON([]byte(tc.raw), upgrader.PriorSchema.Type().TerraformType(ctx))
			a.NoError(err)

			req := resource.UpgradeStateRequest{
				State: &tfsdk.State{Schema: *upgrader.PriorSchema, Raw: prior},
			}
			resp := resource.UpgradeStateResponse{
				State: tfsdk.State{
					Schema: schemaResp.Schema,
					Raw:    tftypes.NewValue(schemaResp.Schema.Type().TerraformType(ctx), nil),
				},
			}

			upgrader.StateUpgrader(ctx, req, &resp)
			a.False(resp.Diagnostics.HasError(), "%v", resp.Diagnostics)

			var private types.Bool
			a.False(resp.State.GetAttribute(ctx, path.Root("private"), &private).HasError())
			a.Equal(types.BoolValue(tc.expected), private)
		})
	}
}
