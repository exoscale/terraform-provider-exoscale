package instance_pool

import (
	"context"
	"crypto/md5" //nolint:gosec // used only to derive a stable synthetic data source ID, not for security purposes
	"fmt"
	"maps"
	"sort"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const markdownDescriptionDataSourceList = `List Exoscale [Instance Pools](https://community.exoscale.com/product/compute/instances/how-to/instance-pools/).

Corresponding resource: [exoscale_instance_pool](../resources/instance_pool.md).`

var _ datasource.DataSourceWithConfigure = (*DataSourceList)(nil)

type DataSourceList struct {
	client *exoscale.Client
}

func NewDataSourceList() datasource.DataSource {
	return &DataSourceList{}
}

// DataSourceListModel defines the exoscale_instance_pool_list data source data model.
type DataSourceListModel struct {
	ID    types.String              `tfsdk:"id"`
	Zone  types.String              `tfsdk:"zone"`
	Pools []DataSourceListPoolModel `tfsdk:"pools"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// DataSourceListPoolModel maps the elements of the `pools` attribute.
type DataSourceListPoolModel struct {
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
}

func (d *DataSourceList) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance_pool_list"
}

func (d *DataSourceList) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "List Exoscale Instance Pools.",
		MarkdownDescription: markdownDescriptionDataSourceList,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The ID of this resource.",
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
			},
			"zone": schema.StringAttribute{
				Description:         "The Exoscale Zone name.",
				MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"pools": schema.ListNestedAttribute{
				Description:         "The list of exoscale_instance_pool.",
				MarkdownDescription: "The list of [exoscale_instance_pool](./instance_pool.md).",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description:         "The instance pool ID.",
							MarkdownDescription: "The instance pool ID.",
							Computed:            true,
						},
						"affinity_group_ids": schema.SetAttribute{
							Description:         "The list of attached exoscale_anti_affinity_group (IDs). Use anti_affinity_group_ids instead.",
							MarkdownDescription: "The list of attached [exoscale_anti_affinity_group](../resources/anti_affinity_group.md) (IDs). Use `anti_affinity_group_ids` instead.",
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
							Description:         "The list of managed instances.",
							MarkdownDescription: "The list of managed instances.",
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
						"labels": schema.MapAttribute{
							Description:         "A map of key/value labels.",
							MarkdownDescription: "A map of key/value labels.",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"min_available": schema.Int64Attribute{
							Description:         "Minimum number of running Instances.",
							MarkdownDescription: "Minimum number of running Instances.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							Description:         "The instance pool name.",
							MarkdownDescription: "The instance pool name.",
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
						"zone": schema.StringAttribute{
							Description:         "The Exoscale Zone name.",
							MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
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

func (d *DataSourceList) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (d *DataSourceList) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceListModel

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

	zone := data.Zone.ValueString()

	client, err := utils.SwitchClientZone(ctx, d.client, exoscale.ZoneName(zone))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	list, err := client.ListInstancePools(ctx)
	if err != nil {
		resp.Diagnostics.AddError("unable to list instance pools", err.Error())
		return
	}

	ids := make([]string, 0, len(list.InstancePools))
	instanceTypes := map[exoscale.UUID]string{}
	data.Pools = make([]DataSourceListPoolModel, 0, len(list.InstancePools))

	for _, item := range list.InstancePools {
		// we use ID to generate a resource ID, we cannot list instance pools without ID.
		if item.ID == "" {
			continue
		}

		ids = append(ids, item.ID.String())

		pool, err := client.GetInstancePool(ctx, item.ID)
		if err != nil {
			resp.Diagnostics.AddError("unable to get instance pool", err.Error())
			return
		}

		// Many pools tend to share the same instance type.
		instanceType := ""
		if pool.InstanceType != nil {
			if _, ok := instanceTypes[pool.InstanceType.ID]; !ok {
				name, err := poolInstanceType(ctx, client, pool)
				if err != nil {
					resp.Diagnostics.AddError("unable to retrieve instance type", err.Error())
					return
				}
				instanceTypes[pool.InstanceType.ID] = name
			}
			instanceType = instanceTypes[pool.InstanceType.ID]
		}

		model, diags := dataSourceListPool(ctx, client, pool, zone, instanceType)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		data.Pools = append(data.Pools, model)
	}

	// by sorting instance pool IDs we can generate the same resource ID regardless of the order in which
	// API returns instance pools in the list.
	sort.Strings(ids)

	data.ID = types.StringValue(fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(ids, ""))))) //nolint:gosec

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "data source read", map[string]any{
		"id": data.ID.ValueString(),
	})
}

// dataSourceListPool builds an element of `pools` from the API. Unlike
// exoscale_instance_pool, the elements of the list have always reported the
// values a pool does not have as empty strings, sets and maps rather than null.
func dataSourceListPool(
	ctx context.Context,
	client *exoscale.Client,
	pool *exoscale.InstancePool,
	zone string,
	instanceType string,
) (DataSourceListPoolModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	model := DataSourceListPoolModel{
		ID:             types.StringValue(pool.ID.String()),
		Name:           types.StringValue(pool.Name),
		Description:    types.StringValue(pool.Description),
		DiskSize:       types.Int64Value(pool.DiskSize),
		InstancePrefix: types.StringValue(pool.InstancePrefix),
		InstanceType:   types.StringValue(instanceType),
		IPv6:           types.BoolValue(utils.DefaultBool(pool.Ipv6Enabled, false)),
		MinAvailable:   types.Int64Value(pool.MinAvailable),
		Size:           types.Int64Value(pool.Size),
		State:          types.StringValue(string(pool.State)),
		Zone:           types.StringValue(zone),
	}

	model.DeployTargetID = types.StringValue("")
	if pool.DeployTarget != nil {
		model.DeployTargetID = types.StringValue(pool.DeployTarget.ID.String())
	}

	model.KeyPair = types.StringValue("")
	if pool.SSHKey != nil {
		model.KeyPair = types.StringValue(pool.SSHKey.Name)
	}

	model.TemplateID = types.StringValue("")
	if pool.Template != nil {
		model.TemplateID = types.StringValue(pool.Template.ID.String())
	}

	labels := map[string]string{}
	maps.Copy(labels, pool.Labels)

	var dg diag.Diagnostics
	model.Labels, dg = types.MapValueFrom(ctx, types.StringType, labels)
	diags.Append(dg...)

	model.AntiAffinityGroupIDs, dg = types.SetValueFrom(ctx, types.StringType, utils.AntiAffiniGroupsToAntiAffinityGroupIDs(pool.AntiAffinityGroups))
	diags.Append(dg...)
	model.AffinityGroupIDs = model.AntiAffinityGroupIDs

	model.ElasticIPIDs, dg = types.SetValueFrom(ctx, types.StringType, utils.ElasticIPsToElasticIPIDs(pool.ElasticIPS))
	diags.Append(dg...)

	model.NetworkIDs, dg = types.SetValueFrom(ctx, types.StringType, utils.PrivateNetworksToPrivateNetworkIDs(pool.PrivateNetworks))
	diags.Append(dg...)

	model.SecurityGroupIDs, dg = types.SetValueFrom(ctx, types.StringType, utils.SecurityGroupsToSecurityGroupIDs(pool.SecurityGroups))
	diags.Append(dg...)

	if diags.HasError() {
		return model, diags
	}

	model.UserData = types.StringValue("")
	if pool.UserData != "" {
		userData, err := utils.DecodeUserData(pool.UserData)
		if err != nil {
			diags.AddError("unable to decode user data", err.Error())
			return model, diags
		}
		model.UserData = types.StringValue(userData)
	}

	instances, _, dg := poolInstances(ctx, client, pool)
	diags.Append(dg...)
	model.Instances = instances

	return model, diags
}
