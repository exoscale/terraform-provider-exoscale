package validators

import (
	"context"
	"fmt"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = (*IsIPAddressValidator)(nil)

type IsIPAddressValidator struct{}

func IsIPAddress() validator.String {
	return IsIPAddressValidator{}
}

func (v IsIPAddressValidator) Description(_ context.Context) string {
	return "Value must be a valid IP address."
}

func (v IsIPAddressValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v IsIPAddressValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if ip := net.ParseIP(req.ConfigValue.ValueString()); ip == nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid IP Address",
			"Value must be a valid IP address, got: "+req.ConfigValue.ValueString(),
		)
	}
}

var _ validator.String = (*IsIPv4AddressValidator)(nil)

type IsIPv4AddressValidator struct{}

func IsIPv4Address() validator.String {
	return IsIPv4AddressValidator{}
}

func (v IsIPv4AddressValidator) Description(_ context.Context) string {
	return "Value must be a valid IPv4 address."
}

func (v IsIPv4AddressValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v IsIPv4AddressValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if ip := net.ParseIP(req.ConfigValue.ValueString()); ip == nil || ip.To4() == nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid IPv4 Address",
			fmt.Sprintf("Value must be a valid IPv4 address, got: %q", req.ConfigValue.ValueString()),
		)
	}
}

type IsNetmaskValidator struct{}

func IsNetmask() validator.String {
	return IsNetmaskValidator{}
}

func (v IsNetmaskValidator) Description(_ context.Context) string {
	return "Value must be a valid IP netmask (e.g. 255.255.255.0)."
}

func (v IsNetmaskValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v IsNetmaskValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	val := req.ConfigValue.ValueString()

	ip := net.ParseIP(val)
	if ip == nil {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Netmask",
			fmt.Sprintf("Value must be a valid netmask, got: %q", val),
		)
		return
	}

	var mask net.IPMask
	if v4 := ip.To4(); v4 != nil {
		mask = net.IPMask(v4)
	} else {
		mask = net.IPMask(ip.To16())
	}

	if ones, bits := mask.Size(); ones == 0 && bits == 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid Netmask",
			fmt.Sprintf("Value is not a valid netmask (bits must be contiguous), got: %q", val),
		)
	}
}
