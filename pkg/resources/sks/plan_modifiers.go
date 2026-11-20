package sks

import (
	"context"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// nodepoolTemplateSelection plans a nodepool "template_selection" from its
// configuration: pinned when "template_id" is set, auto otherwise.
type nodepoolTemplateSelection struct{}

func (nodepoolTemplateSelection) Description(context.Context) string {
	return "The template is pinned when template_id is set, and follows the latest template otherwise."
}

func (m nodepoolTemplateSelection) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (nodepoolTemplateSelection) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	var templateID types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("template_id"), &templateID)...)

	selection := exoscale.SKSNodepoolTemplateSelectionPinned
	if templateID.IsNull() {
		selection = exoscale.SKSNodepoolTemplateSelectionAuto
	}
	resp.PlanValue = types.StringValue(string(selection))
}

// nodepoolTemplateIDUnpinned plans an unknown "template_id" when it is removed
// from the configuration of a pinned nodepool: the nodepool then gets the
// latest template, which only the API knows.
type nodepoolTemplateIDUnpinned struct{}

func (nodepoolTemplateIDUnpinned) Description(context.Context) string {
	return "Removing template_id unpins the template."
}

func (m nodepoolTemplateIDUnpinned) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (nodepoolTemplateIDUnpinned) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if !req.ConfigValue.IsNull() || req.State.Raw.IsNull() {
		return
	}

	var selection types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("template_selection"), &selection)...)
	if selection.ValueString() == string(exoscale.SKSNodepoolTemplateSelectionPinned) {
		resp.PlanValue = types.StringUnknown()
	}
}
