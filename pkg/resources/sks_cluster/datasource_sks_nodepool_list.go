package sks_cluster

import (
	"context"
	"crypto/md5" //nolint:gosec // used only to derive a stable synthetic data source ID, not for security purposes
	"fmt"
	"sort"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const markdownDescriptionDataSourceNodepoolList = `List Exoscale [Scalable Kubernetes Service (SKS)](https://community.exoscale.com/product/compute/containers/) Node Pools in a zone.

Corresponding resource: [exoscale_sks_nodepool](../resources/sks_nodepool.md).`

var _ datasource.DataSourceWithConfigure = (*DataSourceNodepoolList)(nil)

type DataSourceNodepoolList struct {
	client *exoscale.Client
}

func NewDataSourceNodepoolList() datasource.DataSource {
	return &DataSourceNodepoolList{}
}

type DataSourceNodepoolListModel struct {
	ID        types.String                      `tfsdk:"id"`
	Zone      types.String                      `tfsdk:"zone"`
	Nodepools []DataSourceNodepoolListItemModel `tfsdk:"nodepools"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type DataSourceNodepoolListItemModel struct {
	ID        types.String `tfsdk:"id"`
	ClusterID types.String `tfsdk:"cluster_id"`
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
}

func (d *DataSourceNodepoolList) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sks_nodepool_list"
}

func (d *DataSourceNodepoolList) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: markdownDescriptionDataSourceNodepoolList,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The ID of this resource.",
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
			},
			"zone": schema.StringAttribute{
				Description:         "The Exoscale zone name.",
				MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"nodepools": schema.ListNestedAttribute{
				Description:         "The list of SKS node pools in the zone.",
				MarkdownDescription: "The list of [exoscale_sks_nodepool](./sks_nodepool.md) in the zone.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description:         "The SKS node pool ID.",
							MarkdownDescription: "The SKS node pool ID.",
							Computed:            true,
						},
						"cluster_id": schema.StringAttribute{
							Description:         "The parent exoscale_sks_cluster ID.",
							MarkdownDescription: "The parent [exoscale_sks_cluster](./sks_cluster.md) ID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							Description:         "The SKS node pool name.",
							MarkdownDescription: "The SKS node pool name.",
							Computed:            true,
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

func (d *DataSourceNodepoolList) Configure(ctx context.Context, r datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if r.ProviderData == nil {
		return
	}

	d.client = r.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (d *DataSourceNodepoolList) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DataSourceNodepoolListModel

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

	listResp, err := client.ListSKSClusters(ctx)
	if err != nil {
		resp.Diagnostics.AddError("error listing SKS clusters", err.Error())
		return
	}

	// Cache resolved instance types since many node pools tend to share the
	// same one, avoiding redundant API calls.
	instanceTypes := map[exoscale.UUID]string{}
	resolveInstanceType := func(id exoscale.UUID) (string, error) {
		if t, ok := instanceTypes[id]; ok {
			return t, nil
		}

		instanceType, err := client.GetInstanceType(ctx, id)
		if err != nil {
			return "", err
		}

		t := fmt.Sprintf("%s.%s", strings.ToLower(string(instanceType.Family)), strings.ToLower(string(instanceType.Size)))
		instanceTypes[id] = t

		return t, nil
	}

	var ids []string
	var items []DataSourceNodepoolListItemModel

	for ci := range listResp.SKSClusters {
		cluster := &listResp.SKSClusters[ci]

		for ni := range cluster.Nodepools {
			nodepool := &cluster.Nodepools[ni]
			ids = append(ids, nodepool.ID.String())

			item := DataSourceNodepoolListItemModel{
				ID:               types.StringValue(nodepool.ID.String()),
				ClusterID:        types.StringValue(cluster.ID.String()),
				Name:             types.StringValue(nodepool.Name),
				CreatedAt:        types.StringValue(nodepool.CreatedAT.String()),
				Description:      types.StringValue(nodepool.Description),
				DiskSize:         types.Int64Value(nodepool.DiskSize),
				InstancePrefix:   types.StringValue(nodepool.InstancePrefix),
				Size:             types.Int64Value(nodepool.Size),
				State:            types.StringValue(string(nodepool.State)),
				Version:          types.StringValue(nodepool.Version),
				IPv6:             types.BoolValue(nodepool.PublicIPAssignment == exoscale.SKSNodepoolPublicIPAssignmentDual),
				StorageLVM:       types.BoolValue(in(nodepool.Addons, sksNodepoolAddonStorageLVM)),
				NvidiaMigProfile: optionalString(sksNodepoolMIGProfile(nodepool.NvidiaMigProfiles)),
			}

			item.DeployTargetID = types.StringNull()
			if nodepool.DeployTarget != nil {
				item.DeployTargetID = types.StringValue(nodepool.DeployTarget.ID.String())
			}

			item.InstancePoolID = types.StringNull()
			if nodepool.InstancePool != nil {
				item.InstancePoolID = types.StringValue(nodepool.InstancePool.ID.String())
			}

			item.TemplateID = types.StringNull()
			if nodepool.Template != nil {
				item.TemplateID = types.StringValue(nodepool.Template.ID.String())
			}

			item.KubeletMaxPods = types.Int64Null()
			if nodepool.KubeletMaxPods != nil {
				item.KubeletMaxPods = types.Int64Value(*nodepool.KubeletMaxPods)
			}

			item.InstanceType = types.StringNull()
			if nodepool.InstanceType != nil {
				instanceType, err := resolveInstanceType(nodepool.InstanceType.ID)
				if err != nil {
					resp.Diagnostics.AddError("error retrieving instance type", err.Error())
					return
				}
				item.InstanceType = types.StringValue(instanceType)
			}

			aags, dg := types.SetValueFrom(ctx, types.StringType, utils.AntiAffiniGroupsToAntiAffinityGroupIDs(nodepool.AntiAffinityGroups))
			resp.Diagnostics.Append(dg...)
			item.AntiAffinityGroupIDs = aags

			pns, dg := types.SetValueFrom(ctx, types.StringType, utils.PrivateNetworksToPrivateNetworkIDs(nodepool.PrivateNetworks))
			resp.Diagnostics.Append(dg...)
			item.PrivateNetworkIDs = pns

			sgs, dg := types.SetValueFrom(ctx, types.StringType, utils.SecurityGroupsToSecurityGroupIDs(nodepool.SecurityGroups))
			resp.Diagnostics.Append(dg...)
			item.SecurityGroupIDs = sgs

			item.Labels = types.MapNull(types.StringType)
			if len(nodepool.Labels) > 0 {
				labels, dgl := types.MapValueFrom(ctx, types.StringType, nodepool.Labels)
				resp.Diagnostics.Append(dgl...)
				item.Labels = labels
			}

			item.Taints = types.MapNull(types.StringType)
			if len(nodepool.Taints) > 0 {
				taints := make(map[string]string, len(nodepool.Taints))
				for k, v := range nodepool.Taints {
					taints[k] = fmt.Sprintf("%s:%s", v.Value, v.Effect)
				}
				t, dgt := types.MapValueFrom(ctx, types.StringType, taints)
				resp.Diagnostics.Append(dgt...)
				item.Taints = t
			}

			item.KubeletImageGC = types.ObjectNull(kubeletGCAttrTypes())
			if nodepool.KubeletImageGC != nil {
				obj, d := types.ObjectValueFrom(ctx, kubeletGCAttrTypes(), kubeletGCModel{
					MinAge:        optionalString(nodepool.KubeletImageGC.MinAge),
					HighThreshold: types.Int64Value(nodepool.KubeletImageGC.HighThreshold),
					LowThreshold:  types.Int64Value(nodepool.KubeletImageGC.LowThreshold),
				})
				resp.Diagnostics.Append(d...)
				item.KubeletImageGC = obj
			}

			items = append(items, item)
		}
	}

	if resp.Diagnostics.HasError() {
		return
	}

	sort.Strings(ids)
	state.ID = types.StringValue(fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(ids, ""))))) //nolint:gosec
	state.Nodepools = items

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
