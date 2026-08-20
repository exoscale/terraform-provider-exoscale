package domain

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestDomainNameToUnicode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"example.com", "example.com"},
		{"xn--n3h.ws", "☃.ws"},
		{"xn--domain-with--rcb.ch", "domain-with-ä.ch"},
		{"already-unicodeä.com", "already-unicodeä.com"},
		{"", ""},
	}

	for _, tt := range tests {
		got := domainNameToUnicode(tt.input)
		if got != tt.want {
			t.Errorf("domainNameToUnicode(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDomainNameUnicodePlanModifier(t *testing.T) {
	t.Parallel()

	m := domainNameUnicodePlanModifier{}

	tests := []struct {
		name       string
		state      types.String
		config     types.String
		wantPlan   types.String
		wantChange bool
	}{
		{
			name:       "punycode config normalizes to unicode",
			state:      types.StringNull(),
			config:     types.StringValue("xn--n3h.ws"),
			wantPlan:   types.StringValue("☃.ws"),
			wantChange: true,
		},
		{
			name:       "unicode config left untouched",
			state:      types.StringValue("☃.ws"),
			config:     types.StringValue("☃.ws"),
			wantPlan:   types.StringValue("☃.ws"),
			wantChange: true,
		},
		{
			name:       "unknown config is left alone",
			config:     types.StringUnknown(),
			wantChange: false,
		},
		{
			name:       "null config is left alone",
			config:     types.StringNull(),
			wantChange: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := planmodifier.StringRequest{
				StateValue:  tt.state,
				ConfigValue: tt.config,
			}
			resp := &planmodifier.StringResponse{PlanValue: tt.config}

			m.PlanModifyString(context.Background(), req, resp)

			if tt.wantChange && !resp.PlanValue.Equal(tt.wantPlan) {
				t.Errorf("PlanModifyString() PlanValue = %v, want %v", resp.PlanValue, tt.wantPlan)
			}
			if !tt.wantChange && !resp.PlanValue.Equal(tt.config) {
				t.Errorf("PlanModifyString() unexpectedly changed PlanValue to %v", resp.PlanValue)
			}
		})
	}
}
