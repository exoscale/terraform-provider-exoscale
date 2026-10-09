package instance_pool

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"

	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

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
