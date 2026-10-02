package instance

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

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const markdownDescriptionResource = `Manage Exoscale [Compute Instances](https://community.exoscale.com/documentation/compute/).

Corresponding data sources: [exoscale_compute_instance](../data-sources/compute_instance.md), [exoscale_compute_instance_list](../data-sources/compute_instance_list.md).

After the creation, you can retrieve the password of an instance with [Exoscale CLI](https://github.com/exoscale/cli): ` + "`exo compute instance reveal-password NAME`" + `.`

var _ resource.Resource = (*Resource)(nil)
var _ resource.ResourceWithImportState = (*Resource)(nil)
var _ resource.ResourceWithModifyPlan = (*Resource)(nil)

type Resource struct {
	client *exoscale.Client
}

// NewResource creates an instance of Resource.
func NewResource() resource.Resource {
	return &Resource{}
}

// ResourceModel defines the exoscale_compute_instance resource data model.
type ResourceModel struct {
	ID                    types.String `tfsdk:"id"`
	AntiAffinityGroupIDs  types.Set    `tfsdk:"anti_affinity_group_ids"`
	BlockStorageVolumeIDs types.Set    `tfsdk:"block_storage_volume_ids"`
	CreatedAt             types.String `tfsdk:"created_at"`
	DeployTargetID        types.String `tfsdk:"deploy_target_id"`
	DestroyProtected      types.Bool   `tfsdk:"destroy_protected"`
	DiskSize              types.Int64  `tfsdk:"disk_size"`
	ElasticIPIDs          types.Set    `tfsdk:"elastic_ip_ids"`
	EnableSecureBoot      types.Bool   `tfsdk:"enable_secure_boot"`
	EnableTPM             types.Bool   `tfsdk:"enable_tpm"`
	IPv6                  types.Bool   `tfsdk:"ipv6"`
	IPv6Address           types.String `tfsdk:"ipv6_address"`
	Labels                types.Map    `tfsdk:"labels"`
	MACAddress            types.String `tfsdk:"mac_address"`
	Name                  types.String `tfsdk:"name"`
	NetworkInterface      types.Set    `tfsdk:"network_interface"`
	Private               types.Bool   `tfsdk:"private"`
	PrivateNetworkIDs     types.Set    `tfsdk:"private_network_ids"`
	PublicIPAddress       types.String `tfsdk:"public_ip_address"`
	ReverseDNS            types.String `tfsdk:"reverse_dns"`
	SSHKey                types.String `tfsdk:"ssh_key"`
	SSHKeys               types.Set    `tfsdk:"ssh_keys"`
	SecurityGroupIDs      types.Set    `tfsdk:"security_group_ids"`
	State                 types.String `tfsdk:"state"`
	TemplateID            types.String `tfsdk:"template_id"`
	Type                  types.String `tfsdk:"type"`
	UserData              types.String `tfsdk:"user_data"`
	Zone                  types.String `tfsdk:"zone"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// Metadata specifies the resource name.
func (r *Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compute_instance"
}

// Schema defines the resource attributes.
func (r *Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manage Exoscale Compute Instances.",
		MarkdownDescription: markdownDescriptionResource,

		Attributes: map[string]schema.Attribute{
			AttrID: schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			AttrAntiAffinityGroupIDs: schema.SetAttribute{
				MarkdownDescription: "❗ A list of [exoscale_anti_affinity_group](./anti_affinity_group.md) (IDs) to attach to the instance (may only be set at creation time).",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					emptySetIsNull{},
					setRequiresReplace(),
				},
			},
			AttrBlockStorageVolumeIDs: schema.SetAttribute{
				MarkdownDescription: "A list of [exoscale_block_storage_volume](./block_storage_volume.md) (ID) to attach to the instance.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					emptySetIsNull{},
				},
			},
			AttrCreatedAt: schema.StringAttribute{
				MarkdownDescription: "The instance creation date.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			AttrDeployTargetID: schema.StringAttribute{
				MarkdownDescription: "❗ A deploy target ID.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					emptyStringIsNull{},
					stringRequiresReplace(),
				},
			},
			AttrDestroyProtected: schema.BoolAttribute{
				MarkdownDescription: "Mark the instance as protected, the Exoscale API will refuse to delete the instance until the protection is removed (boolean; default: `false`).",
				Optional:            true,
			},
			AttrDiskSize: schema.Int64Attribute{
				MarkdownDescription: "The instance disk size (GiB; at least `10`). Can not be decreased after creation. **WARNING**: updating this attribute stops/restarts the instance.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(10),
				},
			},
			AttrElasticIPIDs: schema.SetAttribute{
				MarkdownDescription: "A list of [exoscale_elastic_ip](./elastic_ip.md) (IDs) to attach to the instance.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					emptySetIsNull{},
				},
			},
			AttrEnableSecureBoot: schema.BoolAttribute{
				MarkdownDescription: "❗ Enable secure boot on the instance (boolean; default: `false`). Can not be changed after the creation.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					// State written by the SDKv2 for an instance the API reports
					// no secure boot status for holds no value until the next
					// refresh: that must not replace the instance.
					boolplanmodifier.RequiresReplaceIf(
						func(_ context.Context, req planmodifier.BoolRequest, resp *boolplanmodifier.RequiresReplaceIfFuncResponse) {
							resp.RequiresReplace = !req.StateValue.IsNull()
						},
						"Changing secure boot replaces the instance.",
						"Changing secure boot replaces the instance.",
					),
				},
			},
			AttrEnableTPM: schema.BoolAttribute{
				MarkdownDescription: "Enable TPM on the instance (boolean; default: `false`). Can not be disabled after the creation. **WARNING**: enabling this attribute stops/restarts the instance.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			AttrIPv6: schema.BoolAttribute{
				MarkdownDescription: "Enable IPv6 on the instance (boolean; default: `false`). Can not be disabled after being enabled.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			AttrIPv6Address: schema.StringAttribute{
				MarkdownDescription: "The instance (main network interface) IPv6 address (if enabled).",
				Computed:            true,
			},
			AttrLabels: schema.MapAttribute{
				MarkdownDescription: "A map of key/value labels.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Map{
					emptyMapIsNull{},
				},
			},
			AttrMACAddress: schema.StringAttribute{
				MarkdownDescription: "MAC address",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			AttrName: schema.StringAttribute{
				MarkdownDescription: "The compute instance name.",
				Required:            true,
			},
			AttrPrivate: schema.BoolAttribute{
				MarkdownDescription: "Whether the instance is private (no public IP addresses; default: false)",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			AttrPrivateNetworkIDs: schema.SetAttribute{
				MarkdownDescription: "A list of private networks (IDs) attached to the instance. Please use the `network_interface.*.network_id` argument instead.",
				DeprecationMessage:  "Use the network_interface block instead.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			AttrPublicIPAddress: schema.StringAttribute{
				MarkdownDescription: "The instance (main network interface) IPv4 address.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			AttrReverseDNS: schema.StringAttribute{
				MarkdownDescription: "Domain name for reverse DNS record.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					emptyStringIsNull{},
				},
			},
			AttrSSHKey: schema.StringAttribute{
				MarkdownDescription: "❗ The [exoscale_ssh_key](./ssh_key.md) (name) to authorize in the instance (may only be set at creation time).",
				DeprecationMessage:  "Use ssh_keys instead",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					stringRequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot(AttrSSHKeys)),
				},
			},
			AttrSSHKeys: schema.SetAttribute{
				MarkdownDescription: "❗ The list of [exoscale_ssh_key](./ssh_key.md) (name) to authorize in the instance (may only be set at creation time).",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					emptySetIsNull{},
					setRequiresReplace(),
				},
			},
			AttrSecurityGroupIDs: schema.SetAttribute{
				MarkdownDescription: "A list of [exoscale_security_group](./security_group.md) (IDs) to attach to the instance.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					emptySetIsNull{},
				},
			},
			AttrState: schema.StringAttribute{
				MarkdownDescription: "The instance state (`running` or `stopped`). If omitted, instance will start and reach `running` state.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					// An update leaves the instance in the state it found it
					// in, unless the configuration says otherwise.
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(exoscale.InstanceStateRunning),
						string(exoscale.InstanceStateStopped),
					),
				},
			},
			AttrTemplateID: schema.StringAttribute{
				MarkdownDescription: "❗ The [exoscale_template](../data-sources/template.md) (ID) to use when creating the instance.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			AttrType: schema.StringAttribute{
				MarkdownDescription: "The instance type (`<family>.<size>`, e.g. `standard.medium`; use the [Exoscale CLI](https://github.com/exoscale/cli/) - `exo compute instance-type list` - for the list of available types). **WARNING**: updating this attribute stops/restarts the instance.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					ignoreCase(),
				},
				Validators: []validator.String{
					stringFuncValidator{
						description: `The instance type must be in the "FAMILY.SIZE" format.`,
						validate: func(v string) error {
							if !strings.Contains(v, ".") {
								return fmt.Errorf(`invalid value %q, expected format "FAMILY.SIZE"`, v)
							}
							return nil
						},
					},
				},
			},
			AttrUserData: schema.StringAttribute{
				MarkdownDescription: "[cloud-init](https://cloudinit.readthedocs.io/) configuration.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					ignoreUserDataEncoding(),
				},
				Validators: []validator.String{
					stringFuncValidator{
						description: "The user data must fit in the maximum allowed length once encoded.",
						validate: func(v string) error {
							_, _, err := utils.EncodeUserData(v)
							return err
						},
					},
				},
			},
			AttrZone: schema.StringAttribute{
				MarkdownDescription: "❗ The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
		},
		Blocks: map[string]schema.Block{
			AttrNetworkInterface: schema.SetNestedBlock{
				MarkdownDescription: "Private network interfaces (may be specified multiple times). Structure is documented below.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"ip_address": schema.StringAttribute{
							MarkdownDescription: "The IPv4 address to request as static DHCP lease if the network interface is attached to a *managed* private network.",
							Optional:            true,
							Computed:            true,
							Validators: []validator.String{
								stringFuncValidator{
									description: "The value must be an IPv4 address.",
									validate: func(v string) error {
										if ip := net.ParseIP(v); ip == nil || ip.To4() == nil {
											return fmt.Errorf("expected a valid IPv4 address, got %q", v)
										}
										return nil
									},
								},
							},
						},
						"mac_address": schema.StringAttribute{
							MarkdownDescription: "MAC address",
							Computed:            true,
						},
						"network_id": schema.StringAttribute{
							MarkdownDescription: "The [exoscale_private_network](./private_network.md) (ID) to attach to the instance.",
							Required:            true,
						},
					},
				},
			},
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

// ModifyPlan keeps the computed attributes an update does not touch from
// being planned as unknown: the IPv6 address only changes when IPv6 gets
// enabled, and the network interfaces only when they are attached.
func (r *Resource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Nothing to do on creation or deletion.
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var plan, state ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.IPv6Address.IsUnknown() && plan.IPv6.Equal(state.IPv6) {
		plan.IPv6Address = state.IPv6Address
	}

	planned, diags := networkInterfaces(ctx, plan.NetworkInterface)
	resp.Diagnostics.Append(diags...)
	current, diags := networkInterfaces(ctx, state.NetworkInterface)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() || plan.NetworkInterface.IsUnknown() {
		return
	}

	attached := len(planned) == len(current)
	for i := range planned {
		nif := findNetworkInterface(current, planned[i])
		if nif == nil {
			attached = false
			continue
		}

		if planned[i].IPAddress.IsUnknown() {
			planned[i].IPAddress = nif.IPAddress
		}
		if planned[i].MACAddress.IsUnknown() {
			planned[i].MACAddress = nif.MACAddress
		}
	}

	plan.NetworkInterface, diags = networkInterfaceSet(ctx, planned)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.PrivateNetworkIDs.IsUnknown() && attached {
		plan.PrivateNetworkIDs = state.PrivateNetworkIDs
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

// findNetworkInterface returns the interface among current that the planned
// one leaves as is: same Private Network, and same IP address when the
// configuration sets one. It returns nil when the planned interface has to be
// (re)attached.
func findNetworkInterface(current []NetworkInterfaceModel, planned NetworkInterfaceModel) *NetworkInterfaceModel {
	if planned.NetworkID.IsUnknown() {
		return nil
	}

	for i := range current {
		if !current[i].NetworkID.Equal(planned.NetworkID) {
			continue
		}

		if planned.IPAddress.IsUnknown() ||
			planned.IPAddress.ValueString() == current[i].IPAddress.ValueString() {
			return &current[i]
		}
	}

	return nil
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) { //nolint:gocyclo
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

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(plan.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	request := exoscale.CreateInstanceRequest{
		Name:               plan.Name.ValueString(),
		DiskSize:           plan.DiskSize.ValueInt64(),
		Template:           &exoscale.Template{ID: exoscale.UUID(plan.TemplateID.ValueString())},
		PublicIPAssignment: exoscale.PublicIPAssignmentInet4,
	}

	switch {
	case plan.Private.ValueBool():
		request.PublicIPAssignment = exoscale.PublicIPAssignmentNone
	case plan.IPv6.ValueBool():
		request.PublicIPAssignment = exoscale.PublicIPAssignmentDual
	}

	if plan.EnableTPM.ValueBool() {
		request.TpmEnabled = exoscale.Ptr(true)
	}
	if plan.EnableSecureBoot.ValueBool() {
		request.SecurebootEnabled = exoscale.Ptr(true)
	}

	if v := plan.DeployTargetID.ValueString(); v != "" {
		request.DeployTarget = &exoscale.DeployTarget{ID: exoscale.UUID(v)}
	}

	antiAffinityGroupIDs, diags := stringSetValues(ctx, plan.AntiAffinityGroupIDs)
	resp.Diagnostics.Append(diags...)
	securityGroupIDs, diags := stringSetValues(ctx, plan.SecurityGroupIDs)
	resp.Diagnostics.Append(diags...)
	sshKeys, diags := stringSetValues(ctx, plan.SSHKeys)
	resp.Diagnostics.Append(diags...)
	elasticIPIDs, diags := stringSetValues(ctx, plan.ElasticIPIDs)
	resp.Diagnostics.Append(diags...)
	blockStorageVolumeIDs, diags := stringSetValues(ctx, plan.BlockStorageVolumeIDs)
	resp.Diagnostics.Append(diags...)
	nifs, diags := networkInterfaces(ctx, plan.NetworkInterface)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	for _, id := range antiAffinityGroupIDs {
		request.AntiAffinityGroups = append(request.AntiAffinityGroups, exoscale.AntiAffinityGroup{ID: exoscale.UUID(id)})
	}
	for _, id := range securityGroupIDs {
		request.SecurityGroups = append(request.SecurityGroups, exoscale.SecurityGroup{ID: exoscale.UUID(id)})
	}

	if len(sshKeys) > 0 {
		for _, name := range sshKeys {
			request.SSHKeys = append(request.SSHKeys, exoscale.SSHKey{Name: name})
		}
	} else if v := plan.SSHKey.ValueString(); v != "" {
		request.SSHKey = &exoscale.SSHKey{Name: v}
	}

	if len(plan.Labels.Elements()) > 0 {
		labels := exoscale.Labels{}
		resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		request.Labels = labels
	}

	instanceType, err := utils.FindInstanceTypeByNameV3(ctx, client, plan.Type.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to retrieve instance type", err.Error())
		return
	}
	request.InstanceType = instanceType

	if v := plan.UserData.ValueString(); v != "" {
		userData, _, err := utils.EncodeUserData(v)
		if err != nil {
			resp.Diagnostics.AddError("unable to encode user data", err.Error())
			return
		}
		request.UserData = userData
	}

	operation, err := client.CreateInstance(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error when creating instance", err.Error())
		return
	}

	operation, err = client.Wait(ctx, operation, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("create instance operation failed", err.Error())
		return
	}

	id := operation.Reference.ID
	plan.ID = types.StringValue(id.String())

	// The instance exists from here on: if one of the steps below fails, keep
	// it in the state (Terraform marks it as tainted) rather than leaving it
	// behind unmanaged.
	created := false
	defer func() {
		if !created {
			resp.Diagnostics.Append(resp.State.Set(ctx, nullUnknown(ctx, plan))...)
		}
	}()

	if plan.DestroyProtected.ValueBool() {
		if err := wait(ctx, client)(client.AddInstanceProtection(ctx, id)); err != nil {
			resp.Diagnostics.AddError("unable to make instance destroy protected", err.Error())
			return
		}
	}

	for _, elasticIPID := range elasticIPIDs {
		if err := attachElasticIP(ctx, client, id, elasticIPID); err != nil {
			resp.Diagnostics.AddError("unable to attach Elastic IP "+elasticIPID, err.Error())
			return
		}
	}

	for _, nif := range nifs {
		if err := attachPrivateNetwork(ctx, client, id, nif); err != nil {
			resp.Diagnostics.AddError("unable to attach Private Network "+nif.NetworkID.ValueString(), err.Error())
			return
		}
	}

	for _, volumeID := range blockStorageVolumeIDs {
		if err := attachBlockStorageVolume(ctx, client, id, volumeID); err != nil {
			resp.Diagnostics.AddError("unable to attach block storage volume "+volumeID, err.Error())
			return
		}
	}

	if v := plan.ReverseDNS.ValueString(); v != "" {
		err := wait(ctx, client)(client.UpdateReverseDNSInstance(
			ctx,
			id,
			exoscale.UpdateReverseDNSInstanceRequest{DomainName: v},
		))
		if err != nil {
			resp.Diagnostics.AddError("unable to create Reverse DNS record", err.Error())
			return
		}
	}

	if plan.State.ValueString() == string(exoscale.InstanceStateStopped) {
		if err := wait(ctx, client)(client.StopInstance(ctx, id)); err != nil {
			resp.Diagnostics.AddError("unable to stop instance", err.Error())
			return
		}
	}

	instance, err := client.GetInstance(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching the created instance", err.Error())
		return
	}

	resp.Diagnostics.Append(applyInstanceComputed(ctx, client, &plan, instance)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created = true
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "resource created", map[string]any{"id": plan.ID})
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
		tflog.Info(ctx, "instance has no ID, removing from state to report drift", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(state.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	instance, err := client.GetInstance(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("API returned an error while fetching instance", err.Error())
		return
	}

	resp.Diagnostics.Append(applyInstance(ctx, client, &state, instance)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Trace(ctx, "resource read done", map[string]any{"id": state.ID})
}

func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) { //nolint:gocyclo
	var plan, state ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
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

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(plan.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	instance, err := client.GetInstance(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching instance", err.Error())
		return
	}

	// Only the attributes that changed are sent: the API leaves an omitted (or
	// null) attribute untouched.
	var (
		update  bool
		request exoscale.UpdateInstanceRequest
	)

	if !plan.Labels.Equal(state.Labels) {
		var labels exoscale.Labels
		resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		// Labels removed from the configuration decode to a nil map, which is
		// serialised as null and left untouched by the API.
		if labels == nil {
			labels = exoscale.Labels{}
		}
		update = true
		request.Labels = labels
	}

	if !plan.Name.Equal(state.Name) {
		update = true
		request.Name = plan.Name.ValueString()
	}

	// TODO(egoscale): UpdateInstanceRequest.UserData is a plain string with
	// `omitempty`, and ResetInstanceField does not cover user-data: user data
	// can be changed but not cleared.
	if plan.UserData.ValueString() != state.UserData.ValueString() && plan.UserData.ValueString() != "" {
		userData, _, err := utils.EncodeUserData(plan.UserData.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("unable to encode user data", err.Error())
			return
		}
		update = true
		request.UserData = userData
	}

	if !plan.IPv6.Equal(state.IPv6) {
		if !plan.IPv6.ValueBool() {
			resp.Diagnostics.AddError("invalid value", "ipv6 can't be disabled")
			return
		}
		if plan.Private.ValueBool() {
			resp.Diagnostics.AddError("invalid value", "ipv6 cannot be enabled on a private instance")
			return
		}
		update = true
		request.PublicIPAssignment = exoscale.PublicIPAssignmentDual
	}

	if update {
		if err := wait(ctx, client)(client.UpdateInstance(ctx, id, request)); err != nil {
			resp.Diagnostics.AddError("unable to update instance", err.Error())
			return
		}
	}

	if plan.ReverseDNS.ValueString() != state.ReverseDNS.ValueString() {
		if v := plan.ReverseDNS.ValueString(); v == "" {
			err = wait(ctx, client)(client.DeleteReverseDNSInstance(ctx, id))
		} else {
			err = wait(ctx, client)(client.UpdateReverseDNSInstance(
				ctx,
				id,
				exoscale.UpdateReverseDNSInstanceRequest{DomainName: v},
			))
		}
		if err != nil {
			resp.Diagnostics.AddError("unable to update Reverse DNS record", err.Error())
			return
		}
	}

	// Attach/detach Block Storage Volumes
	if !plan.BlockStorageVolumeIDs.Equal(state.BlockStorageVolumeIDs) {
		added, removed, diags := stringSetChange(ctx, state.BlockStorageVolumeIDs, plan.BlockStorageVolumeIDs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, volumeID := range added {
			if err := attachBlockStorageVolume(ctx, client, id, volumeID); err != nil {
				resp.Diagnostics.AddError("unable to attach block storage volume "+volumeID, err.Error())
				return
			}
		}

		for _, volumeID := range removed {
			if err := detachBlockStorageVolume(ctx, client, volumeID); err != nil {
				resp.Diagnostics.AddError("unable to detach block storage volume "+volumeID, err.Error())
				return
			}
		}
	}

	if !plan.ElasticIPIDs.Equal(state.ElasticIPIDs) {
		added, removed, diags := stringSetChange(ctx, state.ElasticIPIDs, plan.ElasticIPIDs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, elasticIPID := range added {
			if err := attachElasticIP(ctx, client, id, elasticIPID); err != nil {
				resp.Diagnostics.AddError("unable to attach Elastic IP "+elasticIPID, err.Error())
				return
			}
		}

		for _, elasticIPID := range removed {
			err := wait(ctx, client)(client.DetachInstanceFromElasticIP(
				ctx,
				exoscale.UUID(elasticIPID),
				exoscale.DetachInstanceFromElasticIPRequest{Instance: &exoscale.InstanceTarget{ID: id}},
			))
			if err != nil && !errors.Is(err, exoscale.ErrNotFound) {
				resp.Diagnostics.AddError("unable to detach Elastic IP "+elasticIPID, err.Error())
				return
			}
		}
	}

	if !plan.NetworkInterface.Equal(state.NetworkInterface) {
		planned, diags := networkInterfaces(ctx, plan.NetworkInterface)
		resp.Diagnostics.Append(diags...)
		current, diags := networkInterfaces(ctx, state.NetworkInterface)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// An interface whose IP address changes is detached, then attached again.
		for _, nif := range current {
			if slices.ContainsFunc(planned, func(p NetworkInterfaceModel) bool {
				return findNetworkInterface([]NetworkInterfaceModel{nif}, p) != nil
			}) {
				continue
			}

			err := wait(ctx, client)(client.DetachInstanceFromPrivateNetwork(
				ctx,
				exoscale.UUID(nif.NetworkID.ValueString()),
				exoscale.DetachInstanceFromPrivateNetworkRequest{Instance: &exoscale.Instance{ID: id}},
			))
			if err != nil && !errors.Is(err, exoscale.ErrNotFound) {
				resp.Diagnostics.AddError("unable to detach Private Network "+nif.NetworkID.ValueString(), err.Error())
				return
			}
		}

		for _, nif := range planned {
			if findNetworkInterface(current, nif) != nil {
				continue
			}

			if err := attachPrivateNetwork(ctx, client, id, nif); err != nil {
				resp.Diagnostics.AddError("unable to attach Private Network "+nif.NetworkID.ValueString(), err.Error())
				return
			}
		}
	}

	if !plan.SecurityGroupIDs.Equal(state.SecurityGroupIDs) {
		added, removed, diags := stringSetChange(ctx, state.SecurityGroupIDs, plan.SecurityGroupIDs)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, securityGroupID := range added {
			err := wait(ctx, client)(client.AttachInstanceToSecurityGroup(
				ctx,
				exoscale.UUID(securityGroupID),
				exoscale.AttachInstanceToSecurityGroupRequest{Instance: &exoscale.Instance{ID: id}},
			))
			if err != nil {
				resp.Diagnostics.AddError("unable to attach Security Group "+securityGroupID, err.Error())
				return
			}
		}

		for _, securityGroupID := range removed {
			err := wait(ctx, client)(client.DetachInstanceFromSecurityGroup(
				ctx,
				exoscale.UUID(securityGroupID),
				exoscale.DetachInstanceFromSecurityGroupRequest{Instance: &exoscale.Instance{ID: id}},
			))
			if err != nil && !errors.Is(err, exoscale.ErrNotFound) {
				resp.Diagnostics.AddError("unable to detach Security Group "+securityGroupID, err.Error())
				return
			}
		}
	}

	var (
		// With no `state` in the configuration, the instance is left in the
		// state it was found in.
		wantedState     = plan.State.ValueString()
		stateChanged    = !plan.State.Equal(state.State)
		diskSizeChanged = !plan.DiskSize.Equal(state.DiskSize)
		typeChanged     = !strings.EqualFold(plan.Type.ValueString(), state.Type.ValueString())
		tpmChanged      = !plan.EnableTPM.Equal(state.EnableTPM)
	)

	if stateChanged || diskSizeChanged || typeChanged || tpmChanged {
		// Check if size is below current size to prevent uneeded stop as API will prevent the scale operation
		if diskSizeChanged && instance.DiskSize > plan.DiskSize.ValueInt64() {
			resp.Diagnostics.AddError(
				"invalid value",
				fmt.Sprintf("unable to scale down the disk size, use size > %v", instance.DiskSize),
			)
			return
		}

		if tpmChanged && !plan.EnableTPM.ValueBool() {
			resp.Diagnostics.AddError("invalid value", "TPM can't be disabled")
			return
		}

		// Refresh instance state, since it could have changed since when we
		// first requested it above.
		instance, err = client.GetInstance(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("API returned an error while fetching instance", err.Error())
			return
		}

		// Compute instance scaling/disk resizing/TPM enabling API operations
		// requires the instance to be stopped.
		shouldStop := instance.State == exoscale.InstanceStateRunning &&
			(wantedState == string(exoscale.InstanceStateStopped) || diskSizeChanged || typeChanged || tpmChanged)
		if shouldStop {
			if err := wait(ctx, client)(client.StopInstance(ctx, id)); err != nil {
				resp.Diagnostics.AddError("unable to stop instance", err.Error())
				return
			}
		}

		if diskSizeChanged {
			err := wait(ctx, client)(client.ResizeInstanceDisk(
				ctx,
				id,
				exoscale.ResizeInstanceDiskRequest{DiskSize: plan.DiskSize.ValueInt64()},
			))
			if err != nil {
				resp.Diagnostics.AddError("unable to resize disk", err.Error())
				return
			}
		}

		if typeChanged {
			instanceType, err := utils.FindInstanceTypeByNameV3(ctx, client, plan.Type.ValueString())
			if err != nil {
				resp.Diagnostics.AddError("unable to retrieve instance type", err.Error())
				return
			}
			err = wait(ctx, client)(client.ScaleInstance(ctx, id, exoscale.ScaleInstanceRequest{InstanceType: instanceType}))
			if err != nil {
				resp.Diagnostics.AddError("unable to scale instance", err.Error())
				return
			}
		}

		if tpmChanged {
			if err := wait(ctx, client)(client.EnableTpm(ctx, id)); err != nil {
				resp.Diagnostics.AddError("failed to enable TPM", err.Error())
				return
			}
		}

		shouldStart := wantedState == string(exoscale.InstanceStateRunning) &&
			(shouldStop || instance.State != exoscale.InstanceStateRunning)
		if shouldStart {
			if err := wait(ctx, client)(client.StartInstance(ctx, id, exoscale.StartInstanceRequest{})); err != nil {
				resp.Diagnostics.AddError("unable to start instance", err.Error())
				return
			}
		}
	}

	// as we do not have a `get-instance-protection` API call,
	// the tf state of the `destroy_protected` field cannot be reconciled
	// and we cannot rely on a change of the attribute to detect one.
	// Therefore we simply apply what the practitioner configured
	// If the field is absent, the protection will be removed
	if plan.DestroyProtected.ValueBool() {
		if err := wait(ctx, client)(client.AddInstanceProtection(ctx, id)); err != nil {
			resp.Diagnostics.AddError("unable to make instance destroy protected", err.Error())
			return
		}
	} else {
		if err := wait(ctx, client)(client.RemoveInstanceProtection(ctx, id)); err != nil {
			resp.Diagnostics.AddError("unable to remove destroy protection from instance", err.Error())
			return
		}
	}

	instance, err = client.GetInstance(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching the updated instance", err.Error())
		return
	}

	resp.Diagnostics.Append(applyInstanceComputed(ctx, client, &plan, instance)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "resource update done", map[string]any{"id": plan.ID})
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

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(state.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	err = wait(ctx, client)(client.DeleteReverseDNSInstance(ctx, id))
	if err != nil && !errors.Is(err, exoscale.ErrNotFound) {
		resp.Diagnostics.AddError("unable to delete Reverse DNS record", err.Error())
		return
	}

	err = wait(ctx, client)(client.DeleteInstance(ctx, id))
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("unable to delete instance", err.Error())
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]any{"id": state.ID})
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, "@")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"unexpected import identifier",
			`Expected import identifier with format: id@zone. Got: "`+req.ID+`"`,
		)
		return
	}

	id, err := exoscale.ParseUUID(idParts[0])
	if err != nil {
		resp.Diagnostics.AddError("unable to parse ID", err.Error())
		return
	}

	zone := idParts[1]
	if !slices.Contains(config.Zones, zone) {
		resp.Diagnostics.AddError("invalid value", "zone must be a valid exoscale zone")
		return
	}

	// Set timeouts (quirk https://github.com/hashicorp/terraform-plugin-framework-timeouts/issues/46)
	var t timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &t)...)
	if resp.Diagnostics.HasError() {
		return
	}

	nifs, diags := networkInterfaceSet(ctx, nil)
	resp.Diagnostics.Append(diags...)

	resp.Diagnostics.Append(resp.State.Set(ctx, &ResourceModel{
		ID:                    types.StringValue(id.String()),
		Zone:                  types.StringValue(zone),
		AntiAffinityGroupIDs:  types.SetNull(types.StringType),
		BlockStorageVolumeIDs: types.SetNull(types.StringType),
		ElasticIPIDs:          types.SetNull(types.StringType),
		Labels:                types.MapNull(types.StringType),
		NetworkInterface:      nifs,
		PrivateNetworkIDs:     types.SetNull(types.StringType),
		SSHKeys:               types.SetNull(types.StringType),
		SecurityGroupIDs:      types.SetNull(types.StringType),
		Timeouts:              t,
	})...)

	tflog.Trace(ctx, "resource imported", map[string]any{"id": id.String()})
}

// wait returns a function waiting for the success of an operation, to be
// wrapped around the API call starting it.
func wait(ctx context.Context, client *exoscale.Client) func(*exoscale.Operation, error) error {
	return func(operation *exoscale.Operation, err error) error {
		if err != nil {
			return err
		}

		_, err = client.Wait(ctx, operation, exoscale.OperationStateSuccess)

		return err
	}
}

// stringSetChange returns the values added to and removed from a set of
// strings between the state and the plan.
func stringSetChange(ctx context.Context, state, plan types.Set) (added, removed []string, diags diag.Diagnostics) {
	old, dg := stringSetValues(ctx, state)
	diags.Append(dg...)
	cur, dg := stringSetValues(ctx, plan)
	diags.Append(dg...)

	added, removed = stringSetDiff(old, cur)

	return added, removed, diags
}

func attachElasticIP(ctx context.Context, client *exoscale.Client, id exoscale.UUID, elasticIPID string) error {
	return wait(ctx, client)(client.AttachInstanceToElasticIP(
		ctx,
		exoscale.UUID(elasticIPID),
		exoscale.AttachInstanceToElasticIPRequest{Instance: &exoscale.InstanceTarget{ID: id}},
	))
}

func attachPrivateNetwork(ctx context.Context, client *exoscale.Client, id exoscale.UUID, nif NetworkInterfaceModel) error {
	return wait(ctx, client)(client.AttachInstanceToPrivateNetwork(
		ctx,
		exoscale.UUID(nif.NetworkID.ValueString()),
		exoscale.AttachInstanceToPrivateNetworkRequest{
			Instance: &exoscale.AttachInstanceToPrivateNetworkRequestInstance{ID: id},
			// The IP address is unknown or null unless the configuration sets one.
			IP: net.ParseIP(nif.IPAddress.ValueString()),
		},
	))
}

func attachBlockStorageVolume(ctx context.Context, client *exoscale.Client, id exoscale.UUID, volumeID string) error {
	volume, err := exoscale.ParseUUID(volumeID)
	if err != nil {
		return fmt.Errorf("unable to parse block storage volume ID: %w", err)
	}

	return wait(ctx, client)(client.AttachBlockStorageVolumeToInstance(
		ctx,
		volume,
		exoscale.AttachBlockStorageVolumeToInstanceRequest{Instance: &exoscale.InstanceTarget{ID: id}},
	))
}

func detachBlockStorageVolume(ctx context.Context, client *exoscale.Client, volumeID string) error {
	volume, err := exoscale.ParseUUID(volumeID)
	if err != nil {
		return fmt.Errorf("unable to parse block storage volume ID: %w", err)
	}

	err = wait(ctx, client)(client.DetachBlockStorageVolume(ctx, volume))
	if err != nil {
		// The volume was likely already deleted as part of its regular
		// deletion process.
		if errors.Is(err, exoscale.ErrNotFound) {
			return nil
		}

		// Ideally we would have a custom error defined in OpenAPI spec & egoscale.
		// For now we just check the error text.
		if strings.HasSuffix(err.Error(), "Volume not attached") {
			return nil
		}
	}

	return err
}

// nullUnknown returns the model with the values still unknown set to null, so
// that it can be saved in the state when Create fails half-way.
func nullUnknown(ctx context.Context, model ResourceModel) *ResourceModel {
	for _, v := range []*types.String{
		&model.CreatedAt,
		&model.IPv6Address,
		&model.MACAddress,
		&model.PublicIPAddress,
		&model.State,
	} {
		if v.IsUnknown() {
			*v = types.StringNull()
		}
	}

	if model.PrivateNetworkIDs.IsUnknown() {
		model.PrivateNetworkIDs = types.SetNull(types.StringType)
	}

	nifs, _ := networkInterfaces(ctx, model.NetworkInterface)
	for i := range nifs {
		if nifs[i].IPAddress.IsUnknown() {
			nifs[i].IPAddress = types.StringNull()
		}
		if nifs[i].MACAddress.IsUnknown() {
			nifs[i].MACAddress = types.StringNull()
		}
	}
	model.NetworkInterface, _ = networkInterfaceSet(ctx, nifs)

	return &model
}
