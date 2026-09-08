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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const markdownDescriptionRouteResource = `Manage Exoscale [VPC](https://community.exoscale.com/product/networking/vpc/) Subnet routes.

Routes are immutable: any change to their attributes forces re-creation.

Parent resource: [exoscale_vpc_subnet](./vpc_subnet.md).
`

var _ resource.ResourceWithImportState = (*ResourceRoute)(nil)

type ResourceRoute struct {
	client *exoscale.Client
}

func NewResourceRoute() resource.Resource {
	return &ResourceRoute{}
}

func (r *ResourceRoute) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc_route"
}

func (r *ResourceRoute) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manage Exoscale VPC Subnet routes.",
		MarkdownDescription: markdownDescriptionRouteResource,

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
			"subnet_id": schema.StringAttribute{
				Description:         "The parent Subnet ID.",
				MarkdownDescription: "❗ The parent [exoscale_vpc_subnet](./vpc_subnet.md) ID.",
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
			"destination": schema.StringAttribute{
				Description:         "❗ The route destination CIDR (e.g. `10.9.0.0/24`).",
				MarkdownDescription: "❗ The route destination CIDR (e.g. `10.9.0.0/24`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"target": schema.StringAttribute{
				Description:         "❗ The route target (e.g. `ip=10.0.0.5`).",
				MarkdownDescription: "❗ The route target (e.g. `ip=10.0.0.5`).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Description:         "❗ A free-form text describing the route.",
				MarkdownDescription: "❗ A free-form text describing the route.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

type ResourceRouteModel struct {
	ID          types.String `tfsdk:"id"`
	VpcID       types.String `tfsdk:"vpc_id"`
	SubnetID    types.String `tfsdk:"subnet_id"`
	Zone        types.String `tfsdk:"zone"`
	Destination types.String `tfsdk:"destination"`
	Target      types.String `tfsdk:"target"`
	Description types.String `tfsdk:"description"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *ResourceRoute) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ResourceRoute) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceRouteModel

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
	subnetID, err := exoscale.ParseUUID(plan.SubnetID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse parent Subnet ID", err.Error())
		return
	}
	if _, err := client.GetSubnet(ctx, vpcID, subnetID); err != nil {
		resp.Diagnostics.AddError("API returned error reading parent Subnet", err.Error())
		return
	}

	route, err := client.CreateRoute(ctx, vpcID, subnetID, exoscale.CreateRouteRequest{
		Destination: plan.Destination.ValueString(),
		Target:      plan.Target.ValueString(),
		Description: plan.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("API returned an error when creating route", err.Error())
		return
	}

	plan.ID = types.StringValue(route.ID.String())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceRoute) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceRouteModel

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

	if state.ID.ValueString() == "" || state.VpcID.ValueString() == "" || state.SubnetID.ValueString() == "" {
		tflog.Info(ctx, "route has no ID, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	zone := state.Zone.ValueString()
	if zone == "" {
		tflog.Info(ctx, "route has no zone, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(zone))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	vpcID, err := exoscale.ParseUUID(state.VpcID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse parent VPC ID", err.Error())
		return
	}
	subnetID, err := exoscale.ParseUUID(state.SubnetID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse parent Subnet ID", err.Error())
		return
	}

	// There is no GetRoute endpoint: list the Subnet's routes and find ours.
	resp2, err := client.ListRoutes(ctx, vpcID, subnetID)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("API returned an error while listing routes", err.Error())
		return
	}

	var found *exoscale.ListRouteEntry
	for i, route := range resp2.Routes {
		if route.ID.String() == state.ID.ValueString() {
			found = &resp2.Routes[i]
			break
		}
	}
	if found == nil {
		tflog.Info(ctx, "remote route not found, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	state.Destination = types.StringValue(found.Destination)
	state.Target = types.StringValue(found.Target)
	state.Description = optionalStringValue(found.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is a no-op: every attribute forces resource re-creation.
func (r *ResourceRoute) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *ResourceRoute) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceRouteModel

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

	zone := state.Zone.ValueString()
	if zone == "" {
		tflog.Info(ctx, "route has no zone, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(zone))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	vpcID, err := exoscale.ParseUUID(state.VpcID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse parent VPC ID", err.Error())
		return
	}
	subnetID, err := exoscale.ParseUUID(state.SubnetID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse parent Subnet ID", err.Error())
		return
	}
	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	if err := client.DeleteRoute(ctx, vpcID, subnetID, id); err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("API returned an error when deleting route", err.Error())
		return
	}
}

func (r *ResourceRoute) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, "@")

	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		resp.Diagnostics.AddError(
			"unexpected import identifier",
			"Requires format: vpc_id@subnet_id@route_id@zone",
		)
		return
	}

	vpcID, err := exoscale.ParseUUID(idParts[0])
	if err != nil {
		resp.Diagnostics.AddError("unable to parse parent VPC ID", err.Error())
		return
	}
	subnetID, err := exoscale.ParseUUID(idParts[1])
	if err != nil {
		resp.Diagnostics.AddError("unable to parse parent Subnet ID", err.Error())
		return
	}
	id, err := exoscale.ParseUUID(idParts[2])
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}
	zone := idParts[3]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathVpcID, vpcID.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathSubnetID, subnetID.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathID, id.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathZone, zone)...)
}
