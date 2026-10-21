package instance

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

// The SDKv2 implementation of this resource made no difference between an
// unset attribute and an empty one, and its Read wrote "" / {} in the state
// / [] for attributes missing from the configuration. The plan modifiers below keep
// that state loading without a diff, and keep explicitly empty values
// (`reverse_dns = ""`, `labels = {}`) from producing one either.

// emptyStringIsNull plans the prior value of an Optional+Computed string
// missing from the configuration as long as that value is empty, and null
// (the attribute was removed from the configuration) otherwise.
type emptyStringIsNull struct{}

func (emptyStringIsNull) Description(context.Context) string {
	return "An empty string is equivalent to an unset value."
}

func (m emptyStringIsNull) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (emptyStringIsNull) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	if req.StateValue.ValueString() == "" {
		resp.PlanValue = req.StateValue
		return
	}

	resp.PlanValue = types.StringNull()
}

// emptyMapIsNull is emptyStringIsNull for maps.
type emptyMapIsNull struct{}

func (emptyMapIsNull) Description(context.Context) string {
	return "An empty map is equivalent to an unset value."
}

func (m emptyMapIsNull) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (emptyMapIsNull) PlanModifyMap(_ context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	if len(req.StateValue.Elements()) == 0 {
		resp.PlanValue = req.StateValue
		return
	}

	resp.PlanValue = types.MapNull(types.StringType)
}

// emptySetIsNull is emptyStringIsNull for sets of strings.
type emptySetIsNull struct{}

func (emptySetIsNull) Description(context.Context) string {
	return "An empty set is equivalent to an unset value."
}

func (m emptySetIsNull) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (emptySetIsNull) PlanModifySet(_ context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	if len(req.StateValue.Elements()) == 0 {
		resp.PlanValue = req.StateValue
		return
	}

	resp.PlanValue = types.SetNull(types.StringType)
}

// equivalentString plans the prior value of a string when the configured one
// is equivalent to it, which is what DiffSuppressFunc did with the SDKv2.
type equivalentString struct {
	description string
	equivalent  func(state, config string) bool
}

func (m equivalentString) Description(context.Context) string {
	return m.description
}

func (m equivalentString) MarkdownDescription(context.Context) string {
	return m.description
}

func (m equivalentString) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.StateValue.IsNull() {
		return
	}

	if m.equivalent(req.StateValue.ValueString(), req.ConfigValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}

// ignoreCase ignores case differences.
func ignoreCase() planmodifier.String {
	return equivalentString{
		description: "Case differences are ignored.",
		equivalent:  strings.EqualFold,
	}
}

// ignoreUserDataEncoding ignores the differences caused by the provider
// decoding user_data on read when it was supplied pre-encoded (base64 or
// gzip+base64).
func ignoreUserDataEncoding() planmodifier.String {
	return equivalentString{
		description: "Encoding differences are ignored.",
		equivalent:  sameUserData,
	}
}

// sameUserData tells whether two user data hold the same content, whatever
// their encoding.
func sameUserData(a, b string) bool {
	return normalizeUserData(a) == normalizeUserData(b)
}

func normalizeUserData(v string) string {
	normalized, err := utils.DecodeUserData(v)
	if err != nil {
		return v
	}

	return normalized
}

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
