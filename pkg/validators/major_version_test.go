package validators_test

import (
"context"
"testing"

"github.com/hashicorp/terraform-plugin-framework/path"
"github.com/hashicorp/terraform-plugin-framework/schema/validator"
"github.com/hashicorp/terraform-plugin-framework/types"

exovalidators "github.com/exoscale/terraform-provider-exoscale/pkg/validators"
)

func TestIsMajorVersionValidator(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"major only", "2", false},
		{"major only pg", "16", false},
		{"major.minor rejected", "2.0", true},
		{"major.minor.patch rejected", "16.5.1", true},
		{"semver rejected", "3.3.0", true},
		{"empty string rejected", "", true},
		{"non-numeric rejected", "latest", true},
	}

	v := exovalidators.IsMajorVersionValidator

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
t.Parallel()

			req := validator.StringRequest{
				Path:        path.Root("version"),
				ConfigValue: types.StringValue(tc.input),
			}
			resp := &validator.StringResponse{}
			v.ValidateString(context.Background(), req, resp)

			if tc.wantErr && !resp.Diagnostics.HasError() {
				t.Errorf("input %q: expected validation error but got none", tc.input)
			}
			if !tc.wantErr && resp.Diagnostics.HasError() {
				t.Errorf("input %q: unexpected validation error: %s", tc.input, resp.Diagnostics)
			}
		})
	}
}
