package instance_pool

import (
	"context"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const markdownDescriptionDataSource = `Fetch Exoscale [Instance Pools](https://community.exoscale.com/product/compute/instances/how-to/instance-pools/) data.

Corresponding resource: [exoscale_instance_pool](../resources/instance_pool.md).`

var _ datasource.DataSourceWithConfigure = (*DataSource)(nil)

type DataSource struct {
	client *exoscale.Client
}

func NewDataSource() datasource.DataSource {
	return &DataSource{}
}

// DataSourceModel defines the exoscale_instance_pool data source data model.
type DataSourceModel struct {
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
	Zone                 types.String `tfsdk:"zone"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *DataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_pool"
}

func (d *DataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Fetch Exoscale Instance Pools data.",
		MarkdownDescription: markdownDescriptionDataSource,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The instance pool ID to match (conflicts with 'name').",
				MarkdownDescription: "The instance pool ID to match (conflicts with `name`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("name")),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The pool name to match (conflicts with 'id').",
				MarkdownDescription: "The pool name to match (conflicts with `id`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("id")),
				},
			},
			"zone": schema.StringAttribute{
				Description:         "The Exoscale Zone name.",
				MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"affinity_group_ids": schema.SetAttribute{
				Description:         "The list of attached exoscale_anti_affinity_group (IDs). Use anti_affinity_group_ids instead.",
				MarkdownDescription: "The list of attached [exoscale_anti_affinity_group](../resources/anti_affinity_group.md) (IDs). Use anti_affinity_group_ids instead.",
				DeprecationMessage:  "Use anti_affinity_group_ids instead.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"anti_affinity_group_ids": schema.SetAttribute{
				Description:         "The list of attached exoscale_anti_affinity_group (IDs).",
				MarkdownDescription: "The list of attached [exoscale_anti_affinity_group](../resources/anti_affinity_group.md) (IDs).",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"deploy_target_id": schema.StringAttribute{
				Description:         "The deploy target ID.",
				MarkdownDescription: "The deploy target ID.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				Description:         "The instance pool description.",
				MarkdownDescription: "The instance pool description.",
				Computed:            true,
			},
			"disk_size": schema.Int64Attribute{
				Description:         "The managed instances disk size.",
				MarkdownDescription: "The managed instances disk size.",
				Computed:            true,
			},
			"elastic_ip_ids": schema.SetAttribute{
				Description:         "The list of attached exoscale_elastic_ip (IDs).",
				MarkdownDescription: "The list of attached [exoscale_elastic_ip](../resources/elastic_ip.md) (IDs).",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"instance_prefix": schema.StringAttribute{
				Description:         "The string used to prefix the managed instances name.",
				MarkdownDescription: "The string used to prefix the managed instances name.",
				Computed:            true,
			},
			"instance_type": schema.StringAttribute{
				Description:         "The managed instances type.",
				MarkdownDescription: "The managed instances type.",
				Computed:            true,
			},
			"instances": schema.SetNestedAttribute{
				Description:         "The list of managed instances. Structure is documented below.",
				MarkdownDescription: "The list of managed instances. Structure is documented below.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description:         "The compute instance ID.",
							MarkdownDescription: "The compute instance ID.",
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
				Description:         "Whether IPv6 is enabled on managed instances.",
				MarkdownDescription: "Whether IPv6 is enabled on managed instances.",
				Computed:            true,
			},
			"key_pair": schema.StringAttribute{
				Description:         "The exoscale_ssh_key (name) authorized on the managed instances.",
				MarkdownDescription: "The [exoscale_ssh_key](../resources/ssh_key.md) (name) authorized on the managed instances.",
				Computed:            true,
			},
			// Optional with the SDKv2 too, although it has never been a filter.
			"labels": schema.MapAttribute{
				Description:         "A map of key/value labels.",
				MarkdownDescription: "A map of key/value labels.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
			},
			"min_available": schema.Int64Attribute{
				Description:         "Minimum number of running Instances.",
				MarkdownDescription: "Minimum number of running Instances.",
				Computed:            true,
			},
			"network_ids": schema.SetAttribute{
				Description:         "The list of attached exoscale_private_network (IDs).",
				MarkdownDescription: "The list of attached [exoscale_private_network](../resources/private_network.md) (IDs).",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"security_group_ids": schema.SetAttribute{
				Description:         "The list of attached exoscale_security_group (IDs).",
				MarkdownDescription: "The list of attached [exoscale_security_group](../resources/security_group.md) (IDs).",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"size": schema.Int64Attribute{
				Description:         "The number managed instances.",
				MarkdownDescription: "The number managed instances.",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				Description:         "The pool state.",
				MarkdownDescription: "The pool state.",
				Computed:            true,
			},
			"template_id": schema.StringAttribute{
				Description:         "The managed instances exoscale_template ID.",
				MarkdownDescription: "The managed instances [exoscale_template](./template.md) ID.",
				Computed:            true,
			},
			"user_data": schema.StringAttribute{
				Description:         "cloud-init configuration.",
				MarkdownDescription: "[cloud-init](http://cloudinit.readthedocs.io/en/latest/) configuration.",
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

func (d *DataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (d *DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := data.Timeouts.Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := utils.SwitchClientZone(ctx, d.client, exoscale.ZoneName(data.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	var id exoscale.UUID
	if data.ID.IsNull() {
		list, err := client.ListInstancePools(ctx)
		if err != nil {
			resp.Diagnostics.AddError("unable to list instance pools", err.Error())
			return
		}

		found, err := list.FindInstancePool(data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("unable to find instance pool", err.Error())
			return
		}
		id = found.ID
	} else {
		id, err = exoscale.ParseUUID(data.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("unable to parse instance pool ID", err.Error())
			return
		}
	}

	pool, err := client.GetInstancePool(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("unable to get instance pool", err.Error())
		return
	}

	resp.Diagnostics.Append(applyDataSourcePool(ctx, client, &data, pool)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "data source read", map[string]any{
		"id": data.ID.ValueString(),
	})
}

// applyDataSourcePool fills in the data source model from the API. Values the
// SDKv2 implementation didn't set are null, as they were in its state.
func applyDataSourcePool( //nolint:gocyclo
	ctx context.Context,
	client *exoscale.Client,
	data *DataSourceModel,
	pool *exoscale.InstancePool,
) diag.Diagnostics {
	var diags diag.Diagnostics

	data.ID = types.StringValue(pool.ID.String())
	data.Name = types.StringValue(pool.Name)
	data.Description = types.StringValue(pool.Description)
	data.DiskSize = types.Int64Value(pool.DiskSize)
	data.InstancePrefix = types.StringValue(pool.InstancePrefix)
	data.IPv6 = types.BoolValue(utils.DefaultBool(pool.Ipv6Enabled, false))
	data.MinAvailable = types.Int64Value(pool.MinAvailable)
	data.Size = types.Int64Value(pool.Size)
	data.State = types.StringValue(string(pool.State))

	data.DeployTargetID = types.StringNull()
	if pool.DeployTarget != nil {
		data.DeployTargetID = types.StringValue(pool.DeployTarget.ID.String())
	}

	data.KeyPair = types.StringNull()
	if pool.SSHKey != nil {
		data.KeyPair = types.StringValue(pool.SSHKey.Name)
	}

	data.TemplateID = types.StringNull()
	if pool.Template != nil {
		data.TemplateID = types.StringValue(pool.Template.ID.String())
	}

	data.Labels = types.MapNull(types.StringType)
	if pool.Labels != nil {
		labels, dg := types.MapValueFrom(ctx, types.StringType, pool.Labels)
		diags.Append(dg...)
		data.Labels = labels
	}

	data.AntiAffinityGroupIDs = types.SetNull(types.StringType)
	data.AffinityGroupIDs = types.SetNull(types.StringType)
	if pool.AntiAffinityGroups != nil {
		ids, dg := types.SetValueFrom(ctx, types.StringType, utils.AntiAffiniGroupsToAntiAffinityGroupIDs(pool.AntiAffinityGroups))
		diags.Append(dg...)
		data.AntiAffinityGroupIDs = ids
		data.AffinityGroupIDs = ids
	}

	data.ElasticIPIDs = types.SetNull(types.StringType)
	if pool.ElasticIPS != nil {
		ids, dg := types.SetValueFrom(ctx, types.StringType, utils.ElasticIPsToElasticIPIDs(pool.ElasticIPS))
		diags.Append(dg...)
		data.ElasticIPIDs = ids
	}

	data.NetworkIDs = types.SetNull(types.StringType)
	if pool.PrivateNetworks != nil {
		ids, dg := types.SetValueFrom(ctx, types.StringType, utils.PrivateNetworksToPrivateNetworkIDs(pool.PrivateNetworks))
		diags.Append(dg...)
		data.NetworkIDs = ids
	}

	data.SecurityGroupIDs = types.SetNull(types.StringType)
	if pool.SecurityGroups != nil {
		ids, dg := types.SetValueFrom(ctx, types.StringType, utils.SecurityGroupsToSecurityGroupIDs(pool.SecurityGroups))
		diags.Append(dg...)
		data.SecurityGroupIDs = ids
	}

	if diags.HasError() {
		return diags
	}

	data.UserData = types.StringNull()
	if pool.UserData != "" {
		userData, err := utils.DecodeUserData(pool.UserData)
		if err != nil {
			diags.AddError("unable to decode user data", err.Error())
			return diags
		}
		data.UserData = types.StringValue(userData)
	}

	instanceType, err := poolInstanceType(ctx, client, pool)
	if err != nil {
		diags.AddError("unable to retrieve instance type", err.Error())
		return diags
	}
	data.InstanceType = types.StringValue(instanceType)

	data.Instances = types.SetNull(instanceObjectType)
	if pool.Instances != nil {
		instances, _, dg := poolInstances(ctx, client, pool)
		diags.Append(dg...)
		data.Instances = instances
	}

	return diags
}
