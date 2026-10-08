package iam

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The SDKv2 implementation of exoscale_iam_access_key made no difference
// between an unset set and an empty one. The plan modifiers below keep the
// state it wrote loading without a diff (same as the instance ones).

// emptySetIsNull plans the prior value of an Optional+Computed set of strings
// missing from the configuration as long as that value is empty, and null
// (the attribute was removed from the configuration) otherwise.
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

// setRequiresReplace replaces the access key when the set changes, null and
// empty being the same value.
func setRequiresReplace() planmodifier.Set {
	return setplanmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.SetRequest, resp *setplanmodifier.RequiresReplaceIfFuncResponse) {
			resp.RequiresReplace = req.PlanValue.IsUnknown() ||
				len(req.PlanValue.Elements()) > 0 ||
				len(req.StateValue.Elements()) > 0
		},
		"Changing this attribute replaces the access key.",
		"Changing this attribute replaces the access key.",
	)
}

// operationsFromTags plans the prior operations when they are the configured
// ones plus the ones granted by the tags: the API returns both, which is what
// the SDKv2 DiffSuppressFunc ignored.
type operationsFromTagsModifier struct{}

func operationsFromTags() planmodifier.Set {
	return operationsFromTagsModifier{}
}

func (operationsFromTagsModifier) Description(context.Context) string {
	return "Operations granted by the tags are ignored."
}

func (m operationsFromTagsModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (operationsFromTagsModifier) PlanModifySet(ctx context.Context, req planmodifier.SetRequest, resp *planmodifier.SetResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() || req.StateValue.IsNull() {
		return
	}
	for _, element := range req.ConfigValue.Elements() {
		if element.IsUnknown() {
			return
		}
	}

	var tagsOperations types.Set
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("tags_operations"), &tagsOperations)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var configured, fromTags, current []string
	resp.Diagnostics.Append(req.ConfigValue.ElementsAs(ctx, &configured, false)...)
	resp.Diagnostics.Append(tagsOperations.ElementsAs(ctx, &fromTags, false)...)
	resp.Diagnostics.Append(req.StateValue.ElementsAs(ctx, &current, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	expected := map[string]bool{}
	for _, operation := range configured {
		expected[operation] = true
	}
	for _, operation := range fromTags {
		expected[operation] = true
	}

	if len(current) != len(expected) {
		return
	}
	for _, operation := range current {
		if !expected[operation] {
			return
		}
	}

	resp.PlanValue = req.StateValue
}
