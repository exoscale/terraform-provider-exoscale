package validators

import (
	"context"
	"fmt"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type IsCIDRNetworkValidator struct {
	Max int
	Min int
}

// Description returns a plain text description of the validator's behavior, suitable for a practitioner to understand its impact.
func (v IsCIDRNetworkValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("string must be a valid network notation with significant bits between min and max (inclusive) %d and %d", v.Min, v.Max)
}

// MarkdownDescription returns a markdown formatted description of the validator's behavior, suitable for a practitioner to understand its impact.
func (v IsCIDRNetworkValidator) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("string must be a valid network notation with significant bits between min and max (inclusive) %d and %d", v.Min, v.Max)
}

// Validate runs the main validation logic of the validator, reading configuration data out of `req` and updating `resp` with diagnostics.
func (v IsCIDRNetworkValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// If the value is unknown or null, there is nothing to validate.
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	value := req.ConfigValue.ValueString()

	_, ipnet, err := net.ParseCIDR(value)
	if err != nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid String Format",
			fmt.Sprintf("Cannot parse string as CIDR notation: %v", err),
		)

		return
	}

	if ipnet == nil || value != ipnet.String() {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Network Value",
			fmt.Sprintf("expected a valid network value %s, got %s", ipnet, value),
		)

		return
	}

	sigbits, _ := ipnet.Mask.Size()
	if sigbits < v.Min || sigbits > v.Max {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Network Value",
			fmt.Sprintf("expected network value between %d and %d significant bits, got: %d", v.Min, v.Max, sigbits),
		)

		return
	}
}
