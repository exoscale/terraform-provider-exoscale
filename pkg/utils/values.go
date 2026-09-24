// Refresh helpers should be used in Read to update state as they solve unintended drift
// As per official terraform-framework recommendations:
// We must preserve the prior state value if the updated value is semantically equal.
// This prevents Terraform from showing extraneous drift in plans.
// Globally applied rules are:
// - unknown value in state must be replaced.
//
// NOTE: Refresh helpers provide no protection against a nil pointer state argument.
// This is unlikelly to happen by accident and panic is considered appropriate.
package utils

import (
	"context"
	"net"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// RefreshString helper to update state value.
//
// Empty string on remote is equal to empty string and nil in state.
func RefreshString(state *basetypes.StringValue, remote string) {
	if state.IsUnknown() {
		*state = types.StringValue(remote)
	}

	if (state.IsNull() || state.ValueString() == "") && remote == "" {
		return
	}

	*state = types.StringValue(remote)
}

// RefreshStringPointer helper to update state value.
//
// Unset and empty string on remote is equal to empty string and nil in state.
func RefreshStringPointer(state *basetypes.StringValue, remote *string) {
	if state.IsUnknown() {
		*state = types.StringPointerValue(remote)
	}

	if remote == nil {
		return
	}

	RefreshString(state, *remote)
}

// RefreshIP helper to update net.IP state value.
//
// Works as RefreshString but further suppress literal '<nil>' output by net library.
func RefreshIP(state *basetypes.StringValue, remote net.IP) {
	var value string
	if len(remote) > 0 {
		value = remote.String()
	}

	RefreshString(state, value)
}

// RefreshLabels helper to update state value.
//
// Unset and empty map on remote is equal to empty map and nil in state.
func RefreshLabels(
	ctx context.Context,
	state *basetypes.MapValue,
	labels map[string]string,
) (dg diag.Diagnostics) {
	if len(labels) == 0 {
		if state.IsUnknown() {
			*state = types.MapNull(types.StringType)
		}
		return
	}

	var t basetypes.MapValue
	t, dg = types.MapValueFrom(
		ctx,
		types.StringType,
		labels,
	)
	if dg.HasError() {
		return
	}

	*state = t

	return
}
