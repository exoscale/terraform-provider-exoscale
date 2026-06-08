package privatenetwork

import (
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/validators"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const markdownDescriptionResource = `Manage Exoscale [Private Networks](https://community.exoscale.com/product/networking/private-network).

Corresponding data source: [exoscale_private_network](../data-sources/private_network.md).
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
	resp.TypeName = req.ProviderTypeName + "_private_network"
}

func (r *Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manage Exoscale Private Networks.",
		MarkdownDescription: markdownDescriptionResource,
		Version:             0,

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
				Description:         "The private network name.",
				MarkdownDescription: "The private network name.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Description:         "A free-form text describing the network.",
				MarkdownDescription: "A free-form text describing the network.",
			},
			"labels": schema.MapAttribute{
				Description:         "A map of key/value labels.",
				MarkdownDescription: "A map of key/value labels.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"zone": schema.StringAttribute{
				Description:         "❗ The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				MarkdownDescription: "❗ The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"end_ip": schema.StringAttribute{
				Description:         "(For managed Privnets) The first/last IPv4 addresses used by the DHCP service for dynamic leases.",
				MarkdownDescription: "(For managed Privnets) The first/last IPv4 addresses used by the DHCP service for dynamic leases.",
				Optional:            true,
				Validators:          []validator.String{validators.IsIPAddress()},
			},
			"start_ip": schema.StringAttribute{
				Optional:    true,
				Description: "(For managed Privnets) The first/last IPv4 addresses used by the DHCP service for dynamic leases.",
				Validators:  []validator.String{validators.IsIPAddress()},
			},
			"netmask": schema.StringAttribute{
				Optional:            true,
				Description:         "(For managed Privnets) The network mask defining the IPv4 network allowed for static leases.",
				MarkdownDescription: "(For managed Privnets) The network mask defining the IPv4 network allowed for static leases.",
				Validators:          []validator.String{validators.IsNetmask()},
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
	StartIP     types.String `tfsdk:"start_ip"`
	EndIP       types.String `tfsdk:"end_ip"`
	Netmask     types.String `tfsdk:"netmask"`

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

	request := exoscale.CreatePrivateNetworkRequest{
		Description: plan.Description.ValueString(),
		EndIP:       net.ParseIP(plan.EndIP.ValueString()),
		Name:        plan.Name.ValueString(),
		StartIP:     net.ParseIP(plan.StartIP.ValueString()),
		Netmask:     net.ParseIP(plan.Netmask.ValueString()),
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

	operation, err := client.CreatePrivateNetwork(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"API returned an error when creating private network",
			err.Error(),
		)
		return
	}

	if _, err := r.client.Wait(ctx, operation); err != nil {
		resp.Diagnostics.AddError(
			"create private network operation failed",
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
		tflog.Info(
			ctx,
			"private network has no ID, deleting from state to report drift",
			map[string]any{},
		)
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

	zone := state.Zone.ValueString()
	if zone == "" {
		tflog.Info(
			ctx,
			"private network has no zone, deleting from state to report drift",
			map[string]any{},
		)
		resp.State.RemoveResource(ctx)
		return
	} else if !slices.Contains(config.Zones, zone) {
		resp.Diagnostics.AddError("invalid value", "zone must be a valid exoscale zone")
	}

	client, err := utils.SwitchClientZone(
		ctx,
		r.client,
		exoscale.ZoneName(zone),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to change exoscale client zone",
			err.Error(),
		)
		return
	}

	privateNetwork, err := client.GetPrivateNetwork(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"API returned an error while fetching private network",
			err.Error(),
		)
		return
	}

	state = ResourceModel{
		ID:          types.StringValue(privateNetwork.ID.String()),
		Name:        types.StringValue(privateNetwork.Name),
		Zone:        state.Zone,
		Description: optionalStringValue(privateNetwork.Description),
		StartIP:     ipStringValue(privateNetwork.StartIP),
		EndIP:       ipStringValue(privateNetwork.EndIP),
		Netmask:     ipStringValue(privateNetwork.Netmask),
		Timeouts:    state.Timeouts,
	}
	state.Labels = types.MapNull(types.StringType)
	if privateNetwork.Labels != nil {
		labels, dg := types.MapValueFrom(
			ctx,
			types.StringType,
			privateNetwork.Labels,
		)
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
		tflog.Info(
			ctx,
			"private network has no zone, deleting from state to report drift",
			map[string]any{},
		)
		resp.State.RemoveResource(ctx)
		return
	} else if !slices.Contains(config.Zones, zone) {
		resp.Diagnostics.AddError("invalid value", "zone must be a valid exoscale zone")
	}

	id, err := exoscale.ParseUUID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to parse ID",
			err.Error(),
		)
		return
	}

	client, err := utils.SwitchClientZone(
		ctx,
		r.client,
		exoscale.ZoneName(zone),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to change exoscale client zone",
			err.Error(),
		)
		return
	}

	request := exoscale.UpdatePrivateNetworkRequest{
		Description: plan.Description.ValueString(),
		EndIP:       net.ParseIP(plan.EndIP.ValueString()),
		Name:        plan.Name.ValueString(),
		StartIP:     net.ParseIP(plan.StartIP.ValueString()),
		Netmask:     net.ParseIP(plan.Netmask.ValueString()),
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

	operation, err := client.UpdatePrivateNetwork(ctx, id, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"API returned an error when updating private network",
			err.Error(),
		)
		return
	}

	if _, err := r.client.Wait(ctx, operation); err != nil {
		resp.Diagnostics.AddError(
			"create private network operation failed",
			err.Error(),
		)
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
	timeout, diags := state.Timeouts.Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if state.ID.ValueString() == "" {
		tflog.Info(
			ctx,
			"private network has no ID, deleting from state to report drift",
			map[string]any{},
		)
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

	zone := state.Zone.ValueString()
	if zone == "" {
		tflog.Info(
			ctx,
			"private network has no zone, deleting from state to report drift",
			map[string]any{},
		)
		resp.State.RemoveResource(ctx)
		return
	} else if !slices.Contains(config.Zones, zone) {
		resp.Diagnostics.AddError("invalid value", "zone must be a valid exoscale zone")
	}

	client, err := utils.SwitchClientZone(
		ctx,
		r.client,
		exoscale.ZoneName(zone),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to change exoscale client zone",
			err.Error(),
		)
		return
	}

	op, err := client.DeletePrivateNetwork(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"API returned an error while fetching private network",
			err.Error(),
		)
		return
	}

	if _, err := client.Wait(ctx, op); err != nil {
		resp.Diagnostics.AddError(
			"create private network operation failed",
			err.Error(),
		)
		return
	}
}

func optionalStringValue(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func ipStringValue(ip net.IP) types.String {
	if ip == nil {
		return types.StringNull()
	}
	return types.StringValue(ip.String())
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, "@")

	if len(idParts) != 2 {
		resp.Diagnostics.AddError(
			"unexpected import identifier",
			fmt.Sprintf("Expected import identifier with format: id@zone. Got: %q", req.ID),
		)
		return
	}

	if idParts[0] == "" {
		tflog.Info(
			ctx,
			"private network has no ID, deleting from state to report drift",
			map[string]any{},
		)
		resp.State.RemoveResource(ctx)
		return
	}

	id, err := exoscale.ParseUUID(idParts[0])
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to parse ID",
			err.Error(),
		)
		return
	}

	zone := idParts[1]
	if zone == "" {
		tflog.Info(
			ctx,
			"private network has no zone, deleting from state to report drift",
			map[string]any{},
		)
		resp.State.RemoveResource(ctx)
		return
	} else if !slices.Contains(config.Zones, zone) {
		resp.Diagnostics.AddError("invalid value", "zone must be a valid exoscale zone")
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
