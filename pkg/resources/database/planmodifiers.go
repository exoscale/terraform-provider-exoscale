package database

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// listRequiresReplaceModifier is a plan modifier that triggers resource
// replacement whenever a list attribute changes.
//
// The terraform-plugin-framework ships a `listplanmodifier.RequiresReplace`
// helper upstream, but it is not vendored in this repository, so we provide a
// minimal equivalent here.
type listRequiresReplaceModifier struct{}

// listRequiresReplace returns a plan modifier that will force resource
// replacement on any change to a list attribute.
func listRequiresReplace() planmodifier.List {
	return listRequiresReplaceModifier{}
}

func (m listRequiresReplaceModifier) Description(_ context.Context) string {
	return "If the value of this attribute changes, Terraform will destroy and recreate the resource."
}

func (m listRequiresReplaceModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m listRequiresReplaceModifier) PlanModifyList(_ context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	// Do not replace on resource creation.
	if req.State.Raw.IsNull() {
		return
	}

	// Do not replace on resource destroy.
	if req.Plan.Raw.IsNull() {
		return
	}

	// Do not replace if the plan and state values are equal.
	if req.PlanValue.Equal(req.StateValue) {
		return
	}

	resp.RequiresReplace = true
}
