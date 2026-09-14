package sks_cluster

import (
	"context"
	"fmt"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const markdownDescriptionNodepoolDataSource = `Fetch Exoscale [Scalable Kubernetes Service (SKS)](https://community.exoscale.com/product/compute/containers/) Node Pool data.

Corresponding resource: [exoscale_sks_nodepool](../resources/sks_nodepool.md).`

var _ datasource.DataSourceWithConfigure = (*DataSourceNodepool)(nil)

type DataSourceNodepool struct {
	client *exoscale.Client
}

func NewDataSourceNodepool() datasource.DataSource {
	return &DataSourceNodepool{}
}

type DataSourceNodepoolModel struct {
	ID        types.String `tfsdk:"id"`
	ClusterID types.String `tfsdk:"cluster_id"`
	Zone      types.String `tfsdk:"zone"`
	Name      types.String `tfsdk:"name"`

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

func (d *DataSourceNodepool) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sks_nodepool"
}

func (d *DataSourceNodepool) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: markdownDescriptionNodepoolDataSource,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The SKS node pool ID to match (conflicts with 'name').",
				MarkdownDescription: "The SKS node pool ID to match (conflicts with `name`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("name"),
					}...),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The SKS node pool name to match (conflicts with 'id').",
				MarkdownDescription: "The SKS node pool name to match (conflicts with `id`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("id"),
					}...),
				},
			},
			"cluster_id": schema.StringAttribute{
				Description:         "The parent exoscale_sks_cluster ID.",
				MarkdownDescription: "The parent [exoscale_sks_cluster](../resources/sks_cluster.md) ID.",
				Required:            true,
			},
			"zone": schema.StringAttribute{
				Description:         "The Exoscale zone name.",
				MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"anti_affinity_group_ids": schema.SetAttribute{
				Description:         "A list of exoscale_anti_affinity_group (IDs) attached to the managed instances.",
				MarkdownDescription: "A list of [exoscale_anti_affinity_group](../resources/anti_affinity_group.md) (IDs) attached to the managed instances.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				Description:         "The pool creation date.",
				MarkdownDescription: "The pool creation date.",
				Computed:            true,
			},
			"deploy_target_id": schema.StringAttribute{
				Description:         "A deploy target ID.",
				MarkdownDescription: "A deploy target ID.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				Description:         "A free-form text describing the pool.",
				MarkdownDescription: "A free-form text describing the pool.",
				Computed:            true,
			},
			"disk_size": schema.Int64Attribute{
				Description:         "The managed instances disk size (GiB).",
				MarkdownDescription: "The managed instances disk size (GiB).",
				Computed:            true,
			},
			"instance_pool_id": schema.StringAttribute{
				Description:         "The underlying exoscale_instance_pool ID.",
				MarkdownDescription: "The underlying [exoscale_instance_pool](../resources/instance_pool.md) ID.",
				Computed:            true,
			},
			"instance_prefix": schema.StringAttribute{
				Description:         "The string used to prefix the managed instances name.",
				MarkdownDescription: "The string used to prefix the managed instances name.",
				Computed:            true,
			},
			"instance_type": schema.StringAttribute{
				Description:         "The managed compute instances type ('<family>.<size>', e.g. 'standard.medium').",
				MarkdownDescription: "The managed compute instances type (`<family>.<size>`, e.g. `standard.medium`).",
				Computed:            true,
			},
			"kubelet_max_pods": schema.Int64Attribute{
				Description:         "The maximum number of pods per node.",
				MarkdownDescription: "The maximum number of pods per node.",
				Computed:            true,
			},
			"labels": schema.MapAttribute{
				Description:         "A map of key/value labels.",
				MarkdownDescription: "A map of key/value labels.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"nvidia_mig_profile": schema.StringAttribute{
				Description:         "The NVIDIA Multi-Instance GPU (MIG) profile enabled on the managed GPUs.",
				MarkdownDescription: "The NVIDIA [Multi-Instance GPU (MIG)](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/) profile enabled on the managed GPUs.",
				Computed:            true,
			},
			"ipv6": schema.BoolAttribute{
				Description:         "Whether IPv6 is enabled for the nodepool nodes.",
				MarkdownDescription: "Whether IPv6 is enabled for the nodepool nodes.",
				Computed:            true,
			},
			"private_network_ids": schema.SetAttribute{
				Description:         "A list of exoscale_private_network (IDs) attached to the managed instances.",
				MarkdownDescription: "A list of [exoscale_private_network](../resources/private_network.md) (IDs) attached to the managed instances.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"security_group_ids": schema.SetAttribute{
				Description:         "A list of exoscale_security_group (IDs) attached to the managed instances.",
				MarkdownDescription: "A list of [exoscale_security_group](../resources/security_group.md) (IDs) attached to the managed instances.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"size": schema.Int64Attribute{
				Description:         "The number of managed instances.",
				MarkdownDescription: "The number of managed instances.",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				Description:         "The current pool state.",
				MarkdownDescription: "The current pool state.",
				Computed:            true,
			},
			"storage_lvm": schema.BoolAttribute{
				Description:         "Whether nodes were created with non-standard partitioning for persistent storage.",
				MarkdownDescription: "Whether nodes were created with non-standard partitioning for persistent storage.",
				Computed:            true,
			},
			"taints": schema.MapAttribute{
				Description:         `A map of key/value Kubernetes taints ("<value>:<effect>").`,
				MarkdownDescription: `A map of key/value Kubernetes [taints](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) ('<value>:<effect>").`,
				ElementType:         types.StringType,
				Computed:            true,
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
			"kubelet_image_gc": schema.SingleNestedAttribute{
				Description:         "Configuration for this nodepool's kubelet image garbage collector.",
				MarkdownDescription: "Configuration for this nodepool's kubelet image garbage collector.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"min_age": schema.StringAttribute{
						Description:         "The minimum age for an unused image before it is garbage collected (k8s duration format, eg. 1h)",
						MarkdownDescription: "The minimum age for an unused image before it is garbage collected (k8s duration format, eg. 1h)",
						Computed:            true,
					},
					"high_threshold": schema.Int64Attribute{
						Description:         "The percent of disk usage after which image garbage collection is always run",
						MarkdownDescription: "The percent of disk usage after which image garbage collection is always run",
						Computed:            true,
					},
					"low_threshold": schema.Int64Attribute{
						Description:         "The percent of disk usage before which image garbage collection is never run",
						MarkdownDescription: "The percent of disk usage before which image garbage collection is never run",
						Computed:            true,
					},
				},
			},
		},

		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Read: true,
			}),
		},
	}
}

func (d *DataSourceNodepool) Configure(ctx context.Context, r datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if r.ProviderData == nil {
		return
	}

	d.client = r.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (d *DataSourceNodepool) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DataSourceNodepoolModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
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
		d.client,
		exoscale.ZoneName(state.Zone.ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	clusterID := state.ClusterID.ValueString()
	cluster, err := client.GetSKSCluster(ctx, exoscale.UUID(clusterID))
	if err != nil {
		resp.Diagnostics.AddError("error getting cluster", fmt.Sprintf("error getting cluster %q: %s", clusterID, err))
		return
	}

	var nodepool *exoscale.SKSNodepool
	switch {
	case !state.ID.IsNull():
		nodepool = findSKSNodepool(cluster, state.ID.ValueString())
		if nodepool == nil {
			resp.Diagnostics.AddError("nodepool not found", fmt.Sprintf("no nodepool with id %q found in cluster %q", state.ID.ValueString(), clusterID))
			return
		}
	case !state.Name.IsNull():
		for i := range cluster.Nodepools {
			if cluster.Nodepools[i].Name == state.Name.ValueString() {
				nodepool = &cluster.Nodepools[i]
				break
			}
		}
		if nodepool == nil {
			resp.Diagnostics.AddError("nodepool not found", fmt.Sprintf("no nodepool with name %q found in cluster %q", state.Name.ValueString(), clusterID))
			return
		}
	default: // validation prevents this, exit as a safe guard
		resp.Diagnostics.AddError("missing values", "id and name are missing")
		return
	}

	state.ID = types.StringValue(nodepool.ID.String())
	state.Name = types.StringValue(nodepool.Name)
	state.CreatedAt = types.StringValue(nodepool.CreatedAT.String())
	state.Description = types.StringValue(nodepool.Description)
	state.DiskSize = types.Int64Value(nodepool.DiskSize)
	state.InstancePrefix = types.StringValue(nodepool.InstancePrefix)
	state.Size = types.Int64Value(nodepool.Size)
	state.State = types.StringValue(string(nodepool.State))
	state.Version = types.StringValue(nodepool.Version)
	state.IPv6 = types.BoolValue(nodepool.PublicIPAssignment == exoscale.SKSNodepoolPublicIPAssignmentDual)
	state.StorageLVM = types.BoolValue(in(nodepool.Addons, sksNodepoolAddonStorageLVM))
	state.NvidiaMigProfile = optionalString(sksNodepoolMIGProfile(nodepool.NvidiaMigProfiles))

	state.DeployTargetID = types.StringNull()
	if nodepool.DeployTarget != nil {
		state.DeployTargetID = types.StringValue(nodepool.DeployTarget.ID.String())
	}

	state.InstancePoolID = types.StringNull()
	if nodepool.InstancePool != nil {
		state.InstancePoolID = types.StringValue(nodepool.InstancePool.ID.String())
	}

	state.TemplateID = types.StringNull()
	if nodepool.Template != nil {
		state.TemplateID = types.StringValue(nodepool.Template.ID.String())
	}

	state.KubeletMaxPods = types.Int64Null()
	if nodepool.KubeletMaxPods != nil {
		state.KubeletMaxPods = types.Int64Value(*nodepool.KubeletMaxPods)
	}

	state.InstanceType = types.StringNull()
	if nodepool.InstanceType != nil {
		instanceType, err := client.GetInstanceType(ctx, nodepool.InstanceType.ID)
		if err != nil {
			resp.Diagnostics.AddError("error retrieving instance type", err.Error())
			return
		}
		state.InstanceType = types.StringValue(fmt.Sprintf(
			"%s.%s",
			strings.ToLower(string(instanceType.Family)),
			strings.ToLower(string(instanceType.Size)),
		))
	}

	aags, dg := types.SetValueFrom(ctx, types.StringType, utils.AntiAffiniGroupsToAntiAffinityGroupIDs(nodepool.AntiAffinityGroups))
	resp.Diagnostics.Append(dg...)
	state.AntiAffinityGroupIDs = aags

	pns, dg := types.SetValueFrom(ctx, types.StringType, utils.PrivateNetworksToPrivateNetworkIDs(nodepool.PrivateNetworks))
	resp.Diagnostics.Append(dg...)
	state.PrivateNetworkIDs = pns

	sgs, dg := types.SetValueFrom(ctx, types.StringType, utils.SecurityGroupsToSecurityGroupIDs(nodepool.SecurityGroups))
	resp.Diagnostics.Append(dg...)
	state.SecurityGroupIDs = sgs

	state.Labels = types.MapNull(types.StringType)
	if len(nodepool.Labels) > 0 {
		labels, dgl := types.MapValueFrom(ctx, types.StringType, nodepool.Labels)
		resp.Diagnostics.Append(dgl...)
		state.Labels = labels
	}

	state.Taints = types.MapNull(types.StringType)
	if len(nodepool.Taints) > 0 {
		taints := make(map[string]string, len(nodepool.Taints))
		for k, v := range nodepool.Taints {
			taints[k] = fmt.Sprintf("%s:%s", v.Value, v.Effect)
		}
		t, dgt := types.MapValueFrom(ctx, types.StringType, taints)
		resp.Diagnostics.Append(dgt...)
		state.Taints = t
	}

	state.KubeletImageGC = types.ObjectNull(kubeletGCAttrTypes())
	if nodepool.KubeletImageGC != nil {
		obj, d := types.ObjectValueFrom(ctx, kubeletGCAttrTypes(), kubeletGCModel{
			MinAge:        optionalString(nodepool.KubeletImageGC.MinAge),
			HighThreshold: types.Int64Value(nodepool.KubeletImageGC.HighThreshold),
			LowThreshold:  types.Int64Value(nodepool.KubeletImageGC.LowThreshold),
		})
		resp.Diagnostics.Append(d...)
		state.KubeletImageGC = obj
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
