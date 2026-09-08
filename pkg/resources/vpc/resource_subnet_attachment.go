package vpc

import (
	"context"
	"errors"
	"net"
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

const markdownDescriptionInstanceAttachmentResource = `Attach an Exoscale [Compute Instance](../resources/compute_instance.md) to a [VPC](https://community.exoscale.com/product/networking/vpc/) Subnet.

See [README.md](https://github.com/exoscale/terraform-provider-exoscale/blob/master/pkg/resources/vpc/README.md) for the rationale behind modeling this as a standalone resource rather than a block on ` + "`exoscale_compute_instance`" + `.
`

var _ resource.ResourceWithImportState = (*ResourceSubnetAttachment)(nil)

type ResourceSubnetAttachment struct {
	client *exoscale.Client
}

func NewResourceSubnetAttachment() resource.Resource {
	return &ResourceSubnetAttachment{}
}

func (r *ResourceSubnetAttachment) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc_subnet_attachment"
}

func (r *ResourceSubnetAttachment) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Attach a Compute instance to a VPC Subnet.",
		MarkdownDescription: markdownDescriptionInstanceAttachmentResource,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"instance_id": schema.StringAttribute{
				Description:         "The Compute instance ID to attach.",
				MarkdownDescription: "❗ The [exoscale_compute_instance](../resources/compute_instance.md) (ID) to attach.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"vpc_id": schema.StringAttribute{
				Description:         "The VPC ID the Subnet belongs to.",
				MarkdownDescription: "❗ The [exoscale_vpc](./vpc.md) (ID) the Subnet belongs to.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"subnet_id": schema.StringAttribute{
				Description:         "The Subnet ID to attach the instance to.",
				MarkdownDescription: "❗ The [exoscale_vpc_subnet](./vpc_subnet.md) (ID) to attach the instance to.",
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
			"ipv4_address": schema.StringAttribute{
				Description:         "❗ The IPv4 address to assign to the instance in the Subnet. Automatically allocated by the platform if not set.",
				MarkdownDescription: "❗ The IPv4 address to assign to the instance in the Subnet. Automatically allocated by the platform if not set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

type ResourceSubnetAttachmentModel struct {
	ID          types.String `tfsdk:"id"`
	InstanceID  types.String `tfsdk:"instance_id"`
	VpcID       types.String `tfsdk:"vpc_id"`
	SubnetID    types.String `tfsdk:"subnet_id"`
	Zone        types.String `tfsdk:"zone"`
	IPv4Address types.String `tfsdk:"ipv4_address"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *ResourceSubnetAttachment) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ResourceSubnetAttachment) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceSubnetAttachmentModel

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

	instanceID, err := exoscale.ParseUUID(plan.InstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse instance ID", err.Error())
		return
	}
	vpcID, err := exoscale.ParseUUID(plan.VpcID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse VPC ID", err.Error())
		return
	}
	subnetID, err := exoscale.ParseUUID(plan.SubnetID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse Subnet ID", err.Error())
		return
	}

	attachReq := exoscale.AttachInstanceToSubnetRequest{
		Instance: &exoscale.InstanceRef{ID: instanceID},
	}
	if ip := plan.IPv4Address.ValueString(); ip != "" {
		parsed := net.ParseIP(ip)
		if parsed == nil || parsed.To4() == nil {
			resp.Diagnostics.AddError("invalid ipv4_address", "must be a valid IPv4 address: "+ip)
			return
		}
		attachReq.Ipv4 = parsed
	}

	operation, err := client.AttachInstanceToSubnet(ctx, vpcID, subnetID, attachReq)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error when attaching instance to Subnet", err.Error())
		return
	}
	if _, err := client.Wait(ctx, operation, exoscale.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("attach instance to Subnet operation failed", err.Error())
		return
	}

	instance, err := client.GetInstance(ctx, instanceID)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching instance", err.Error())
		return
	}

	ipv4 := ""
	if instance.Vpc != nil {
		for _, s := range instance.Vpc.Subnets {
			if s.ID == subnetID {
				if s.Ipv4 != nil {
					ipv4 = s.Ipv4.String()
				}
				break
			}
		}
	}

	plan.ID = types.StringValue(instanceID.String() + "/" + vpcID.String() + "/" + subnetID.String())
	plan.IPv4Address = types.StringValue(ipv4)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceSubnetAttachment) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceSubnetAttachmentModel

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

	zone := state.Zone.ValueString()
	if zone == "" || state.InstanceID.ValueString() == "" {
		tflog.Info(ctx, "attachment has no zone/instance ID, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(zone))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	instanceID, err := exoscale.ParseUUID(state.InstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse instance ID", err.Error())
		return
	}
	subnetID, err := exoscale.ParseUUID(state.SubnetID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse Subnet ID", err.Error())
		return
	}

	instance, err := client.GetInstance(ctx, instanceID)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("API returned an error while fetching instance", err.Error())
		return
	}

	var found *exoscale.InstanceVpcSubnets
	if instance.Vpc != nil {
		for i, s := range instance.Vpc.Subnets {
			if s.ID == subnetID {
				found = &instance.Vpc.Subnets[i]
				break
			}
		}
	}
	if found == nil {
		tflog.Info(ctx, "instance is no longer attached to this Subnet, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	ipv4 := ""
	if found.Ipv4 != nil {
		ipv4 = found.Ipv4.String()
	}
	state.IPv4Address = types.StringValue(ipv4)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is a no-op: every attribute forces resource re-creation.
func (r *ResourceSubnetAttachment) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *ResourceSubnetAttachment) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceSubnetAttachmentModel

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
		tflog.Info(ctx, "attachment has no zone, deleting from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(zone))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	instanceID, err := exoscale.ParseUUID(state.InstanceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse instance ID", err.Error())
		return
	}
	vpcID, err := exoscale.ParseUUID(state.VpcID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse VPC ID", err.Error())
		return
	}
	subnetID, err := exoscale.ParseUUID(state.SubnetID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse Subnet ID", err.Error())
		return
	}

	operation, err := client.DetachInstanceFromSubnet(ctx, vpcID, subnetID, exoscale.DetachInstanceFromSubnetRequest{
		Instance: &exoscale.InstanceRef{ID: instanceID},
	})
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("API returned an error when detaching instance from Subnet", err.Error())
		return
	}
	if _, err := client.Wait(ctx, operation, exoscale.OperationStateSuccess); err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("detach instance from Subnet operation failed", err.Error())
		return
	}
}

func (r *ResourceSubnetAttachment) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, "@")

	if len(idParts) != 4 || idParts[0] == "" || idParts[1] == "" || idParts[2] == "" || idParts[3] == "" {
		resp.Diagnostics.AddError(
			"unexpected import identifier",
			"Requires format: instance_id@vpc_id@subnet_id@zone",
		)
		return
	}

	instanceID, err := exoscale.ParseUUID(idParts[0])
	if err != nil {
		resp.Diagnostics.AddError("unable to parse instance ID", err.Error())
		return
	}
	vpcID, err := exoscale.ParseUUID(idParts[1])
	if err != nil {
		resp.Diagnostics.AddError("unable to parse VPC ID", err.Error())
		return
	}
	subnetID, err := exoscale.ParseUUID(idParts[2])
	if err != nil {
		resp.Diagnostics.AddError("unable to parse Subnet ID", err.Error())
		return
	}
	zone := idParts[3]

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathID,
		instanceID.String()+"/"+vpcID.String()+"/"+subnetID.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathInstanceID, instanceID.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathVpcID, vpcID.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathSubnetID, subnetID.String())...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathZone, zone)...)
}
