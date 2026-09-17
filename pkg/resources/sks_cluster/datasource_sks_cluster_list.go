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

const markdownDescriptionDataSourceClusterList = `List Exoscale [Scalable Kubernetes Service (SKS)](https://community.exoscale.com/product/compute/containers/) Clusters in a zone.

Corresponding resource: [exoscale_sks_cluster](../resources/sks_cluster.md).`

var _ datasource.DataSourceWithConfigure = (*DataSourceClusterList)(nil)

type DataSourceClusterList struct {
	client *exoscale.Client
}

func NewDataSourceClusterList() datasource.DataSource {
	return &DataSourceClusterList{}
}

type DataSourceClusterListModel struct {
	ID       types.String                     `tfsdk:"id"`
	Zone     types.String                     `tfsdk:"zone"`
	Clusters []DataSourceClusterListItemModel `tfsdk:"clusters"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type DataSourceClusterListItemModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`

	Addons                 types.Set    `tfsdk:"addons"`
	AutoUpgrade            types.Bool   `tfsdk:"auto_upgrade"`
	CNI                    types.String `tfsdk:"cni"`
	CreatedAt              types.String `tfsdk:"created_at"`
	DefaultSecurityGroupID types.String `tfsdk:"default_security_group_id"`
	Description            types.String `tfsdk:"description"`
	EnableKubeProxy        types.Bool   `tfsdk:"enable_kube_proxy"`
	Endpoint               types.String `tfsdk:"endpoint"`
	FeatureGates           types.Set    `tfsdk:"feature_gates"`
	Labels                 types.Map    `tfsdk:"labels"`
	Nodepools              types.Set    `tfsdk:"nodepools"`
	ServiceLevel           types.String `tfsdk:"service_level"`
	State                  types.String `tfsdk:"state"`
	Version                types.String `tfsdk:"version"`
}

func (d *DataSourceClusterList) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sks_cluster_list"
}

func (d *DataSourceClusterList) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: markdownDescriptionDataSourceClusterList,

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
			"clusters": schema.ListNestedAttribute{
				Description:         "The list of SKS clusters in the zone.",
				MarkdownDescription: "The list of [exoscale_sks_cluster](./sks_cluster.md) in the zone.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description:         "The SKS cluster ID.",
							MarkdownDescription: "The SKS cluster ID.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							Description:         "The SKS cluster name.",
							MarkdownDescription: "The SKS cluster name.",
							Computed:            true,
						},
						"addons": schema.SetAttribute{
							Description:         "The list of enabled add-ons.",
							MarkdownDescription: "The list of enabled add-ons.",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"auto_upgrade": schema.BoolAttribute{
							Description:         "Enable automatic upgrading of the control plane version.",
							MarkdownDescription: "Enable automatic upgrading of the control plane version.",
							Computed:            true,
						},
						"cni": schema.StringAttribute{
							Description:         "The CNI plugin that is to be used.",
							MarkdownDescription: "The CNI plugin that is to be used.",
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							Description:         "The cluster creation date.",
							MarkdownDescription: "The cluster creation date.",
							Computed:            true,
						},
						"default_security_group_id": schema.StringAttribute{
							Description:         "The ID of the cluster's ad-hoc default security group.",
							MarkdownDescription: "The ID of the cluster's ad-hoc default security group.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							Description:         "A free-form text describing the cluster.",
							MarkdownDescription: "A free-form text describing the cluster.",
							Computed:            true,
						},
						"enable_kube_proxy": schema.BoolAttribute{
							Description:         "Indicates whether the Kubernetes network proxy is deployed.",
							MarkdownDescription: "Indicates whether the Kubernetes network proxy is deployed.",
							Computed:            true,
						},
						"endpoint": schema.StringAttribute{
							Description:         "The cluster API endpoint.",
							MarkdownDescription: "The cluster API endpoint.",
							Computed:            true,
						},
						"feature_gates": schema.SetAttribute{
							Description:         "Feature gates options for the cluster.",
							MarkdownDescription: "Feature gates options for the cluster.",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"labels": schema.MapAttribute{
							Description:         "A map of key/value labels.",
							MarkdownDescription: "A map of key/value labels.",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"nodepools": schema.SetAttribute{
							Description:         "The list of exoscale_sks_nodepool (IDs) attached to the cluster.",
							MarkdownDescription: "The list of [exoscale_sks_nodepool](./sks_nodepool.md) (IDs) attached to the cluster.",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"service_level": schema.StringAttribute{
							Description:         "The service level of the control plane.",
							MarkdownDescription: "The service level of the control plane.",
							Computed:            true,
						},
						"state": schema.StringAttribute{
							Description:         "The cluster state.",
							MarkdownDescription: "The cluster state.",
							Computed:            true,
						},
						"version": schema.StringAttribute{
							Description:         "The version of the control plane.",
							MarkdownDescription: "The version of the control plane.",
							Computed:            true,
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

func (d *DataSourceClusterList) Configure(ctx context.Context, r datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if r.ProviderData == nil {
		return
	}

	d.client = r.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (d *DataSourceClusterList) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DataSourceClusterListModel

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

	ids := make([]string, len(listResp.SKSClusters))
	items := make([]DataSourceClusterListItemModel, len(listResp.SKSClusters))
	for i := range listResp.SKSClusters {
		cluster := &listResp.SKSClusters[i]
		ids[i] = cluster.ID.String()

		item := DataSourceClusterListItemModel{
			ID:           types.StringValue(cluster.ID.String()),
			Name:         types.StringValue(cluster.Name),
			CNI:          types.StringValue(string(cluster.Cni)),
			CreatedAt:    types.StringValue(cluster.CreatedAT.String()),
			Description:  types.StringValue(cluster.Description),
			Endpoint:     types.StringValue(cluster.Endpoint),
			ServiceLevel: types.StringValue(string(cluster.Level)),
			State:        types.StringValue(string(cluster.State)),
			Version:      types.StringValue(cluster.Version),
		}

		item.AutoUpgrade = types.BoolNull()
		if cluster.AutoUpgrade != nil {
			item.AutoUpgrade = types.BoolValue(*cluster.AutoUpgrade)
		}

		item.EnableKubeProxy = types.BoolNull()
		if cluster.EnableKubeProxy != nil {
			item.EnableKubeProxy = types.BoolValue(*cluster.EnableKubeProxy)
		}

		item.DefaultSecurityGroupID = types.StringNull()
		if cluster.DefaultSecurityGroupID != nil {
			item.DefaultSecurityGroupID = types.StringValue(cluster.DefaultSecurityGroupID.String())
		}

		addons, dg := types.SetValueFrom(ctx, types.StringType, sliceOrEmpty(cluster.Addons))
		resp.Diagnostics.Append(dg...)
		item.Addons = addons

		featureGates, dg := types.SetValueFrom(ctx, types.StringType, sliceOrEmpty(cluster.FeatureGates))
		resp.Diagnostics.Append(dg...)
		item.FeatureGates = featureGates

		nodepools := make([]string, len(cluster.Nodepools))
		for ni, np := range cluster.Nodepools {
			nodepools[ni] = np.ID.String()
		}
		nps, dg := types.SetValueFrom(ctx, types.StringType, nodepools)
		resp.Diagnostics.Append(dg...)
		item.Nodepools = nps

		item.Labels = types.MapNull(types.StringType)
		if len(cluster.Labels) > 0 {
			labels, dgl := types.MapValueFrom(ctx, types.StringType, cluster.Labels)
			resp.Diagnostics.Append(dgl...)
			item.Labels = labels
		}

		items[i] = item
	}

	if resp.Diagnostics.HasError() {
		return
	}

	sort.Strings(ids)
	state.ID = types.StringValue(fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(ids, ""))))) //nolint:gosec
	state.Clusters = items

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
