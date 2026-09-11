package sks_cluster

import (
	"context"

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

const markdownDescriptionDataSourceCluster = `Fetch Exoscale [Scalable Kubernetes Service (SKS)](https://community.exoscale.com/product/compute/containers/) Cluster data.

Corresponding resource: [exoscale_sks_cluster](../resources/sks_cluster.md).`

var _ datasource.DataSourceWithConfigure = (*DataSourceCluster)(nil)

type DataSourceCluster struct {
	client *exoscale.Client
}

func NewDataSource() datasource.DataSource {
	return &DataSourceCluster{}
}

type DataSourceClusterModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Zone types.String `tfsdk:"zone"`

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

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *DataSourceCluster) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sks_cluster"
}

func (d *DataSourceCluster) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: markdownDescriptionDataSourceCluster,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The SKS cluster ID to match (conflicts with 'name').",
				MarkdownDescription: "The SKS cluster ID to match (conflicts with `name`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("name"),
					}...),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The SKS cluster name to match (conflicts with 'id').",
				MarkdownDescription: "The SKS cluster name to match (conflicts with `id`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("id"),
					}...),
				},
			},
			"zone": schema.StringAttribute{
				Description:         "The Exoscale zone name.",
				MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"addons": schema.SetAttribute{
				MarkdownDescription: "The list of enabled add-ons.",
				Description:         "The list of enabled add-ons.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"auto_upgrade": schema.BoolAttribute{
				MarkdownDescription: "Enable automatic upgrading of the control plane version.",
				Description:         "Enable automatic upgrading of the control plane version.",
				Computed:            true,
			},
			"cni": schema.StringAttribute{
				MarkdownDescription: "The CNI plugin that is to be used.",
				Description:         "The CNI plugin that is to be used.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The cluster creation date.",
				Description:         "The cluster creation date.",
				Computed:            true,
			},
			"default_security_group_id": schema.StringAttribute{
				MarkdownDescription: "The ID of the cluster's ad-hoc default security group.",
				Description:         "The ID of the cluster's ad-hoc default security group.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A free-form text describing the cluster.",
				Description:         "A free-form text describing the cluster.",
				Computed:            true,
			},
			"enable_kube_proxy": schema.BoolAttribute{
				MarkdownDescription: "Indicates whether the Kubernetes network proxy is deployed.",
				Description:         "Indicates whether the Kubernetes network proxy is deployed.",
				Computed:            true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "The cluster API endpoint.",
				Description:         "The cluster API endpoint.",
				Computed:            true,
			},
			"feature_gates": schema.SetAttribute{
				MarkdownDescription: "Feature gates options for the cluster.",
				Description:         "Feature gates options for the cluster.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"labels": schema.MapAttribute{
				MarkdownDescription: "A map of key/value labels.",
				Description:         "A map of key/value labels.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"nodepools": schema.SetAttribute{
				MarkdownDescription: "The list of [exoscale_sks_nodepool](./sks_nodepool.md) (IDs) attached to the cluster.",
				Description:         "The list of exoscale_sks_nodepool (IDs) attached to the cluster.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"service_level": schema.StringAttribute{
				MarkdownDescription: "The service level of the control plane.",
				Description:         "The service level of the control plane.",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "The cluster state.",
				Description:         "The cluster state.",
				Computed:            true,
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "The version of the control plane.",
				Description:         "The version of the control plane.",
				Computed:            true,
			},
		},

		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Read: true,
			}),
		},
	}
}

func (d *DataSourceCluster) Configure(ctx context.Context, r datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if r.ProviderData == nil {
		return
	}

	d.client = r.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (d *DataSourceCluster) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DataSourceClusterModel

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

	var cluster *exoscale.SKSCluster
	switch {
	case !state.ID.IsNull():
		cluster, err = client.GetSKSCluster(ctx, exoscale.UUID(state.ID.ValueString()))
		if err != nil {
			resp.Diagnostics.AddError("error getting cluster", err.Error())
			return
		}
	case !state.Name.IsNull():
		clusters, err := client.ListSKSClusters(ctx)
		if err != nil {
			resp.Diagnostics.AddError("error listing clusters", err.Error())
			return
		}
		found, err := clusters.FindSKSCluster(state.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("error finding cluster", err.Error())
			return
		}
		cluster = &found
	default: // validation prevents this, exit as a safe guard
		resp.Diagnostics.AddError("missing values", "id and name are missing")
		return
	}

	state.ID = types.StringValue(cluster.ID.String())
	state.Name = types.StringValue(cluster.Name)
	state.CNI = types.StringValue(string(cluster.Cni))
	state.CreatedAt = types.StringValue(cluster.CreatedAT.String())
	state.Description = types.StringValue(cluster.Description)
	state.Endpoint = types.StringValue(cluster.Endpoint)
	state.ServiceLevel = types.StringValue(string(cluster.Level))
	state.State = types.StringValue(string(cluster.State))
	state.Version = types.StringValue(cluster.Version)

	state.AutoUpgrade = types.BoolNull()
	if cluster.AutoUpgrade != nil {
		state.AutoUpgrade = types.BoolValue(*cluster.AutoUpgrade)
	}

	state.EnableKubeProxy = types.BoolNull()
	if cluster.EnableKubeProxy != nil {
		state.EnableKubeProxy = types.BoolValue(*cluster.EnableKubeProxy)
	}

	state.DefaultSecurityGroupID = types.StringNull()
	if cluster.DefaultSecurityGroupID != nil {
		state.DefaultSecurityGroupID = types.StringValue(cluster.DefaultSecurityGroupID.String())
	}

	addons, dg := types.SetValueFrom(ctx, types.StringType, sliceOrEmpty(cluster.Addons))
	resp.Diagnostics.Append(dg...)
	state.Addons = addons

	featureGates, dg := types.SetValueFrom(ctx, types.StringType, sliceOrEmpty(cluster.FeatureGates))
	resp.Diagnostics.Append(dg...)
	state.FeatureGates = featureGates

	nodepools := make([]string, len(cluster.Nodepools))
	for i, np := range cluster.Nodepools {
		nodepools[i] = np.ID.String()
	}
	nps, dg := types.SetValueFrom(ctx, types.StringType, nodepools)
	resp.Diagnostics.Append(dg...)
	state.Nodepools = nps

	state.Labels = types.MapNull(types.StringType)
	if len(cluster.Labels) > 0 {
		labels, dgl := types.MapValueFrom(ctx, types.StringType, cluster.Labels)
		resp.Diagnostics.Append(dgl...)
		state.Labels = labels
	}

	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
