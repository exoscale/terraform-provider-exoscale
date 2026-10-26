package instance_pool

import (
	"context"
	"errors"
	"regexp"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	DefaultInstancePrefix = "pool"
)

const markdownDescriptionResource = `Manage Exoscale [Instance Pools](https://community.exoscale.com/product/compute/instances/how-to/instance-pools/).

Corresponding data sources: [exoscale_instance_pool](../data-sources/instance_pool.md), [exoscale_instance_pool_list](../data-sources/instance_pool_list.md).`

var _ resource.Resource = (*Resource)(nil)
var _ resource.ResourceWithImportState = (*Resource)(nil)
var _ resource.ResourceWithModifyPlan = (*Resource)(nil)
var _ resource.ResourceWithUpgradeState = (*Resource)(nil)

type Resource struct {
	client *exoscale.Client
}

// NewResource creates an instance of Resource.
func NewResource() resource.Resource {
	return &Resource{}
}

// ResourceModel defines the exoscale_instance_pool resource data model.
type ResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	AffinityGroupIDs     types.Set    `tfsdk:"affinity_group_ids"`
	AntiAffinityGroupIDs types.Set    `tfsdk:"anti_affinity_group_ids"`
	DeployTargetID       types.String `tfsdk:"deploy_target_id"`
	Description          types.String `tfsdk:"description"`
	DiskSize             types.Int64  `tfsdk:"disk_size"`
	ElasticIPIDs         types.Set    `tfsdk:"elastic_ip_ids"`
	InstancePrefix       types.String `tfsdk:"instance_prefix"`
	InstanceType         types.String `tfsdk:"instance_type"`
	Instances            types.Set    `tfsdk:"instances"`
	IPv6                 types.Bool   `tfsdk:"ipv6"`
	KeyPair              types.String `tfsdk:"key_pair"`
	Labels               types.Map    `tfsdk:"labels"`
	MinAvailable         types.Int64  `tfsdk:"min_available"`
	Name                 types.String `tfsdk:"name"`
	NetworkIDs           types.Set    `tfsdk:"network_ids"`
	SecurityGroupIDs     types.Set    `tfsdk:"security_group_ids"`
	Size                 types.Int64  `tfsdk:"size"`
	State                types.String `tfsdk:"state"`
	TemplateID           types.String `tfsdk:"template_id"`
	UserData             types.String `tfsdk:"user_data"`
	VirtualMachines      types.Set    `tfsdk:"virtual_machines"`
	Zone                 types.String `tfsdk:"zone"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// Metadata specifies the resource name.
func (r *Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_pool"
}

// Schema defines the resource attributes.
func (r *Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// Version 1: SDKv2 to framework migration, see UpgradeState.
		Version: 1,

		Description:         "Manage Exoscale Instance Pools.",
		MarkdownDescription: markdownDescriptionResource,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The ID of this resource.",
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"affinity_group_ids": schema.SetAttribute{
				Description:         "A list of exoscale_anti_affinity_group (IDs; may only be set at creation time).",
				MarkdownDescription: "A list of [exoscale_anti_affinity_group](./anti_affinity_group.md) (IDs; may only be set at creation time).",
				DeprecationMessage:  "Use anti_affinity_group_ids instead.",
				ElementType:         types.StringType,
				Optional:            true,
				Validators: []validator.Set{
					setvalidator.ConflictsWith(path.MatchRoot("anti_affinity_group_ids")),
				},
			},
			"anti_affinity_group_ids": schema.SetAttribute{
				Description:         "A list of exoscale_anti_affinity_group (IDs; may only be set at creation time).",
				MarkdownDescription: "A list of [exoscale_anti_affinity_group](./anti_affinity_group.md) (IDs; may only be set at creation time).",
				ElementType:         types.StringType,
				Optional:            true,
				Validators: []validator.Set{
					setvalidator.ConflictsWith(path.MatchRoot("affinity_group_ids")),
				},
			},
			"deploy_target_id": schema.StringAttribute{
				Description:         "A deploy target ID.",
				MarkdownDescription: "A deploy target ID.",
				Optional:            true,
			},
			"description": schema.StringAttribute{
				Description:         "A free-form text describing the pool.",
				MarkdownDescription: "A free-form text describing the pool.",
				Optional:            true,
			},
			"disk_size": schema.Int64Attribute{
				Description:         "The managed instances disk size (GiB).",
				MarkdownDescription: "The managed instances disk size (GiB).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"elastic_ip_ids": schema.SetAttribute{
				Description:         "A list of exoscale_elastic_ip (IDs).",
				MarkdownDescription: "A list of [exoscale_elastic_ip](./elastic_ip.md) (IDs).",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"instance_prefix": schema.StringAttribute{
				Description:         "The string used to prefix managed instances name (default: 'pool').",
				MarkdownDescription: "The string used to prefix managed instances name (default: `pool`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(DefaultInstancePrefix),
			},
			"instance_type": schema.StringAttribute{
				Description:         "The managed compute instances type ('<family>.<size>', e.g. 'standard.medium'; use the Exoscale CLI - 'exo compute instance-type list' - for the list of available types).",
				MarkdownDescription: "The managed compute instances type (`<family>.<size>`, e.g. `standard.medium`; use the [Exoscale CLI](https://github.com/exoscale/cli/) - `exo compute instance-type list` - for the list of available types).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					ignoreCase(),
				},
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`\.`), `must be in the "FAMILY.SIZE" format`),
				},
			},
			"instances": schema.SetNestedAttribute{
				Description:         "The list of managed instances. Structure is documented below.",
				MarkdownDescription: "The list of managed instances. Structure is documented below.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description:         "The instance ID.",
							MarkdownDescription: "The instance ID.",
							Computed:            true,
						},
						"ipv6_address": schema.StringAttribute{
							Description:         "The instance (main network interface) IPv6 address.",
							MarkdownDescription: "The instance (main network interface) IPv6 address.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							Description:         "The instance name.",
							MarkdownDescription: "The instance name.",
							Computed:            true,
						},
						"public_ip_address": schema.StringAttribute{
							Description:         "The instance (main network interface) IPv4 address.",
							MarkdownDescription: "The instance (main network interface) IPv4 address.",
							Computed:            true,
						},
					},
				},
			},
			"ipv6": schema.BoolAttribute{
				Description:         "Enable IPv6 on managed instances (boolean; default: 'false').",
				MarkdownDescription: "Enable IPv6 on managed instances (boolean; default: `false`).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"key_pair": schema.StringAttribute{
				Description:         "The exoscale_ssh_key (name) to authorize in the managed instances.",
				MarkdownDescription: "The [exoscale_ssh_key](./ssh_key.md) (name) to authorize in the managed instances.",
				Optional:            true,
			},
			"labels": schema.MapAttribute{
				Description:         "A map of key/value labels.",
				MarkdownDescription: "A map of key/value labels.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"min_available": schema.Int64Attribute{
				Description:         "Minimum number of running Instances.",
				MarkdownDescription: "Minimum number of running Instances.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.AtLeast(0),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The instance pool name.",
				MarkdownDescription: "The instance pool name.",
				Required:            true,
			},
			"network_ids": schema.SetAttribute{
				Description:         "A list of exoscale_private_network (IDs).",
				MarkdownDescription: "A list of [exoscale_private_network](./private_network.md) (IDs).",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"security_group_ids": schema.SetAttribute{
				Description:         "A list of exoscale_security_group (IDs).",
				MarkdownDescription: "A list of [exoscale_security_group](./security_group.md) (IDs).",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"size": schema.Int64Attribute{
				Description:         "The number of managed instances.",
				MarkdownDescription: "The number of managed instances.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"state": schema.StringAttribute{
				Description:         "The instance pool state.",
				MarkdownDescription: "The instance pool state.",
				Computed:            true,
			},
			"template_id": schema.StringAttribute{
				Description:         "The exoscale_template (ID) to use when creating the managed instances.",
				MarkdownDescription: "The [exoscale_template](../data-sources/template.md) (ID) to use when creating the managed instances.",
				Required:            true,
			},
			"user_data": schema.StringAttribute{
				Description:         "cloud-init configuration to apply to the managed instances.",
				MarkdownDescription: "[cloud-init](http://cloudinit.readthedocs.io/) configuration to apply to the managed instances.",
				Optional:            true,
				PlanModifiers: []planmodifier.String{
					ignoreUserDataEncoding(),
				},
			},
			"virtual_machines": schema.SetAttribute{
				Description:         "The list of managed instances (IDs). Please use the 'instances.*.id' attribute instead.",
				MarkdownDescription: "The list of managed instances (IDs). Please use the `instances.*.id` attribute instead.",
				DeprecationMessage:  "Use the instances exported attribute instead.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"zone": schema.StringAttribute{
				Description:         "❗ The Exoscale Zone name.",
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
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

// resourceModelV0 is the state written by the SDKv2 implementation.
type resourceModelV0 struct {
	ResourceModel
	ServiceOffering types.String `tfsdk:"service_offering"`
}

func (r *Resource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		// SDKv2 to framework migration: the SDKv2 wrote "" / {} for an unset
		// description and labels, they are normalised to null here, once,
		// rather than on every plan. The deprecated service_offering is
		// dropped. Every other attribute is compatible as-is (`instances` was
		// a block, with the same type).
		0: {
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id":                      schema.StringAttribute{Optional: true, Computed: true},
					"affinity_group_ids":      schema.SetAttribute{ElementType: types.StringType, Optional: true},
					"anti_affinity_group_ids": schema.SetAttribute{ElementType: types.StringType, Optional: true},
					"deploy_target_id":        schema.StringAttribute{Optional: true},
					"description":             schema.StringAttribute{Optional: true},
					"disk_size":               schema.Int64Attribute{Optional: true, Computed: true},
					"elastic_ip_ids":          schema.SetAttribute{ElementType: types.StringType, Optional: true},
					"instance_prefix":         schema.StringAttribute{Optional: true},
					"instance_type":           schema.StringAttribute{Optional: true, Computed: true},
					"ipv6":                    schema.BoolAttribute{Optional: true},
					"key_pair":                schema.StringAttribute{Optional: true},
					"labels":                  schema.MapAttribute{ElementType: types.StringType, Optional: true},
					"min_available":           schema.Int64Attribute{Optional: true, Computed: true},
					"name":                    schema.StringAttribute{Required: true},
					"network_ids":             schema.SetAttribute{ElementType: types.StringType, Optional: true},
					"security_group_ids":      schema.SetAttribute{ElementType: types.StringType, Optional: true},
					"service_offering":        schema.StringAttribute{Optional: true, Computed: true},
					"size":                    schema.Int64Attribute{Required: true},
					"state":                   schema.StringAttribute{Computed: true},
					"template_id":             schema.StringAttribute{Required: true},
					"user_data":               schema.StringAttribute{Optional: true},
					"virtual_machines":        schema.SetAttribute{ElementType: types.StringType, Optional: true, Computed: true},
					"zone":                    schema.StringAttribute{Required: true},
				},
				Blocks: map[string]schema.Block{
					"instances": schema.SetNestedBlock{
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"id":                schema.StringAttribute{Optional: true},
								"ipv6_address":      schema.StringAttribute{Computed: true},
								"name":              schema.StringAttribute{Optional: true},
								"public_ip_address": schema.StringAttribute{Computed: true},
							},
						},
					},
					"timeouts": timeouts.BlockAll(ctx),
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var prior resourceModelV0

				resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
				if resp.Diagnostics.HasError() {
					return
				}

				state := prior.ResourceModel

				if state.Description.ValueString() == "" {
					state.Description = types.StringNull()
				}
				if len(state.Labels.Elements()) == 0 {
					state.Labels = types.MapNull(types.StringType)
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			},
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
// being planned as unknown: the managed instances only change with the size
// of the pool.
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

	if plan.Size.Equal(state.Size) {
		if plan.Instances.IsUnknown() {
			plan.Instances = state.Instances
		}
		if plan.VirtualMachines.IsUnknown() {
			plan.VirtualMachines = state.VirtualMachines
		}
	}

	// A configuration differing from the state only by case or encoding (see
	// ignoreCase, ignoreUserDataEncoding) is planned as no change, but still
	// gets the pool state planned as unknown: keep it.
	if plan.State.IsUnknown() {
		unchanged := plan
		unchanged.State = state.State
		resp.Diagnostics.Append(resp.Plan.Set(ctx, &unchanged)...)
		if resp.Diagnostics.HasError() || resp.Plan.Raw.Equal(req.State.Raw) {
			return
		}
	}

	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
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

	request := exoscale.CreateInstancePoolRequest{
		Name:           plan.Name.ValueString(),
		Description:    plan.Description.ValueString(),
		DiskSize:       plan.DiskSize.ValueInt64(),
		InstancePrefix: plan.InstancePrefix.ValueString(),
		Ipv6Enabled:    exoscale.Ptr(plan.IPv6.ValueBool()),
		MinAvailable:   plan.MinAvailable.ValueInt64(),
		Size:           plan.Size.ValueInt64(),
		Template:       &exoscale.Template{ID: exoscale.UUID(plan.TemplateID.ValueString())},
	}

	if v := plan.DeployTargetID.ValueString(); v != "" {
		request.DeployTarget = &exoscale.DeployTarget{ID: exoscale.UUID(v)}
	}

	if v := plan.KeyPair.ValueString(); v != "" {
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

	antiAffinityGroups, diags := antiAffinityGroupIDs(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	securityGroups, diags := idValues(ctx, plan.SecurityGroupIDs)
	resp.Diagnostics.Append(diags...)
	privateNetworks, diags := idValues(ctx, plan.NetworkIDs)
	resp.Diagnostics.Append(diags...)
	elasticIPs, diags := idValues(ctx, plan.ElasticIPIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request.AntiAffinityGroups = utils.AntiAffinityGroupIDsToAntiAffinityGroups(antiAffinityGroups)
	request.SecurityGroups = utils.SecurityGroupIDsToSecurityGroups(securityGroups)
	request.PrivateNetworks = utils.PrivateNetworkIDsToPrivateNetworks(privateNetworks)
	request.ElasticIPS = utils.ElasticIPIDsToElasticIPs(elasticIPs)

	request.InstanceType, err = utils.FindInstanceTypeByNameV3(ctx, client, plan.InstanceType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to retrieve instance type", err.Error())
		return
	}

	if v := plan.UserData.ValueString(); v != "" {
		userData, _, err := utils.EncodeUserData(v)
		if err != nil {
			resp.Diagnostics.AddError("unable to encode user data", err.Error())
			return
		}
		request.UserData = userData
	}

	operation, err := client.CreateInstancePool(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error when creating instance pool", err.Error())
		return
	}

	operation, err = client.Wait(ctx, operation, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("create instance pool operation failed", err.Error())
		return
	}

	id := operation.Reference.ID
	plan.ID = types.StringValue(id.String())

	// The pool exists from here on: save it in the state right away, so that
	// if one of the steps below fails it is kept there (Terraform marks it as
	// tainted) rather than left behind unmanaged.
	resp.Diagnostics.Append(resp.State.Set(ctx, nullUnknown(plan))...)
	if resp.Diagnostics.HasError() {
		return
	}

	pool, err := client.GetInstancePool(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching the created instance pool", err.Error())
		return
	}

	resp.Diagnostics.Append(applyPoolComputed(ctx, client, &plan, pool)...)
	if resp.Diagnostics.HasError() {
		return
	}

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

	pool, err := client.GetInstancePool(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("API returned an error while fetching instance pool", err.Error())
		return
	}

	resp.Diagnostics.Append(applyPool(ctx, client, &state, pool)...)
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

	var (
		update  bool
		request exoscale.UpdateInstancePoolRequest
		resets  []exoscale.ResetInstancePoolFieldField
	)

	// The anti-affinity groups, deploy target, Elastic IPs, Private Networks,
	// Security Groups and SSH key are always sent: the API clears the ones
	// left out (null).
	antiAffinityGroups, diags := antiAffinityGroupIDs(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	securityGroups, diags := idValues(ctx, plan.SecurityGroupIDs)
	resp.Diagnostics.Append(diags...)
	privateNetworks, diags := idValues(ctx, plan.NetworkIDs)
	resp.Diagnostics.Append(diags...)
	elasticIPs, diags := idValues(ctx, plan.ElasticIPIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	request.AntiAffinityGroups = utils.AntiAffinityGroupIDsToAntiAffinityGroups(antiAffinityGroups)
	request.SecurityGroups = utils.SecurityGroupIDsToSecurityGroups(securityGroups)
	request.PrivateNetworks = utils.PrivateNetworkIDsToPrivateNetworks(privateNetworks)
	request.ElasticIPS = utils.ElasticIPIDsToElasticIPs(elasticIPs)
	request.SSHKey = &exoscale.SSHKey{Name: plan.KeyPair.ValueString()}
	if v := plan.DeployTargetID.ValueString(); v != "" {
		request.DeployTarget = &exoscale.DeployTarget{ID: exoscale.UUID(v)}
	}

	if !plan.AntiAffinityGroupIDs.Equal(state.AntiAffinityGroupIDs) ||
		!plan.AffinityGroupIDs.Equal(state.AffinityGroupIDs) ||
		!plan.SecurityGroupIDs.Equal(state.SecurityGroupIDs) ||
		!plan.NetworkIDs.Equal(state.NetworkIDs) ||
		!plan.ElasticIPIDs.Equal(state.ElasticIPIDs) ||
		plan.KeyPair.ValueString() != state.KeyPair.ValueString() ||
		plan.DeployTargetID.ValueString() != state.DeployTargetID.ValueString() {
		update = true
	}

	if plan.Description.ValueString() != state.Description.ValueString() {
		if v := plan.Description.ValueString(); v == "" {
			resets = append(resets, exoscale.ResetInstancePoolFieldFieldDescription)
		} else {
			update = true
			request.Description = v
		}
	}

	if !plan.DiskSize.IsUnknown() && !plan.DiskSize.Equal(state.DiskSize) {
		update = true
		request.DiskSize = plan.DiskSize.ValueInt64()
	}

	if !plan.InstancePrefix.Equal(state.InstancePrefix) {
		update = true
		request.InstancePrefix = exoscale.Ptr(plan.InstancePrefix.ValueString())
	}

	if !plan.IPv6.Equal(state.IPv6) {
		update = true
		request.Ipv6Enabled = exoscale.Ptr(plan.IPv6.ValueBool())
	}

	if !plan.Labels.Equal(state.Labels) {
		if len(plan.Labels.Elements()) == 0 {
			// An empty map is left out of the request (omitempty).
			resets = append(resets, exoscale.ResetInstancePoolFieldFieldLabels)
		} else {
			labels := exoscale.Labels{}
			resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			update = true
			request.Labels = labels
		}
	}

	if !plan.Name.Equal(state.Name) {
		update = true
		request.Name = plan.Name.ValueString()
	}

	if !strings.EqualFold(plan.InstanceType.ValueString(), state.InstanceType.ValueString()) {
		instanceType, err := utils.FindInstanceTypeByNameV3(ctx, client, plan.InstanceType.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("unable to retrieve instance type", err.Error())
			return
		}
		update = true
		request.InstanceType = &exoscale.InstanceType{ID: instanceType.ID}
	}

	if !plan.TemplateID.Equal(state.TemplateID) {
		update = true
		request.Template = &exoscale.Template{ID: exoscale.UUID(plan.TemplateID.ValueString())}
	}

	if plan.UserData.ValueString() != state.UserData.ValueString() {
		if v := plan.UserData.ValueString(); v == "" {
			resets = append(resets, exoscale.ResetInstancePoolFieldFieldUserData)
		} else {
			userData, _, err := utils.EncodeUserData(v)
			if err != nil {
				resp.Diagnostics.AddError("unable to encode user data", err.Error())
				return
			}
			update = true
			request.UserData = &userData
		}
	}

	if !plan.MinAvailable.IsUnknown() && !plan.MinAvailable.Equal(state.MinAvailable) {
		update = true
		request.MinAvailable = exoscale.Ptr(plan.MinAvailable.ValueInt64())
	}

	if update {
		if err := wait(ctx, client)(client.UpdateInstancePool(ctx, id, request)); err != nil {
			resp.Diagnostics.AddError("unable to update instance pool", err.Error())
			return
		}
	}

	for _, field := range resets {
		if err := wait(ctx, client)(client.ResetInstancePoolField(ctx, id, field)); err != nil {
			resp.Diagnostics.AddError("unable to reset instance pool "+string(field), err.Error())
			return
		}
	}

	if !plan.Size.Equal(state.Size) {
		err := wait(ctx, client)(client.ScaleInstancePool(
			ctx,
			id,
			exoscale.ScaleInstancePoolRequest{Size: plan.Size.ValueInt64()},
		))
		if err != nil {
			resp.Diagnostics.AddError("unable to scale instance pool", err.Error())
			return
		}
	}

	pool, err := client.GetInstancePool(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching the updated instance pool", err.Error())
		return
	}

	resp.Diagnostics.Append(applyPoolComputed(ctx, client, &plan, pool)...)
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

	err = wait(ctx, client)(client.DeleteInstancePool(ctx, id))
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("unable to delete instance pool", err.Error())
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]any{"id": state.ID})
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	utils.ImportStatePassthroughZonedID(ctx, req, resp)
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

// nullUnknown returns the model with the values still unknown set to null, so
// that it can be saved in the state when Create fails half-way.
func nullUnknown(model ResourceModel) *ResourceModel {
	if model.State.IsUnknown() {
		model.State = types.StringNull()
	}

	for _, v := range []*types.Int64{
		&model.DiskSize,
		&model.MinAvailable,
	} {
		if v.IsUnknown() {
			*v = types.Int64Null()
		}
	}

	if model.Instances.IsUnknown() {
		model.Instances = types.SetNull(instanceObjectType)
	}
	if model.VirtualMachines.IsUnknown() {
		model.VirtualMachines = types.SetNull(types.StringType)
	}

	return &model
}
