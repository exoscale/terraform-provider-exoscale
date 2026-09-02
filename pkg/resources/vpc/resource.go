package vpc

import (
	"context"
	"errors"
	"slices"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const markdownDescriptionResource = `Manage Exoscale Virtual Private Clouds (VPC).

Corresponding data source: [exoscale_vpc](../data-sources/vpc.md).
`

var _ resource.ResourceWithImportState = (*Resource)(nil)

type Resource struct {
	client *exoscale.Client
}

func NewResource() resource.Resource {
	return &Resource{}
}

// Metadata specifies resource name.
func (r *Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc"
}

func (r *Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manage Exoscale Virtual Private Clouds (VPC).",
		MarkdownDescription: markdownDescriptionResource,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "The VPC name.",
				MarkdownDescription: "The VPC name.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Description:         "A text describing the VPC.",
				MarkdownDescription: "A text describing the VPC.",
			},
			"labels": schema.MapAttribute{
				Description:         "A map of key/value labels.",
				MarkdownDescription: "A map of key/value labels.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"zone": schema.StringAttribute{
				Description:         "❗ The Exoscale zone name.",
				MarkdownDescription: "❗ The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"default": schema.BoolAttribute{
				Description:         "Whether this is the organization's default VPC for the zone.",
				MarkdownDescription: "Whether this is the organization's default VPC for the zone.",
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

type ResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Zone        types.String `tfsdk:"zone"`
	Description types.String `tfsdk:"description"`
	Labels      types.Map    `tfsdk:"labels"`
	Default     types.Bool   `tfsdk:"default"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Create(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := utils.SwitchClientZone(
		ctx,
		r.client,
		exoscale.ZoneName(plan.Zone.ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to change exoscale client zone",
			err.Error(),
		)
		return
	}

	request := exoscale.CreateVpcRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}
	if len(plan.Labels.Elements()) > 0 {
		labels := exoscale.Labels{}

		dg := plan.Labels.ElementsAs(ctx, &labels, false)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		request.Labels = labels
	}

	operation, err := client.CreateVpc(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"API returned an error when creating VPC",
			err.Error(),
		)
		return
	}

	operation, err = client.Wait(ctx, operation, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError(
			"create VPC operation failed",
			err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(operation.Reference.ID.String())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
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

	if state.ID.ValueString() == "" {
		tflog.Info(ctx, "VPC has no ID, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	zone := state.Zone.ValueString()
	if zone == "" {
		tflog.Info(ctx, "VPC has no zone, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	client, err := utils.SwitchClientZone(
		ctx,
		r.client,
		exoscale.ZoneName(zone),
	)
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	vpc, err := client.GetVpc(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("API returned an error while fetching VPC", err.Error())
		return
	}

	state = ResourceModel{
		ID:          types.StringValue(vpc.ID.String()),
		Name:        types.StringValue(vpc.Name),
		Zone:        state.Zone,
		Description: optionalStringValue(vpc.Description),
		Default:     types.BoolValue(vpc.Default != nil && *vpc.Default),
		Timeouts:    state.Timeouts,
	}
	state.Labels = types.MapNull(types.StringType)
	if len(vpc.Labels) > 0 {
		labels, dg := types.MapValueFrom(ctx, types.StringType, vpc.Labels)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}
		state.Labels = labels
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceModel

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

	zone := plan.Zone.ValueString()
	if zone == "" {
		tflog.Info(ctx, "VPC has no zone, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	id, err := exoscale.ParseUUID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	client, err := utils.SwitchClientZone(
		ctx,
		r.client,
		exoscale.ZoneName(zone),
	)
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	name := plan.Name.ValueString()
	description := plan.Description.ValueString()
	request := exoscale.UpdateVpcRequest{
		Name:        &name,
		Description: &description,
	}
	if len(plan.Labels.Elements()) > 0 {
		labels := exoscale.Labels{}

		dg := plan.Labels.ElementsAs(ctx, &labels, false)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		request.Labels = labels
	}

	if _, err := client.UpdateVpc(ctx, id, request); err != nil {
		resp.Diagnostics.AddError("API returned an error when updating VPC", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
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

	if state.ID.ValueString() == "" {
		tflog.Info(ctx, "VPC has no ID, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	zone := state.Zone.ValueString()
	if zone == "" {
		tflog.Info(ctx, "VPC has no zone, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	client, err := utils.SwitchClientZone(
		ctx,
		r.client,
		exoscale.ZoneName(zone),
	)
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	if err := client.DeleteVpc(ctx, id); err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("API returned an error while deleting VPC", err.Error())
		return
	}
}

func optionalStringValue(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, "@")

	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			"unexpected import identifier",
			`Expected import identifier with format: id@zone. Got: "`+req.ID+`"`,
		)
		return
	}

	if idParts[0] == "" {
		tflog.Info(ctx, "VPC has no ID, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	id, err := exoscale.ParseUUID(idParts[0])
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	zone := idParts[1]
	if zone == "" {
		tflog.Info(ctx, "VPC has no zone, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	} else if !slices.Contains(config.Zones, zone) {
		resp.Diagnostics.AddError("invalid value", "zone must be a valid exoscale zone")
		return
	}

	// Set timeouts (quirk https://github.com/hashicorp/terraform-plugin-framework-timeouts/issues/46)
	var t timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &t)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &ResourceModel{
		ID:       types.StringValue(id.String()),
		Zone:     types.StringValue(zone),
		Labels:   types.MapNull(types.StringType),
		Timeouts: t,
	})...)
}
