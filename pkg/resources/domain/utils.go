package domain

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"golang.org/x/net/idna"
)

// domainNameToUnicode converts an ACE/punycode domain name to its Unicode
// representation. If the name is already Unicode, or conversion fails, the
// original value is returned unchanged. This is used to suppress spurious
// plan diffs when users specify a punycode name but the API returns unicode.
func domainNameToUnicode(name string) string {
	unicode, err := idna.ToUnicode(name)
	if err != nil {
		return name
	}
	return unicode
}

// domainNameUnicodePlanModifier suppresses plan diffs between the punycode
// and unicode forms of a domain name: the API always returns the unicode
// form, so a config using ACE/punycode would otherwise produce a perpetual
// diff (and, combined with stringplanmodifier.RequiresReplace, a perpetual
// forced replacement).
type domainNameUnicodePlanModifier struct{}

func (m domainNameUnicodePlanModifier) Description(_ context.Context) string {
	return "Suppresses diffs between punycode and unicode forms of the domain name."
}

func (m domainNameUnicodePlanModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m domainNameUnicodePlanModifier) PlanModifyString(
	_ context.Context,
	req planmodifier.StringRequest,
	resp *planmodifier.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	// Always normalize to the unicode form: the API only ever returns unicode
	// names, so the planned value must match it for the "no-op apply" and
	// "create with punycode input" cases to produce a consistent result.
	resp.PlanValue = types.StringValue(domainNameToUnicode(req.ConfigValue.ValueString()))
}
