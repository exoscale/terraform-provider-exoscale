// Package planmodifiers holds the plan modifiers shared by the resources
// migrated from the SDKv2 to the plugin framework.
package planmodifiers

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The SDKv2 made no difference between an unset attribute and an empty one,
// and its Read wrote "" / {} / [] in the state for Optional+Computed
// attributes missing from the configuration. The plan modifiers below keep
// that state loading without a diff, and keep explicitly empty values
// (`reverse_dns = ""`, `labels = {}`) from producing one either.

// EmptyStringIsNull plans the prior value of an Optional+Computed string
// missing from the configuration as long as that value is empty, and null
// (the attribute was removed from the configuration) otherwise.
func EmptyStringIsNull() planmodifier.String {
	return emptyStringIsNull{}
}

// EmptyMapIsNull is EmptyStringIsNull for maps.
func EmptyMapIsNull() planmodifier.Map {
	return emptyMapIsNull{}
}

// EmptySetIsNull is EmptyStringIsNull for sets.
func EmptySetIsNull() planmodifier.Set {
	return emptySetIsNull{}
}

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

type emptyMapIsNull struct{}

func (emptyMapIsNull) Description(context.Context) string {
	return "An empty map is equivalent to an unset value."
}

func (m emptyMapIsNull) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (emptyMapIsNull) PlanModifyMap(ctx context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	if len(req.StateValue.Elements()) == 0 {
		resp.PlanValue = req.StateValue
		return
	}

	resp.PlanValue = types.MapNull(req.PlanValue.ElementType(ctx))
}

type emptySetIsNull struct{}

func (emptySetIsNull) Description(context.Context) string {
	return "An empty set is equivalent to an unset value."
}

func (m emptySetIsNull) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (emptySetIsNull) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}

	if len(req.StateValue.Elements()) == 0 {
		resp.PlanValue = req.StateValue
		return
	}

	resp.PlanValue = types.SetNull(req.PlanValue.ElementType(ctx))
}
