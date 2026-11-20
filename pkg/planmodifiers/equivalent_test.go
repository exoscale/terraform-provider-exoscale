package planmodifiers

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEquivalentString(t *testing.T) {
	t.Parallel()

	userData := "#cloud-config\n"
	encoded := base64.StdEncoding.EncodeToString([]byte(userData))

	tests := []struct {
		name     string
		modifier planmodifier.String
		config   types.String
		state    types.String
		want     types.String
	}{
		{"case only", IgnoreCase(), types.StringValue("Standard.Medium"), types.StringValue("standard.medium"), types.StringValue("standard.medium")},
		{"different value", IgnoreCase(), types.StringValue("standard.large"), types.StringValue("standard.medium"), types.StringValue("standard.large")},
		{"null state", IgnoreCase(), types.StringValue("Standard.Medium"), types.StringNull(), types.StringValue("Standard.Medium")},
		{"encoded user data", IgnoreUserDataEncoding(), types.StringValue(encoded), types.StringValue(userData), types.StringValue(userData)},
		{"changed user data", IgnoreUserDataEncoding(), types.StringValue("#cloud-config\nusers: []\n"), types.StringValue(userData), types.StringValue("#cloud-config\nusers: []\n")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := planmodifier.StringRequest{ConfigValue: tt.config, StateValue: tt.state, PlanValue: tt.config}
			resp := &planmodifier.StringResponse{PlanValue: tt.config}
			tt.modifier.PlanModifyString(context.Background(), req, resp)

			if !resp.PlanValue.Equal(tt.want) {
				t.Errorf("got %s, want %s", resp.PlanValue, tt.want)
			}
		})
	}
}
