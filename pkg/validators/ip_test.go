package validators

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestIsIPv4Address(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		value   types.String
		wantErr bool
	}{
		{"ipv4", types.StringValue("10.0.0.1"), false},
		{"null", types.StringNull(), false},
		{"unknown", types.StringUnknown(), false},
		{"ipv6", types.StringValue("2001:db8::1"), true},
		{"not an ip", types.StringValue("10.0.0"), true},
		{"empty", types.StringValue(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resp := &validator.StringResponse{}
			IsIPv4Address().ValidateString(context.Background(), validator.StringRequest{ConfigValue: tt.value}, resp)

			if got := resp.Diagnostics.HasError(); got != tt.wantErr {
				t.Errorf("got error %t, want %t: %v", got, tt.wantErr, resp.Diagnostics)
			}
		})
	}
}
