package sks_cluster

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const (
	defaultSKSNodepoolDiskSize       int64 = 50
	defaultSKSNodepoolInstancePrefix       = "pool"

	sksNodepoolAddonStorageLVM = "storage-lvm"
)

const markdownDescriptionNodepoolResource = `Manage Exoscale [Scalable Kubernetes Service (SKS)](https://community.exoscale.com/product/compute/containers/) Node Pools.

Corresponding data source: [exoscale_sks_nodepool](../data-sources/sks_nodepool.md).`

var _ resource.Resource = &ResourceNodepool{}
var _ resource.ResourceWithImportState = &ResourceNodepool{}
var _ resource.ResourceWithUpgradeState = &ResourceNodepool{}

type ResourceNodepool struct {
	client *exoscale.Client
}

func NewResourceNodepool() resource.Resource {
	return &ResourceNodepool{}
}

type kubeletGCModel struct {
	MinAge        types.String `tfsdk:"min_age"`
	HighThreshold types.Int64  `tfsdk:"high_threshold"`
	LowThreshold  types.Int64  `tfsdk:"low_threshold"`
}

func kubeletGCAttrTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"min_age":        types.StringType,
		"high_threshold": types.Int64Type,
		"low_threshold":  types.Int64Type,
	}
}

type ResourceNodepoolModel struct {
	ID                   types.String `tfsdk:"id"`
	ClusterID            types.String `tfsdk:"cluster_id"`
	Zone                 types.String `tfsdk:"zone"`
	AntiAffinityGroupIDs types.Set    `tfsdk:"anti_affinity_group_ids"`
	CreatedAt            types.String `tfsdk:"created_at"`
	DeployTargetID       types.String `tfsdk:"deploy_target_id"`
	Description          types.String `tfsdk:"description"`
	DiskSize             types.Int64  `tfsdk:"disk_size"`
	InstancePoolID       types.String `tfsdk:"instance_pool_id"`
	InstancePrefix       types.String `tfsdk:"instance_prefix"`
	InstanceType         types.String `tfsdk:"instance_type"`
	KubeletMaxPods       types.Int64  `tfsdk:"kubelet_max_pods"`
	Labels               types.Map    `tfsdk:"labels"`
	Name                 types.String `tfsdk:"name"`
	NvidiaMigProfile     types.String `tfsdk:"nvidia_mig_profile"`
	IPv6                 types.Bool   `tfsdk:"ipv6"`
	PrivateNetworkIDs    types.Set    `tfsdk:"private_network_ids"`
	SecurityGroupIDs     types.Set    `tfsdk:"security_group_ids"`
	Size                 types.Int64  `tfsdk:"size"`
	State                types.String `tfsdk:"state"`
	StorageLVM           types.Bool   `tfsdk:"storage_lvm"`
	Taints               types.Map    `tfsdk:"taints"`
	TemplateID           types.String `tfsdk:"template_id"`
	Version              types.String `tfsdk:"version"`

	KubeletImageGC types.Object `tfsdk:"kubelet_image_gc"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *ResourceNodepool) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sks_nodepool"
}

func (r *ResourceNodepool) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// Version 1: kubelet_image_gc moved from an SDKv2 TypeSet(MaxItems: 1)
		// (array-shaped in state) to a framework SingleNestedAttribute
		// (object-shaped). See UpgradeState below.
		Version: 1,

		Description:         "Manage Exoscale Scalable Kubernetes Service (SKS) Node Pools.",
		MarkdownDescription: markdownDescriptionNodepoolResource,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "The SKS node pool ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_id": schema.StringAttribute{
				Description:         "❗ The parent exoscale_sks_cluster ID.",
				MarkdownDescription: "❗ The parent [exoscale_sks_cluster](./sks_cluster.md) ID.",
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
			"anti_affinity_group_ids": schema.SetAttribute{
				Description:         "A list of exoscale_anti_affinity_group (IDs) to be attached to the managed instances.",
				MarkdownDescription: "A list of [exoscale_anti_affinity_group](./anti_affinity_group.md) (IDs) to be attached to the managed instances.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"created_at": schema.StringAttribute{
				Description:         "The pool creation date.",
				MarkdownDescription: "The pool creation date.",
				Computed:            true,
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
				Description:         "The managed instances disk size (GiB; default: '50').",
				MarkdownDescription: "The managed instances disk size (GiB; default: `50`).",
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(defaultSKSNodepoolDiskSize),
			},
			"instance_pool_id": schema.StringAttribute{
				Description:         "The underlying exoscale_instance_pool ID.",
				MarkdownDescription: "The underlying [exoscale_instance_pool](./instance_pool.md) ID.",
				Computed:            true,
			},
			"instance_prefix": schema.StringAttribute{
				Description:         "The string used to prefix the managed instances name (default 'pool').",
				MarkdownDescription: "The string used to prefix the managed instances name (default `pool`).",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(defaultSKSNodepoolInstancePrefix),
			},
			"instance_type": schema.StringAttribute{
				Description:         "The managed compute instances type ('<family>.<size>', e.g. 'standard.medium'; use the Exoscale CLI - 'exo compute instance-type list' - for the list of available types).",
				MarkdownDescription: "The managed compute instances type (`<family>.<size>`, e.g. `standard.medium`; use the [Exoscale CLI](https://github.com/exoscale/cli/) - `exo compute instance-type list` - for the list of available types).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`\.`), `invalid value, expected format "FAMILY.SIZE"`),
				},
			},
			"kubelet_image_gc": schema.SingleNestedAttribute{
				Description:         "Configuration for this nodepool's kubelet image garbage collector.",
				MarkdownDescription: "Configuration for this nodepool's kubelet image garbage collector.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
				Attributes: map[string]schema.Attribute{
					"min_age": schema.StringAttribute{
						Description:         "The minimum age for an unused image before it is garbage collected (k8s duration format, eg. 1h)",
						MarkdownDescription: "The minimum age for an unused image before it is garbage collected (k8s duration format, eg. 1h)",
						Optional:            true,
						Computed:            true,
					},
					"high_threshold": schema.Int64Attribute{
						Description:         "The percent of disk usage after which image garbage collection is always run",
						MarkdownDescription: "The percent of disk usage after which image garbage collection is always run",
						Optional:            true,
						Computed:            true,
					},
					"low_threshold": schema.Int64Attribute{
						Description:         "The percent of disk usage before which image garbage collection is never run",
						MarkdownDescription: "The percent of disk usage before which image garbage collection is never run",
						Optional:            true,
						Computed:            true,
					},
				},
			},
			"kubelet_max_pods": schema.Int64Attribute{
				Description:         "The maximum number of pods per node (default is 110).",
				MarkdownDescription: "The maximum number of pods per node (default is 110).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"labels": schema.MapAttribute{
				Description:         "A map of key/value labels.",
				MarkdownDescription: "A map of key/value labels.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"name": schema.StringAttribute{
				Description:         "The SKS node pool name.",
				MarkdownDescription: "The SKS node pool name.",
				Required:            true,
			},
			"nvidia_mig_profile": schema.StringAttribute{
				Description: "The NVIDIA Multi-Instance GPU (MIG) profile to enable on the managed GPUs. " +
					"The GPU family is inferred from 'instance_type': 'gpua30.*' accepts '2g.12gb', '1g.6gb+me', '1g.6gb', '2g.12gb+me', '4g.24gb'; " +
					"'gpurtx6000pro.*' accepts '1g.24gb-me', '1g.24gb', '2g.48gb-me', '2g.48gb', '4g.96gb+gfx', '1g.24gb+me', '2g.48gb+me.all', '1g.24gb+gfx', '1g.24gb+me.all', '4g.96gb', '2g.48gb+gfx'.",
				MarkdownDescription: "The NVIDIA [Multi-Instance GPU (MIG)](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/) profile to enable on the managed GPUs. " +
					"The GPU family is inferred from `instance_type`: `gpua30.*` accepts `2g.12gb`, `1g.6gb+me`, `1g.6gb`, `2g.12gb+me`, `4g.24gb`; " +
					"`gpurtx6000pro.*` accepts `1g.24gb-me`, `1g.24gb`, `2g.48gb-me`, `2g.48gb`, `4g.96gb+gfx`, `1g.24gb+me`, `2g.48gb+me.all`, `1g.24gb+gfx`, `1g.24gb+me.all`, `4g.96gb`, `2g.48gb+gfx`.",
				Optional: true,
			},
			"ipv6": schema.BoolAttribute{
				Description:         "Enable IPV6 for the nodepool nodes",
				MarkdownDescription: "Enable IPV6 for the nodepool nodes",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"private_network_ids": schema.SetAttribute{
				Description:         "A list of exoscale_private_network (IDs) to be attached to the managed instances.",
				MarkdownDescription: "A list of [exoscale_private_network](./private_network.md) (IDs) to be attached to the managed instances.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"security_group_ids": schema.SetAttribute{
				Description:         "A list of [exoscale_security_group](./security_group.md) (IDs) to be attached to the managed instances.",
				MarkdownDescription: "A list of exoscale_security_group (IDs) to be attached to the managed instances.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"size": schema.Int64Attribute{
				Required: true,
			},
			"state": schema.StringAttribute{
				Description:         "The current pool state.",
				MarkdownDescription: "The current pool state.",
				Computed:            true,
			},
			"storage_lvm": schema.BoolAttribute{
				Description:         "Create nodes with non-standard partitioning for persistent storage (requires min 100G of disk space) (may only be set at creation time).",
				MarkdownDescription: "Create nodes with non-standard partitioning for persistent storage (requires min 100G of disk space) (may only be set at creation time).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"taints": schema.MapAttribute{
				Description:         `A map of key/value Kubernetes taints ('taints = { <key> = "<value>:<effect>" }').`,
				MarkdownDescription: `A map of key/value Kubernetes [taints](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) ('taints = { <key> = "<value>:<effect>" }').`,
				ElementType:         types.StringType,
				Optional:            true,
			},
			"template_id": schema.StringAttribute{
				Description:         "The managed instances template ID.",
				MarkdownDescription: "The managed instances template ID.",
				Computed:            true,
			},
			"version": schema.StringAttribute{
				Description:         "The managed instances version.",
				MarkdownDescription: "The managed instances version.",
				Computed:            true,
			},
		},

		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *ResourceNodepool) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ResourceNodepool) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.SplitN(req.ID, "@", 2)
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"invalid import ID",
			fmt.Sprintf(`invalid ID %q, expected format "<CLUSTER-ID>/<NODEPOOL-ID>@<ZONE>"`, req.ID),
		)
		return
	}

	zone := idParts[1]
	if !in(config.Zones, zone) {
		resp.Diagnostics.AddError("invalid value", "zone must be a valid exoscale zone")
		return
	}

	clusterAndNodepool := strings.SplitN(idParts[0], "/", 2)
	if len(clusterAndNodepool) != 2 || clusterAndNodepool[0] == "" || clusterAndNodepool[1] == "" {
		resp.Diagnostics.AddError(
			"invalid import ID",
			fmt.Sprintf(`invalid ID %q, expected format "<CLUSTER-ID>/<NODEPOOL-ID>@<ZONE>"`, req.ID),
		)
		return
	}
	clusterID, nodepoolID := clusterAndNodepool[0], clusterAndNodepool[1]

	// Set timeouts (quirk https://github.com/hashicorp/terraform-plugin-framework-timeouts/issues/46)
	var t timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &t)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &ResourceNodepoolModel{
		ID:                   types.StringValue(nodepoolID),
		ClusterID:            types.StringValue(clusterID),
		Zone:                 types.StringValue(zone),
		Labels:               types.MapNull(types.StringType),
		Taints:               types.MapNull(types.StringType),
		AntiAffinityGroupIDs: types.SetNull(types.StringType),
		PrivateNetworkIDs:    types.SetNull(types.StringType),
		SecurityGroupIDs:     types.SetNull(types.StringType),
		KubeletImageGC:       types.ObjectNull(kubeletGCAttrTypes()),
		Timeouts:             t,
	})...)
}

// resourceNodepoolModelV0 mirrors the schema this resource had at version 0:
type resourceNodepoolModelV0 struct {
	ID                   types.String     `tfsdk:"id"`
	ClusterID            types.String     `tfsdk:"cluster_id"`
	Zone                 types.String     `tfsdk:"zone"`
	AntiAffinityGroupIDs types.Set        `tfsdk:"anti_affinity_group_ids"`
	CreatedAt            types.String     `tfsdk:"created_at"`
	DeployTargetID       types.String     `tfsdk:"deploy_target_id"`
	Description          types.String     `tfsdk:"description"`
	DiskSize             types.Int64      `tfsdk:"disk_size"`
	InstancePoolID       types.String     `tfsdk:"instance_pool_id"`
	InstancePrefix       types.String     `tfsdk:"instance_prefix"`
	InstanceType         types.String     `tfsdk:"instance_type"`
	KubeletImageGC       []kubeletGCModel `tfsdk:"kubelet_image_gc"`
	KubeletMaxPods       types.Int64      `tfsdk:"kubelet_max_pods"`
	Labels               types.Map        `tfsdk:"labels"`
	Name                 types.String     `tfsdk:"name"`
	NvidiaMigProfile     types.String     `tfsdk:"nvidia_mig_profile"`
	IPv6                 types.Bool       `tfsdk:"ipv6"`
	PrivateNetworkIDs    types.Set        `tfsdk:"private_network_ids"`
	SecurityGroupIDs     types.Set        `tfsdk:"security_group_ids"`
	Size                 types.Int64      `tfsdk:"size"`
	State                types.String     `tfsdk:"state"`
	StorageLVM           types.Bool       `tfsdk:"storage_lvm"`
	Taints               types.Map        `tfsdk:"taints"`
	TemplateID           types.String     `tfsdk:"template_id"`
	Version              types.String     `tfsdk:"version"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *ResourceNodepool) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		// SDKv2 to Framework migration: kubelet_image_gc was stored as an
		// array (SDKv2 TypeSet(MaxItems: 1)); it is now a single object
		// attribute. Every other attribute is byte-compatible as-is.
		0: {
			PriorSchema: &schema.Schema{
				Attributes: map[string]schema.Attribute{
					"id":                      schema.StringAttribute{Computed: true},
					"cluster_id":              schema.StringAttribute{Required: true},
					"zone":                    schema.StringAttribute{Required: true},
					"anti_affinity_group_ids": schema.SetAttribute{ElementType: types.StringType, Optional: true},
					"created_at":              schema.StringAttribute{Computed: true},
					"deploy_target_id":        schema.StringAttribute{Optional: true},
					"description":             schema.StringAttribute{Optional: true},
					"disk_size":               schema.Int64Attribute{Optional: true, Computed: true},
					"instance_pool_id":        schema.StringAttribute{Computed: true},
					"instance_prefix":         schema.StringAttribute{Optional: true, Computed: true},
					"instance_type":           schema.StringAttribute{Required: true},
					"kubelet_max_pods":        schema.Int64Attribute{Optional: true, Computed: true},
					"labels":                  schema.MapAttribute{ElementType: types.StringType, Optional: true},
					"name":                    schema.StringAttribute{Required: true},
					"nvidia_mig_profile":      schema.StringAttribute{Optional: true},
					"ipv6":                    schema.BoolAttribute{Optional: true, Computed: true},
					"private_network_ids":     schema.SetAttribute{ElementType: types.StringType, Optional: true},
					"security_group_ids":      schema.SetAttribute{ElementType: types.StringType, Optional: true},
					"size":                    schema.Int64Attribute{Required: true},
					"state":                   schema.StringAttribute{Computed: true},
					"storage_lvm":             schema.BoolAttribute{Optional: true, Computed: true},
					"taints":                  schema.MapAttribute{ElementType: types.StringType, Optional: true},
					"template_id":             schema.StringAttribute{Computed: true},
					"version":                 schema.StringAttribute{Computed: true},
				},
				Blocks: map[string]schema.Block{
					"kubelet_image_gc": schema.ListNestedBlock{
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"min_age":        schema.StringAttribute{Optional: true},
								"high_threshold": schema.Int64Attribute{Optional: true},
								"low_threshold":  schema.Int64Attribute{Optional: true},
							},
						},
					},
					"timeouts": timeouts.BlockAll(ctx),
				},
			},
			StateUpgrader: func(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
				var priorState resourceNodepoolModelV0

				resp.Diagnostics.Append(req.State.Get(ctx, &priorState)...)
				if resp.Diagnostics.HasError() {
					return
				}

				upgraded := ResourceNodepoolModel{
					ID:                   priorState.ID,
					ClusterID:            priorState.ClusterID,
					Zone:                 priorState.Zone,
					AntiAffinityGroupIDs: priorState.AntiAffinityGroupIDs,
					CreatedAt:            priorState.CreatedAt,
					DeployTargetID:       priorState.DeployTargetID,
					Description:          priorState.Description,
					DiskSize:             priorState.DiskSize,
					InstancePoolID:       priorState.InstancePoolID,
					InstancePrefix:       priorState.InstancePrefix,
					InstanceType:         priorState.InstanceType,
					KubeletMaxPods:       priorState.KubeletMaxPods,
					Labels:               priorState.Labels,
					Name:                 priorState.Name,
					NvidiaMigProfile:     priorState.NvidiaMigProfile,
					IPv6:                 priorState.IPv6,
					PrivateNetworkIDs:    priorState.PrivateNetworkIDs,
					SecurityGroupIDs:     priorState.SecurityGroupIDs,
					Size:                 priorState.Size,
					State:                priorState.State,
					StorageLVM:           priorState.StorageLVM,
					Taints:               priorState.Taints,
					TemplateID:           priorState.TemplateID,
					Version:              priorState.Version,
					Timeouts:             priorState.Timeouts,
				}

				upgraded.KubeletImageGC = types.ObjectNull(kubeletGCAttrTypes())
				if len(priorState.KubeletImageGC) > 0 {
					obj, d := types.ObjectValueFrom(ctx, kubeletGCAttrTypes(), priorState.KubeletImageGC[0])
					resp.Diagnostics.Append(d...)
					upgraded.KubeletImageGC = obj
				}

				resp.Diagnostics.Append(resp.State.Set(ctx, upgraded)...)
			},
		},
	}
}

func (r *ResourceNodepool) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceNodepoolModel

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
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	cluster, err := client.GetSKSCluster(ctx, exoscale.UUID(plan.ClusterID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching SKS cluster", err.Error())
		return
	}

	createReq := exoscale.CreateSKSNodepoolRequest{}

	antiAffinityGroups := []exoscale.AntiAffinityGroup{}
	if !plan.AntiAffinityGroupIDs.IsNull() && !plan.AntiAffinityGroupIDs.IsUnknown() {
		var ids []string
		resp.Diagnostics.Append(plan.AntiAffinityGroupIDs.ElementsAs(ctx, &ids, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, id := range ids {
			antiAffinityGroups = append(antiAffinityGroups, exoscale.AntiAffinityGroup{ID: exoscale.UUID(id)})
		}
	}
	createReq.AntiAffinityGroups = antiAffinityGroups

	if v := plan.DeployTargetID.ValueString(); v != "" {
		createReq.DeployTarget = &exoscale.DeployTarget{ID: exoscale.UUID(v)}
	}

	if v := plan.Description.ValueString(); v != "" {
		createReq.Description = v
	}

	createReq.DiskSize = plan.DiskSize.ValueInt64()
	createReq.InstancePrefix = plan.InstancePrefix.ValueString()

	if plan.IPv6.ValueBool() {
		createReq.PublicIPAssignment = exoscale.CreateSKSNodepoolRequestPublicIPAssignmentDual
	} else {
		createReq.PublicIPAssignment = exoscale.CreateSKSNodepoolRequestPublicIPAssignmentInet4
	}

	instanceType, err := utils.FindInstanceTypeByNameV3(ctx, client, plan.InstanceType.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("error retrieving instance type", err.Error())
		return
	}
	createReq.InstanceType = &exoscale.InstanceType{ID: instanceType.ID}

	if b, ok := r.kubeletGCBlock(ctx, plan, &resp.Diagnostics); ok {
		createReq.KubeletImageGC = &exoscale.KubeletImageGC{
			MinAge:        b.MinAge.ValueString(),
			HighThreshold: b.HighThreshold.ValueInt64(),
			LowThreshold:  b.LowThreshold.ValueInt64(),
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if !plan.KubeletMaxPods.IsNull() && !plan.KubeletMaxPods.IsUnknown() {
		v := plan.KubeletMaxPods.ValueInt64()
		createReq.KubeletMaxPods = &v
	}

	if !plan.Labels.IsNull() && !plan.Labels.IsUnknown() && len(plan.Labels.Elements()) > 0 {
		labels := map[string]string{}
		resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		createReq.Labels = labels
	}

	createReq.Name = plan.Name.ValueString()

	if profile := plan.NvidiaMigProfile.ValueString(); profile != "" {
		profiles, err := sksNodepoolMIGProfiles(sksNodepoolInstanceTypeFamily(plan.InstanceType.ValueString()), profile)
		if err != nil {
			resp.Diagnostics.AddError("invalid nvidia_mig_profile", err.Error())
			return
		}
		createReq.NvidiaMigProfiles = profiles
	}

	privateNetworks := []exoscale.PrivateNetwork{}
	if !plan.PrivateNetworkIDs.IsNull() && !plan.PrivateNetworkIDs.IsUnknown() {
		var ids []string
		resp.Diagnostics.Append(plan.PrivateNetworkIDs.ElementsAs(ctx, &ids, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, id := range ids {
			privateNetworks = append(privateNetworks, exoscale.PrivateNetwork{ID: exoscale.UUID(id)})
		}
	}
	createReq.PrivateNetworks = privateNetworks

	securityGroups := []exoscale.SecurityGroup{}
	if !plan.SecurityGroupIDs.IsNull() && !plan.SecurityGroupIDs.IsUnknown() {
		var ids []string
		resp.Diagnostics.Append(plan.SecurityGroupIDs.ElementsAs(ctx, &ids, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, id := range ids {
			securityGroups = append(securityGroups, exoscale.SecurityGroup{ID: exoscale.UUID(id)})
		}
	}
	createReq.SecurityGroups = securityGroups

	createReq.Size = plan.Size.ValueInt64()

	var addOns []string
	if plan.StorageLVM.ValueBool() {
		addOns = append(addOns, sksNodepoolAddonStorageLVM)
	}
	if len(addOns) > 0 {
		createReq.Addons = addOns
	}

	if !plan.Taints.IsNull() && !plan.Taints.IsUnknown() {
		var taintValues map[string]string
		resp.Diagnostics.Append(plan.Taints.ElementsAs(ctx, &taintValues, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		taints := make(exoscale.SKSNodepoolTaints, len(taintValues))
		for k, v := range taintValues {
			taint, err := parseSKSNodepoolTaintV3(v)
			if err != nil {
				resp.Diagnostics.AddError("invalid taint", fmt.Sprintf("invalid taint %q: %s", v, err))
				return
			}
			taints[k] = *taint
		}
		createReq.Taints = taints
	}

	op, err := client.CreateSKSNodepool(ctx, cluster.ID, createReq)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error when creating SKS nodepool", err.Error())
		return
	}
	op, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("create SKS nodepool operation failed", err.Error())
		return
	}

	plan.ID = types.StringValue(op.Reference.ID.String())

	tflog.Debug(ctx, "create finished successfully", map[string]any{"id": plan.ID.ValueString()})

	cluster, err = client.GetSKSCluster(ctx, cluster.ID)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching SKS cluster", err.Error())
		return
	}

	nodepool := findSKSNodepool(cluster, plan.ID.ValueString())
	if nodepool == nil {
		resp.Diagnostics.AddError("SKS nodepool not found after creation", plan.ID.ValueString())
		return
	}

	if diags := r.applyNodepool(ctx, client, cluster, nodepool, &plan); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceNodepool) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceNodepoolModel

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

	client, err := utils.SwitchClientZone(
		ctx,
		r.client,
		exoscale.ZoneName(state.Zone.ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	cluster, err := client.GetSKSCluster(ctx, exoscale.UUID(state.ClusterID.ValueString()))
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			// Parent SKS cluster doesn't exist anymore, so doesn't the SKS Nodepool.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("API returned an error while fetching SKS cluster", err.Error())
		return
	}

	nodepool := findSKSNodepool(cluster, state.ID.ValueString())
	if nodepool == nil {
		// Resource doesn't exist anymore, signaling the core to remove it from the state.
		resp.State.RemoveResource(ctx)
		return
	}

	if diags := r.applyNodepool(ctx, client, cluster, nodepool, &state); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	tflog.Debug(ctx, "read finished successfully", map[string]any{"id": state.ID.ValueString()})

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ResourceNodepool) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ResourceNodepoolModel

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

	client, err := utils.SwitchClientZone(
		ctx,
		r.client,
		exoscale.ZoneName(plan.Zone.ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	cluster, err := client.GetSKSCluster(ctx, exoscale.UUID(plan.ClusterID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching SKS cluster", err.Error())
		return
	}

	nodepool := findSKSNodepool(cluster, state.ID.ValueString())
	if nodepool == nil {
		resp.Diagnostics.AddError("SKS nodepool not found", fmt.Sprintf("SKS Nodepool %q not found", state.ID.ValueString()))
		return
	}

	updateReq := exoscale.UpdateSKSNodepoolRequest{
		AntiAffinityGroups: nodepool.AntiAffinityGroups,
		DeployTarget:       nodepool.DeployTarget,
		Description:        nodepool.Description,
		DiskSize:           nodepool.DiskSize,
		InstancePrefix:     nodepool.InstancePrefix,
		InstanceType:       nodepool.InstanceType,
		KubeletMaxPods:     nodepool.KubeletMaxPods,
		Labels:             nodepool.Labels,
		Name:               nodepool.Name,
		NvidiaMigProfiles:  nodepool.NvidiaMigProfiles,
		PrivateNetworks:    nodepool.PrivateNetworks,
		PublicIPAssignment: exoscale.UpdateSKSNodepoolRequestPublicIPAssignment(nodepool.PublicIPAssignment),
		SecurityGroups:     nodepool.SecurityGroups,
		Taints:             nodepool.Taints,
	}

	var updated bool

	if !plan.AntiAffinityGroupIDs.Equal(state.AntiAffinityGroupIDs) {
		var ids []string
		if !plan.AntiAffinityGroupIDs.IsNull() && !plan.AntiAffinityGroupIDs.IsUnknown() {
			resp.Diagnostics.Append(plan.AntiAffinityGroupIDs.ElementsAs(ctx, &ids, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		aags := make([]exoscale.AntiAffinityGroup, len(ids))
		for i, id := range ids {
			aags[i] = exoscale.AntiAffinityGroup{ID: exoscale.UUID(id)}
		}
		updateReq.AntiAffinityGroups = aags
		updated = true
	}

	if !plan.DeployTargetID.Equal(state.DeployTargetID) {
		updateReq.DeployTarget = &exoscale.DeployTarget{ID: exoscale.UUID(plan.DeployTargetID.ValueString())}
		updated = true
	}

	if !plan.Description.Equal(state.Description) {
		updateReq.Description = plan.Description.ValueString()
		updated = true
	}

	if !plan.DiskSize.Equal(state.DiskSize) {
		updateReq.DiskSize = plan.DiskSize.ValueInt64()
		updated = true
	}

	if !plan.InstancePrefix.Equal(state.InstancePrefix) {
		updateReq.InstancePrefix = plan.InstancePrefix.ValueString()
		updated = true
	}

	if !plan.InstanceType.Equal(state.InstanceType) {
		instanceType, err := utils.FindInstanceTypeByNameV3(ctx, client, plan.InstanceType.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("error retrieving instance type", err.Error())
			return
		}
		updateReq.InstanceType = instanceType
		updated = true
	}

	if !plan.Labels.Equal(state.Labels) {
		labels := map[string]string{}
		if !plan.Labels.IsNull() && !plan.Labels.IsUnknown() {
			resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &labels, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		updateReq.Labels = labels
		updated = true
	}

	if !plan.Name.Equal(state.Name) {
		updateReq.Name = plan.Name.ValueString()
		updated = true
	}

	// Recompute the MIG profiles when either the profile itself or the instance
	// type (and thus the inferred GPU family) changes.
	if !plan.NvidiaMigProfile.Equal(state.NvidiaMigProfile) || !plan.InstanceType.Equal(state.InstanceType) {
		if profile := plan.NvidiaMigProfile.ValueString(); profile != "" {
			profiles, err := sksNodepoolMIGProfiles(sksNodepoolInstanceTypeFamily(plan.InstanceType.ValueString()), profile)
			if err != nil {
				resp.Diagnostics.AddError("invalid nvidia_mig_profile", err.Error())
				return
			}
			updateReq.NvidiaMigProfiles = profiles
		} else {
			updateReq.NvidiaMigProfiles = nil
		}
		updated = true
	}

	if !plan.PrivateNetworkIDs.Equal(state.PrivateNetworkIDs) {
		var ids []string
		if !plan.PrivateNetworkIDs.IsNull() && !plan.PrivateNetworkIDs.IsUnknown() {
			resp.Diagnostics.Append(plan.PrivateNetworkIDs.ElementsAs(ctx, &ids, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		pns := make([]exoscale.PrivateNetwork, len(ids))
		for i, id := range ids {
			pns[i] = exoscale.PrivateNetwork{ID: exoscale.UUID(id)}
		}
		updateReq.PrivateNetworks = pns
		updated = true
	}

	if !plan.SecurityGroupIDs.Equal(state.SecurityGroupIDs) {
		var ids []string
		if !plan.SecurityGroupIDs.IsNull() && !plan.SecurityGroupIDs.IsUnknown() {
			resp.Diagnostics.Append(plan.SecurityGroupIDs.ElementsAs(ctx, &ids, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		sgs := make([]exoscale.SecurityGroup, len(ids))
		for i, id := range ids {
			sgs[i] = exoscale.SecurityGroup{ID: exoscale.UUID(id)}
		}
		updateReq.SecurityGroups = sgs
		updated = true
	}

	if !plan.KubeletImageGC.Equal(state.KubeletImageGC) {
		if b, ok := r.kubeletGCBlock(ctx, plan, &resp.Diagnostics); ok {
			updateReq.KubeletImageGC = &exoscale.KubeletImageGC{
				MinAge:        b.MinAge.ValueString(),
				HighThreshold: b.HighThreshold.ValueInt64(),
				LowThreshold:  b.LowThreshold.ValueInt64(),
			}
		} else {
			updateReq.KubeletImageGC = nil
		}
		if resp.Diagnostics.HasError() {
			return
		}
		updated = true
	}

	if !plan.KubeletMaxPods.Equal(state.KubeletMaxPods) {
		if !plan.KubeletMaxPods.IsNull() && !plan.KubeletMaxPods.IsUnknown() {
			v := plan.KubeletMaxPods.ValueInt64()
			updateReq.KubeletMaxPods = &v
		}
		updated = true
	}

	if !plan.Taints.Equal(state.Taints) {
		taints := make(exoscale.SKSNodepoolTaints)
		if !plan.Taints.IsNull() && !plan.Taints.IsUnknown() {
			var taintValues map[string]string
			resp.Diagnostics.Append(plan.Taints.ElementsAs(ctx, &taintValues, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			for k, v := range taintValues {
				taint, err := parseSKSNodepoolTaintV3(v)
				if err != nil {
					resp.Diagnostics.AddError("invalid taint", fmt.Sprintf("invalid taint %q: %s", v, err))
					return
				}
				taints[k] = *taint
			}
		}
		updateReq.Taints = taints
		updated = true
	}

	if !plan.IPv6.Equal(state.IPv6) {
		if plan.IPv6.ValueBool() {
			updateReq.PublicIPAssignment = exoscale.UpdateSKSNodepoolRequestPublicIPAssignmentDual
		} else {
			updateReq.PublicIPAssignment = exoscale.UpdateSKSNodepoolRequestPublicIPAssignmentInet4
		}
		updated = true
	}

	if updated {
		op, err := client.UpdateSKSNodepool(ctx, cluster.ID, nodepool.ID, updateReq)
		if err != nil {
			resp.Diagnostics.AddError("API returned an error when updating SKS nodepool", err.Error())
			return
		}
		if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
			resp.Diagnostics.AddError("update SKS nodepool operation failed", err.Error())
			return
		}
	}

	if !plan.Size.Equal(state.Size) {
		op, err := client.ScaleSKSNodepool(
			ctx,
			cluster.ID,
			nodepool.ID,
			exoscale.ScaleSKSNodepoolRequest{
				Size: plan.Size.ValueInt64(),
			},
		)
		if err != nil {
			resp.Diagnostics.AddError("API returned an error when scaling SKS nodepool", err.Error())
			return
		}
		if _, err = client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
			resp.Diagnostics.AddError("scale SKS nodepool operation failed", err.Error())
			return
		}
	}

	tflog.Debug(ctx, "update finished successfully", map[string]any{"id": nodepool.ID.String()})

	cluster, err = client.GetSKSCluster(ctx, cluster.ID)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching SKS cluster", err.Error())
		return
	}

	updatedNodepool := findSKSNodepool(cluster, state.ID.ValueString())
	if updatedNodepool == nil {
		resp.Diagnostics.AddError("SKS nodepool not found after update", state.ID.ValueString())
		return
	}

	if diags := r.applyNodepool(ctx, client, cluster, updatedNodepool, &plan); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceNodepool) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceNodepoolModel

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

	client, err := utils.SwitchClientZone(
		ctx,
		r.client,
		exoscale.ZoneName(state.Zone.ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	cluster, err := client.GetSKSCluster(ctx, exoscale.UUID(state.ClusterID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching SKS cluster", err.Error())
		return
	}

	op, err := client.DeleteSKSNodepool(ctx, cluster.ID, exoscale.UUID(state.ID.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("API returned an error when deleting SKS nodepool", err.Error())
		return
	}
	if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("delete SKS nodepool operation failed", err.Error())
		return
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]any{"id": state.ID.ValueString()})
}

// kubeletGCBlock decodes the (at most one) "kubelet_image_gc" nested object from a model.
func (r *ResourceNodepool) kubeletGCBlock(ctx context.Context, m ResourceNodepoolModel, diags *diag.Diagnostics) (kubeletGCModel, bool) {
	if m.KubeletImageGC.IsNull() || m.KubeletImageGC.IsUnknown() {
		return kubeletGCModel{}, false
	}
	var block kubeletGCModel
	diags.Append(m.KubeletImageGC.As(ctx, &block, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return kubeletGCModel{}, false
	}

	return block, true
}

// findSKSNodepool returns the nodepool with the given ID from cluster.Nodepools,
// or nil if it isn't found.
func findSKSNodepool(cluster *exoscale.SKSCluster, id string) *exoscale.SKSNodepool {
	for i := range cluster.Nodepools {
		if cluster.Nodepools[i].ID.String() == id {
			return &cluster.Nodepools[i]
		}
	}

	return nil
}

// setContainsString returns whether s (a types.Set of strings) contains v.
func setContainsString(ctx context.Context, s types.Set, v string) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	if s.IsNull() || s.IsUnknown() {
		return false, diags
	}

	var values []string
	diags.Append(s.ElementsAs(ctx, &values, false)...)

	return in(values, v), diags
}

// applyNodepool fetches the nodepool's instance type (and its CA... none here, just
// the instance type label) and maps the remote state onto model. It mirrors the
// former SDKv2 resourceSKSNodepoolApply.
func (r *ResourceNodepool) applyNodepool(
	ctx context.Context,
	client *exoscale.Client,
	cluster *exoscale.SKSCluster,
	nodepool *exoscale.SKSNodepool,
	model *ResourceNodepoolModel,
) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(nodepool.ID.String())

	model.AntiAffinityGroupIDs = types.SetNull(types.StringType)
	if len(nodepool.AntiAffinityGroups) > 0 {
		aags, d := types.SetValueFrom(ctx, types.StringType, utils.AntiAffiniGroupsToAntiAffinityGroupIDs(nodepool.AntiAffinityGroups))
		diags.Append(d...)
		model.AntiAffinityGroupIDs = aags
	}

	if nodepool.Addons != nil {
		model.StorageLVM = types.BoolValue(in(nodepool.Addons, sksNodepoolAddonStorageLVM))
	}

	model.CreatedAt = types.StringValue(nodepool.CreatedAT.String())

	if nodepool.DeployTarget != nil {
		model.DeployTargetID = types.StringValue(nodepool.DeployTarget.ID.String())
	}

	model.Description = optionalString(nodepool.Description)
	model.DiskSize = types.Int64Value(nodepool.DiskSize)

	model.InstancePoolID = types.StringNull()
	if nodepool.InstancePool != nil {
		model.InstancePoolID = types.StringValue(nodepool.InstancePool.ID.String())
	}

	model.InstancePrefix = types.StringValue(nodepool.InstancePrefix)

	instanceType, err := client.GetInstanceType(ctx, nodepool.InstanceType.ID)
	if err != nil {
		diags.AddError("error retrieving instance type", err.Error())
		return diags
	}
	model.InstanceType = types.StringValue(fmt.Sprintf(
		"%s.%s",
		strings.ToLower(string(instanceType.Family)),
		strings.ToLower(string(instanceType.Size)),
	))

	model.KubeletImageGC = types.ObjectNull(kubeletGCAttrTypes())
	if nodepool.KubeletImageGC != nil {
		obj, d := types.ObjectValueFrom(ctx, kubeletGCAttrTypes(), kubeletGCModel{
			MinAge:        optionalString(nodepool.KubeletImageGC.MinAge),
			HighThreshold: types.Int64Value(nodepool.KubeletImageGC.HighThreshold),
			LowThreshold:  types.Int64Value(nodepool.KubeletImageGC.LowThreshold),
		})
		diags.Append(d...)
		model.KubeletImageGC = obj
	}

	model.KubeletMaxPods = types.Int64Null()
	if nodepool.KubeletMaxPods != nil {
		model.KubeletMaxPods = types.Int64Value(*nodepool.KubeletMaxPods)
	}

	labels := types.MapNull(types.StringType)
	if len(nodepool.Labels) > 0 {
		l, d := types.MapValueFrom(ctx, types.StringType, nodepool.Labels)
		diags.Append(d...)
		labels = l
	}
	model.Labels = labels

	model.Name = types.StringValue(nodepool.Name)
	model.NvidiaMigProfile = optionalString(sksNodepoolMIGProfile(nodepool.NvidiaMigProfiles))

	model.PrivateNetworkIDs = types.SetNull(types.StringType)
	if len(nodepool.PrivateNetworks) > 0 {
		pns, d := types.SetValueFrom(ctx, types.StringType, utils.PrivateNetworksToPrivateNetworkIDs(nodepool.PrivateNetworks))
		diags.Append(d...)
		model.PrivateNetworkIDs = pns
	}

	{
		sgs := utils.SecurityGroupsToSecurityGroupIDs(nodepool.SecurityGroups)

		// When the parent cluster was created with `create_default_security_group`,
		// the API auto-attaches that SG to every nodepool. Hide it from state
		// unless the user's config explicitly lists it; otherwise Terraform would
		// see perpetual drift against a user config that doesn't mention it.
		if cluster.DefaultSecurityGroupID != nil {
			defaultID := cluster.DefaultSecurityGroupID.String()
			userIncludes, d := setContainsString(ctx, model.SecurityGroupIDs, defaultID)
			diags.Append(d...)
			if !userIncludes {
				filtered := sgs[:0]
				for _, id := range sgs {
					if id != defaultID {
						filtered = append(filtered, id)
					}
				}
				sgs = filtered
			}
		}

		model.SecurityGroupIDs = types.SetNull(types.StringType)
		if len(sgs) > 0 {
			sgSet, d := types.SetValueFrom(ctx, types.StringType, sgs)
			diags.Append(d...)
			model.SecurityGroupIDs = sgSet
		}
	}

	model.Size = types.Int64Value(nodepool.Size)
	model.State = types.StringValue(string(nodepool.State))

	model.Taints = types.MapNull(types.StringType)
	if len(nodepool.Taints) > 0 {
		taints := make(map[string]string, len(nodepool.Taints))
		for k, v := range nodepool.Taints {
			taints[k] = fmt.Sprintf("%s:%s", v.Value, v.Effect)
		}
		t, d := types.MapValueFrom(ctx, types.StringType, taints)
		diags.Append(d...)
		model.Taints = t
	}

	model.TemplateID = types.StringNull()
	if nodepool.Template != nil {
		model.TemplateID = types.StringValue(nodepool.Template.ID.String())
	}

	model.Version = types.StringValue(nodepool.Version)

	model.IPv6 = types.BoolValue(nodepool.PublicIPAssignment == exoscale.SKSNodepoolPublicIPAssignmentDual)

	return diags
}
