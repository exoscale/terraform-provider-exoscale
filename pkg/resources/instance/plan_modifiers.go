package instance

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// stringRequiresReplace replaces the instance when the string changes, null
// and "" being the same value.
func stringRequiresReplace() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			resp.RequiresReplace = req.PlanValue.IsUnknown() ||
				req.PlanValue.ValueString() != req.StateValue.ValueString()
		},
		"Changing this attribute replaces the instance.",
		"Changing this attribute replaces the instance.",
	)
}

// setRequiresReplace replaces the instance when the set changes, null and
// empty being the same value.
func setRequiresReplace() planmodifier.Set {
	return setplanmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.SetRequest, resp *setplanmodifier.RequiresReplaceIfFuncResponse) {
			resp.RequiresReplace = req.PlanValue.IsUnknown() ||
				len(req.PlanValue.Elements()) > 0 ||
				len(req.StateValue.Elements()) > 0
		},
		"Changing this attribute replaces the instance.",
		"Changing this attribute replaces the instance.",
	)
}

// stringFuncValidator validates a string with a function.
type stringFuncValidator struct {
	description string
	validate    func(string) error
}

func (v stringFuncValidator) Description(context.Context) string {
	return v.description
}

func (v stringFuncValidator) MarkdownDescription(context.Context) string {
	return v.description
}

func (v stringFuncValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	if err := v.validate(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "invalid value", err.Error())
	}
}

// deprecatedSetBlockValidator warns when a set block is configured at least
// once. Use it instead of DeprecationMessage on set blocks: Terraform sends an
// empty set (not null) when the block is absent, so DeprecationMessage would
// warn on every resource, even those not using the block.
type deprecatedSetBlockValidator struct {
	message string
}

func (v deprecatedSetBlockValidator) Description(context.Context) string {
	return v.message
}

func (v deprecatedSetBlockValidator) MarkdownDescription(context.Context) string {
	return v.message
}

func (v deprecatedSetBlockValidator) ValidateSet(_ context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || len(req.ConfigValue.Elements()) == 0 {
		return
	}

	resp.Diagnostics.AddAttributeWarning(req.Path, "Block Deprecated", v.message)
}
