package security_group

import (
	"context"
	"errors"
	"fmt"
	"slices"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
)

const ResourceDescription = `Manage [Exoscale Security Groups](https://community.exoscale.com/product/compute/instances/quick-start/#firewall-rules---security-groups).

Security Groups are firewall rules. Each Security Group may be attached to one or many compute instances.

Individual firewall rules are configured using linked resource: [exoscale_security_group_rule](./security_group_rule.md).

Corresponding data source: [exoscale_security_group](../data_sources/security_group.md).`

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &Resource{}
var _ resource.ResourceWithImportState = &Resource{}

type Resource struct {
	client *exoscale.Client
}

// NewResource creates instance of Resource.
func NewResource() resource.Resource {
	return &Resource{}
}

func NewResourceModel() ResourceModel {
	return ResourceModel{
		ID:              types.StringNull(),
		Name:            types.StringNull(),
		Description:     types.StringNull(),
		ExternalSources: types.SetNull(types.StringType),
		Timeouts:        timeouts.Value{},
	}
}

// ResourceModel defines the resource data model.
type ResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	ExternalSources types.Set    `tfsdk:"external_sources"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// Metadata specifies resource name.
func (r *Resource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_security_group"
}

// Schema defines resource attributes.
func (r *Resource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		Description:         "Manage Security Groups",
		MarkdownDescription: ResourceDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "security group ID",
				MarkdownDescription: "The ID of the Security Group.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description:         "security group name",
				MarkdownDescription: "❗ The name of the Security Group.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description:         "security group description",
				MarkdownDescription: "❗ A free-form text describing the the Security Group.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"external_sources": schema.SetAttribute{
				ElementType:         types.StringType,
				Description:         "external network sources",
				MarkdownDescription: `A list of external network sources, in [CIDR](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing#CIDR_notation) notation.`,
				Optional:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

// Configure sets up resource dependencies.
func (r *Resource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

// Create resources by receiving Terraform configuration and plan data, performing creation logic, and saving Terraform state data.
func (r *Resource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan ResourceModel
	state := NewResourceModel()

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Timeouts = plan.Timeouts

	timeout, diags := plan.Timeouts.Create(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	sg := exoscale.CreateSecurityGroupRequest{
		Name: plan.Name.ValueString(),
	}
	if !plan.Description.IsNull() {
		sg.Description = plan.Description.ValueString()
	}

	op, err := r.client.CreateSecurityGroup(
		ctx,
		sg,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"API returned error when creating security group",
			err.Error(),
		)
		return
	}

	if _, err := r.client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError(
			"create security group operation failed",
			err.Error(),
		)
		return
	}

	state.ID = types.StringValue(op.Reference.ID.String())
	state.Name = plan.Name
	state.Description = plan.Description
	if !plan.ExternalSources.IsNull() { // start with empty set to avoid error
		dg := diag.Diagnostics{}
		state.ExternalSources, dg = types.SetValue(types.StringType, []attr.Value{})
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

	}

	tflog.Info(
		ctx,
		"SG created, saving state before syncing external sources",
		map[string]any{},
	)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	if resp.Diagnostics.HasError() || plan.ExternalSources.IsNull() {
		return
	}

	var cidrs []string
	resp.Diagnostics.Append(plan.ExternalSources.ElementsAs(ctx, &cidrs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	setElems := []attr.Value{}
	for _, cidr := range cidrs {
		op, err := r.client.AddExternalSourceToSecurityGroup(
			ctx,
			exoscale.UUID(state.ID.ValueString()),
			exoscale.AddExternalSourceToSecurityGroupRequest{
				Cidr: cidr,
			},
		)
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf(
					"API returned error when adding external source %q",
					cidr,
				),
				err.Error(),
			)
			return
		}

		if _, err := r.client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
			resp.Diagnostics.AddError(
				"add external source to operation failed",
				err.Error(),
			)
			return
		}

		setElems = append(setElems, types.StringValue((cidr)))
		dg := diag.Diagnostics{}
		state.ExternalSources, dg = types.SetValue(types.StringType, setElems)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		tflog.Info(
			ctx,
			"external source added to SG, saving state",
			map[string]any{"cidr": cidr},
		)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
}

// Read (refresh) resources by receiving Terraform prior state data, performing read logic, and saving refreshed Terraform state data.
func (r *Resource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var state ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// If ID is empty (resource doesn't exist), we can remove it from state
	if state.ID.ValueString() == "" {
		resp.State.RemoveResource(ctx)
		return
	}

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to parse ID",
			err.Error(),
		)
		return
	}

	sg, err := r.client.GetSecurityGroup(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			// Resource doesn't exist anymore because it was deleted manually.
			// We must remove it from the state so terraform can report the drift.
			resp.State.RemoveResource(ctx)
		} else {
			resp.Diagnostics.AddError(
				"API returned error reading security group",
				err.Error(),
			)
		}

		return
	}

	state.Name = types.StringValue(sg.Name)
	if !state.Description.IsNull() {
		state.Description = types.StringValue(sg.Description)
	}

	if !state.ExternalSources.IsNull() {
		state.ExternalSources = types.SetNull(types.StringType)

		setElems := []attr.Value{}
		for _, cidr := range sg.ExternalSources {
			setElems = append(setElems, types.StringValue((cidr)))
		}

		dg := diag.Diagnostics{}
		state.ExternalSources, dg = types.SetValue(types.StringType, setElems)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update resources in-place by receiving Terraform prior state, configuration, and plan data, performing update logic, and saving updated Terraform state data.
func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Update(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Only ExternalSources are update-able and also Optional.
	// When `Null` in plan, we don't sync the remote and can exit early.
	if plan.ExternalSources.IsNull() {
		state.ExternalSources = plan.ExternalSources
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to parse ID",
			err.Error(),
		)
		return
	}

	if !plan.ExternalSources.Equal(state.ExternalSources) {
		stateElems := state.ExternalSources.Elements()
		planElems := plan.ExternalSources.Elements()
		if len(stateElems) == 0 && len(planElems) == 0 { // unset in state, empty in plan
			state.ExternalSources = plan.ExternalSources
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}

		newElems, delElems := diffExternalSources(stateElems, planElems)

		for _, deleted := range delElems {
			op, err := r.client.RemoveExternalSourceFromSecurityGroup(
				ctx,
				id,
				exoscale.RemoveExternalSourceFromSecurityGroupRequest{
					Cidr: deleted.(basetypes.StringValue).ValueString(),
				},
			)
			if err != nil {
				if !errors.Is(err, exoscale.ErrNotFound) { // proceed on 404
					resp.Diagnostics.AddError(
						fmt.Sprintf(
							"API returned error when removing external source %q",
							deleted.String(),
						),
						err.Error(),
					)
					return
				}
			} else if _, err := r.client.Wait(
				ctx,
				op,
				exoscale.OperationStateSuccess,
			); err != nil {
				resp.Diagnostics.AddError(
					"remove external source operation failed",
					err.Error(),
				)
				return
			}

			tflog.Info(
				ctx,
				"SG updated, saving state before syncing external sources",
				map[string]any{},
			)
			var deletedKey int
			for key, elem := range stateElems {
				if elem.String() == deleted.String() {
					deletedKey = key
					break
				}
			}
			stateElems = slices.Delete(stateElems, deletedKey, deletedKey+1)
			dg := diag.Diagnostics{}
			state.ExternalSources, dg = types.SetValue(types.StringType, stateElems)
			if dg.HasError() {
				resp.Diagnostics.Append(dg...)
				return
			}

			tflog.Info(
				ctx,
				"external source removed from SG, saving state",
				map[string]any{"cidr": deleted.(basetypes.StringValue).ValueString()},
			)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}

		for _, added := range newElems {
			op, err := r.client.AddExternalSourceToSecurityGroup(
				ctx,
				id,
				exoscale.AddExternalSourceToSecurityGroupRequest{
					Cidr: added.(basetypes.StringValue).ValueString(),
				},
			)
			if err != nil {
				resp.Diagnostics.AddError(
					fmt.Sprintf(
						"API returned error when adding external source %q",
						added.String(),
					),
					err.Error(),
				)
				return
			}
			if _, err := r.client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
				resp.Diagnostics.AddError(
					"add external source operation failed",
					err.Error(),
				)
				return
			}

			tflog.Info(
				ctx,
				"external source added to SG, saving state",
				map[string]any{"cidr": added.(basetypes.StringValue).ValueString()},
			)
			stateElems = append(stateElems, added)
			dg := diag.Diagnostics{}
			state.ExternalSources, dg = types.SetValue(types.StringType, stateElems)
			if dg.HasError() {
				resp.Diagnostics.Append(dg...)
				return
			}

			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}
}

// Delete resources by receiving Terraform prior state data and performing deletion logic.
func (r *Resource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var state ResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Delete(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to parse ID",
			err.Error(),
		)
		return
	}

	op, err := r.client.DeleteSecurityGroup(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"API returned error when deleting security group",
			err.Error(),
		)
		return
	}

	_, err = r.client.Wait(ctx, op, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError(
			"delete security group operation failed",
			err.Error(),
		)
		return
	}
}

// ImportState lets Terraform begin managing existing infrastructure resources.
func (r *Resource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	state := NewResourceModel()

	// Set timeouts (quirk https://github.com/hashicorp/terraform-plugin-framework-timeouts/issues/46)
	var timeouts timeouts.Value
	resp.Diagnostics.Append(
		resp.State.GetAttribute(ctx, path.Root("timeouts"), &timeouts)...,
	)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Timeouts = timeouts

	state.ID = types.StringValue(req.ID)

	// Save state into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// diffExternalSources finds newly added and deleted external sources during update.
func diffExternalSources(state, plan []attr.Value) (added, deleted []attr.Value) {
	stateTable := map[string]bool{}
	for _, val := range state {
		stateTable[val.(basetypes.StringValue).ValueString()] = false
	}

	for _, val := range plan {
		str := val.(basetypes.StringValue).ValueString()
		if _, ok := stateTable[str]; ok {
			stateTable[str] = true
		} else {
			added = append(added, val)
		}
	}

	for str, exists := range stateTable {
		if !exists {
			deleted = append(deleted, types.StringValue(str))
		}
	}

	return
}
