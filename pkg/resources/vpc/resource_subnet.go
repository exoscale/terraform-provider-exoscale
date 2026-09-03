package vpc

import (
	"context"
	"errors"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const markdownDescriptionSubnetResource = `Manage Exoscale [VPC](https://community.exoscale.com/product/networking/vpc/) Subnets.

Corresponding data source: [exoscale_vpc_subnet](../data-sources/vpc_subnet.md).
`

var _ resource.ResourceWithImportState = (*ResourceSubnet)(nil)

type ResourceSubnet struct {
	client *exoscale.Client
}

func NewResourceSubnet() resource.Resource {
	return &ResourceSubnet{}
}

func (r *ResourceSubnet) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc_subnet"
}

func (r *ResourceSubnet) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manage Exoscale VPC Subnets.",
		MarkdownDescription: markdownDescriptionSubnetResource,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Description:         "The parent VPC ID.",
				MarkdownDescription: "❗ The parent [exoscale_vpc](./vpc.md) ID.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
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
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "The Subnet name.",
				MarkdownDescription: "The Subnet name.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				Description:         "A free-form text describing the Subnet.",
				MarkdownDescription: "A free-form text describing the Subnet.",
			},
			"ipv4_block": schema.StringAttribute{
				Description:         "❗ The Subnet IPv4 CIDR (e.g. `10.0.0.0/24`). Automatically allocated by the platform if not set.",
				MarkdownDescription: "❗ The Subnet IPv4 CIDR (e.g. `10.0.0.0/24`). Automatically allocated by the platform if not set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"address_family": schema.StringAttribute{
				Description:         "❗ The Subnet address family. Currently only `inet4` is supported.",
				MarkdownDescription: "❗ The Subnet address family. Currently only `inet4` is supported.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(string(exoscale.CreateSubnetRequestAddressfamilyInet4)),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(string(exoscale.CreateSubnetRequestAddressfamilyInet4)),
				},
			},
			"address_space": schema.StringAttribute{
				Description:         "❗ The Subnet address space. Currently only `private` is supported.",
				MarkdownDescription: "❗ The Subnet address space. Currently only `private` is supported.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(string(exoscale.CreateSubnetRequestAddressSpacePrivate)),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(string(exoscale.CreateSubnetRequestAddressSpacePrivate)),
				},
			},
			"labels": schema.MapAttribute{
				Description:         "A map of key/value labels.",
				MarkdownDescription: "A map of key/value labels.",
				ElementType:         types.StringType,
				Optional:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

type ResourceSubnetModel struct {
	ID            types.String `tfsdk:"id"`
	VpcID         types.String `tfsdk:"vpc_id"`
	Zone          types.String `tfsdk:"zone"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	IPv4Block     types.String `tfsdk:"ipv4_block"`
	AddressFamily types.String `tfsdk:"address_family"`
	AddressSpace  types.String `tfsdk:"address_space"`
	Labels        types.Map    `tfsdk:"labels"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *ResourceSubnet) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ResourceSubnet) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceSubnetModel

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

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(plan.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	vpcID, err := exoscale.ParseUUID(plan.VpcID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse parent VPC ID", err.Error())
		return
	}
	if _, err := client.GetVpc(ctx, vpcID); err != nil {
		resp.Diagnostics.AddError("API returned error reading parent VPC", err.Error())
		return
	}

	request := exoscale.CreateSubnetRequest{
		Name:          plan.Name.ValueString(),
		Description:   plan.Description.ValueString(),
		Ipv4Block:     plan.IPv4Block.ValueString(),
		AddressSpace:  exoscale.CreateSubnetRequestAddressSpace(plan.AddressSpace.ValueString()),
		Addressfamily: exoscale.CreateSubnetRequestAddressfamily(plan.AddressFamily.ValueString()),
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

	operation, err := client.CreateSubnet(ctx, vpcID, request)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error when creating Subnet", err.Error())
		return
	}

	operation, err = client.Wait(ctx, operation, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("create Subnet operation failed", err.Error())
		return
	}

	// ipv4_block (and, in principle, address_family/address_space) may be
	// server-allocated/normalized: re-fetch the Subnet so no Computed
	// attribute is left Unknown in the final state.
	subnet, err := client.GetSubnet(ctx, vpcID, operation.Reference.ID)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching created Subnet", err.Error())
		return
	}

	plan.ID = types.StringValue(subnet.ID.String())
	plan.IPv4Block = optionalStringValue(subnet.Ipv4Block)
	plan.AddressFamily = types.StringValue(string(subnet.Addressfamily))
	plan.AddressSpace = types.StringValue(string(subnet.AddressSpace))

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceSubnet) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceSubnetModel

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

	if state.ID.ValueString() == "" || state.VpcID.ValueString() == "" {
		tflog.Info(ctx, "Subnet has no ID, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	vpcID, err := exoscale.ParseUUID(state.VpcID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse parent VPC ID", err.Error())
		return
	}
	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	zone := state.Zone.ValueString()
	if zone == "" {
		tflog.Info(ctx, "Subnet has no zone, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(zone))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	subnet, err := client.GetSubnet(ctx, vpcID, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("API returned an error while fetching Subnet", err.Error())
		return
	}

	state = ResourceSubnetModel{
		ID:            types.StringValue(subnet.ID.String()),
		VpcID:         state.VpcID,
		Zone:          state.Zone,
		Name:          types.StringValue(subnet.Name),
		Description:   optionalStringValue(subnet.Description),
		IPv4Block:     optionalStringValue(subnet.Ipv4Block),
		AddressFamily: types.StringValue(string(subnet.Addressfamily)),
		AddressSpace:  types.StringValue(string(subnet.AddressSpace)),
		Timeouts:      state.Timeouts,
	}
	state.Labels = types.MapNull(types.StringType)
	if len(subnet.Labels) > 0 {
		labels, dg := types.MapValueFrom(ctx, types.StringType, subnet.Labels)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}
		state.Labels = labels
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ResourceSubnet) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceSubnetModel

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
		tflog.Info(ctx, "Subnet has no zone, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(zone))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	vpcID, err := exoscale.ParseUUID(plan.VpcID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse parent VPC ID", err.Error())
		return
	}
	id, err := exoscale.ParseUUID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	name := plan.Name.ValueString()
	description := plan.Description.ValueString()
	request := exoscale.UpdateSubnetRequest{
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

	if _, err := client.UpdateSubnet(ctx, vpcID, id, request); err != nil {
		resp.Diagnostics.AddError("API returned an error when updating Subnet", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceSubnet) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceSubnetModel

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

	if state.ID.ValueString() == "" || state.VpcID.ValueString() == "" {
		tflog.Info(ctx, "Subnet has no ID, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	vpcID, err := exoscale.ParseUUID(state.VpcID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse parent VPC ID", err.Error())
		return
	}
	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	zone := state.Zone.ValueString()
	if zone == "" {
		tflog.Info(ctx, "Subnet has no zone, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(zone))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	if err := client.DeleteSubnet(ctx, vpcID, id); err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("API returned an error while deleting Subnet", err.Error())
		return
	}
}

func (r *ResourceSubnet) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, "@")

	if len(idParts) != 3 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" {
		resp.Diagnostics.AddError(
			"unexpected import identifier",
			"Requires format: vpc_id@subnet_id@zone",
		)
		return
	}

	vpcID, err := exoscale.ParseUUID(idParts[0])
	if err != nil {
		resp.Diagnostics.AddError("unable to parse parent VPC ID", err.Error())
		return
	}
	id, err := exoscale.ParseUUID(idParts[1])
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}
	zone := idParts[2]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathVpcID, vpcID.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathID, id.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathZone, zone)...)
}
