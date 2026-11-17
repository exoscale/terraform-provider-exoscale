package planmodifiers

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEmptyStringIsNull(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config types.String
		state  types.String
		plan   types.String
		want   types.String
	}{
		{"configured", types.StringValue("a"), types.StringValue(""), types.StringValue("a"), types.StringValue("a")},
		{"legacy empty state", types.StringNull(), types.StringValue(""), types.StringUnknown(), types.StringValue("")},
		{"null state", types.StringNull(), types.StringNull(), types.StringUnknown(), types.StringNull()},
		{"removed from config", types.StringNull(), types.StringValue("a"), types.StringUnknown(), types.StringNull()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := planmodifier.StringRequest{ConfigValue: tt.config, StateValue: tt.state, PlanValue: tt.plan}
			resp := &planmodifier.StringResponse{PlanValue: tt.plan}
			EmptyStringIsNull().PlanModifyString(context.Background(), req, resp)

			if !resp.PlanValue.Equal(tt.want) {
				t.Errorf("got %s, want %s", resp.PlanValue, tt.want)
			}
		})
	}
}

func TestEmptyMapIsNull(t *testing.T) {
	t.Parallel()

	var (
		empty    = types.MapValueMust(types.StringType, map[string]attr.Value{})
		nonEmpty = types.MapValueMust(types.StringType, map[string]attr.Value{"k": types.StringValue("v")})
		unknown  = types.MapUnknown(types.StringType)
		null     = types.MapNull(types.StringType)
	)

	tests := []struct {
		name   string
		config types.Map
		state  types.Map
		want   types.Map
	}{
		{"configured", nonEmpty, empty, nonEmpty},
		{"legacy empty state", null, empty, empty},
		{"removed from config", null, nonEmpty, null},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plan := tt.config
			if tt.config.IsNull() {
				plan = unknown
			}

			req := planmodifier.MapRequest{ConfigValue: tt.config, StateValue: tt.state, PlanValue: plan}
			resp := &planmodifier.MapResponse{PlanValue: plan}
			EmptyMapIsNull().PlanModifyMap(context.Background(), req, resp)

			if !resp.PlanValue.Equal(tt.want) {
				t.Errorf("got %s, want %s", resp.PlanValue, tt.want)
			}
		})
	}
}

func TestEmptySetIsNull(t *testing.T) {
	t.Parallel()

	var (
		empty    = types.SetValueMust(types.StringType, []attr.Value{})
		nonEmpty = types.SetValueMust(types.StringType, []attr.Value{types.StringValue("a")})
		unknown  = types.SetUnknown(types.StringType)
		null     = types.SetNull(types.StringType)
	)

	tests := []struct {
		name   string
		config types.Set
		state  types.Set
		want   types.Set
	}{
		{"configured", nonEmpty, empty, nonEmpty},
		{"legacy empty state", null, empty, empty},
		{"removed from config", null, nonEmpty, null},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plan := tt.config
			if tt.config.IsNull() {
				plan = unknown
			}

			req := planmodifier.SetRequest{ConfigValue: tt.config, StateValue: tt.state, PlanValue: plan}
			resp := &planmodifier.SetResponse{PlanValue: plan}
			EmptySetIsNull().PlanModifySet(context.Background(), req, resp)

			if !resp.PlanValue.Equal(tt.want) {
				t.Errorf("got %s, want %s", resp.PlanValue, tt.want)
			}
		})
	}
}

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
