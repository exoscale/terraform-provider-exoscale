package instance

import (
	"context"
	"crypto/md5"
	"fmt"
	"sort"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/filter"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	markdownDescriptionDatasourceList = `List Exoscale [Compute Instances](https://community.exoscale.com/documentation/compute/).

Corresponding resource: [exoscale_compute_instance](../resources/compute_instance.md).`

	filterDescriptionBool   = "Match against this bool"
	filterDescriptionInt    = "Match against this int"
	filterDescriptionString = `Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.`
	filterDescriptionMap    = `Match against key/values. Keys are matched exactly, while values may be matched as a regex if you supply a string that begins and ends with "/"`
)

var _ datasource.DataSource = (*DataSourceList)(nil)
var _ datasource.DataSourceWithConfigure = (*DataSourceList)(nil)

// DataSourceList is the exoscale_compute_instance_list data source implementation.
type DataSourceList struct {
	client *exoscale.Client
}

// NewDataSourceList creates an instance of DataSourceList.
func NewDataSourceList() datasource.DataSource {
	return &DataSourceList{}
}

// DataSourceListModel defines the exoscale_compute_instance_list data model.
// Apart from `zone`, `instances` and `timeouts`, the attributes are filters.
type DataSourceListModel struct {
	CreatedAt        types.String `tfsdk:"created_at"`
	DeployTargetID   types.String `tfsdk:"deploy_target_id"`
	DiskSize         types.Int64  `tfsdk:"disk_size"`
	EnableSecureBoot types.Bool   `tfsdk:"enable_secure_boot"`
	EnableTPM        types.Bool   `tfsdk:"enable_tpm"`
	ID               types.String `tfsdk:"id"`
	IPv6             types.Bool   `tfsdk:"ipv6"`
	IPv6Address      types.String `tfsdk:"ipv6_address"`
	Labels           types.Map    `tfsdk:"labels"`
	ManagerID        types.String `tfsdk:"manager_id"`
	ManagerType      types.String `tfsdk:"manager_type"`
	Name             types.String `tfsdk:"name"`
	PublicIPAddress  types.String `tfsdk:"public_ip_address"`
	ReverseDNS       types.String `tfsdk:"reverse_dns"`
	SSHKey           types.String `tfsdk:"ssh_key"`
	State            types.String `tfsdk:"state"`
	TemplateID       types.String `tfsdk:"template_id"`
	Type             types.String `tfsdk:"type"`
	UserData         types.String `tfsdk:"user_data"`
	Zone             types.String `tfsdk:"zone"`

	Instances []InstanceModel `tfsdk:"instances"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *DataSourceList) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compute_instance_list"
}

func (d *DataSourceList) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "List Exoscale Compute Instances.",
		MarkdownDescription: markdownDescriptionDatasourceList,

		Attributes: map[string]schema.Attribute{
			"zone": schema.StringAttribute{
				Description:         "The Exoscale Zone name.",
				MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"instances": schema.ListNestedAttribute{
				Description:         "The list of exoscale_compute_instance.",
				MarkdownDescription: "The list of [exoscale_compute_instance](./compute_instance.md).",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"anti_affinity_group_ids": schema.SetAttribute{
							Description:         "The list of attached exoscale_anti_affinity_group (IDs).",
							MarkdownDescription: "The list of attached [exoscale_anti_affinity_group](../resources/anti_affinity_group.md) (IDs).",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"created_at": schema.StringAttribute{
							Description:         "The compute instance creation date.",
							MarkdownDescription: "The compute instance creation date.",
							Computed:            true,
						},
						"deploy_target_id": schema.StringAttribute{
							Description:         "A deploy target ID.",
							MarkdownDescription: "A deploy target ID.",
							Computed:            true,
						},
						"disk_size": schema.Int64Attribute{
							Description:         "The instance disk size (GiB).",
							MarkdownDescription: "The instance disk size (GiB).",
							Computed:            true,
						},
						"elastic_ip_ids": schema.SetAttribute{
							Description:         "The list of attached exoscale_elastic_ip (IDs).",
							MarkdownDescription: "The list of attached [exoscale_elastic_ip](../resources/elastic_ip.md) (IDs).",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"enable_secure_boot": schema.BoolAttribute{
							Description:         "Indicates if the instance has secure boot enabled.",
							MarkdownDescription: "Indicates if the instance has secure boot enabled.",
							Computed:            true,
						},
						"enable_tpm": schema.BoolAttribute{
							Description:         "Indicates if the instance has TPM enabled.",
							MarkdownDescription: "Indicates if the instance has TPM enabled.",
							Computed:            true,
						},
						"id": schema.StringAttribute{
							Description:         "The compute instance ID.",
							MarkdownDescription: "The compute instance ID.",
							Computed:            true,
						},
						"ipv6": schema.BoolAttribute{
							Description:         "Whether IPv6 is enabled on the instance.",
							MarkdownDescription: "Whether IPv6 is enabled on the instance.",
							Computed:            true,
						},
						"ipv6_address": schema.StringAttribute{
							Description:         "The instance (main network interface) IPv6 address (if enabled).",
							MarkdownDescription: "The instance (main network interface) IPv6 address (if enabled).",
							Computed:            true,
						},
						"labels": schema.MapAttribute{
							Description:         "A map of key/value labels.",
							MarkdownDescription: "A map of key/value labels.",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"manager_id": schema.StringAttribute{
							Description:         "The instance manager ID, if any.",
							MarkdownDescription: "The instance manager ID, if any.",
							Computed:            true,
						},
						"manager_type": schema.StringAttribute{
							Description:         "The instance manager type (instance pool, SKS node pool, etc.), if any.",
							MarkdownDescription: "The instance manager type (instance pool, SKS node pool, etc.), if any.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							Description:         "The instance name.",
							MarkdownDescription: "The instance name.",
							Computed:            true,
						},
						"private_network_ids": schema.SetAttribute{
							Description:         "The list of attached exoscale_private_network (IDs).",
							MarkdownDescription: "The list of attached [exoscale_private_network](../resources/private_network.md) (IDs).",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"public_ip_address": schema.StringAttribute{
							Description:         "The instance (main network interface) IPv4 address.",
							MarkdownDescription: "The instance (main network interface) IPv4 address.",
							Computed:            true,
						},
						"reverse_dns": schema.StringAttribute{
							Description:         "Domain name for reverse DNS record.",
							MarkdownDescription: "Domain name for reverse DNS record.",
							Computed:            true,
						},
						"ssh_key": schema.StringAttribute{
							Description:         "The exoscale_ssh_key (name) authorized on the instance.",
							MarkdownDescription: "The [exoscale_ssh_key](../resources/ssh_key.md) (name) authorized on the instance.",
							DeprecationMessage:  "Use ssh_keys instead",
							Computed:            true,
						},
						"ssh_keys": schema.SetAttribute{
							Description:         "The list of exoscale_ssh_key (name) authorized on the instance.",
							MarkdownDescription: "The list of [exoscale_ssh_key](../resources/ssh_key.md) (name) authorized on the instance.",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"security_group_ids": schema.SetAttribute{
							Description:         "The list of attached exoscale_security_group (IDs).",
							MarkdownDescription: "The list of attached [exoscale_security_group](../resources/security_group.md) (IDs).",
							ElementType:         types.StringType,
							Computed:            true,
						},
						"state": schema.StringAttribute{
							Description:         "The instance state.",
							MarkdownDescription: "The instance state.",
							Computed:            true,
						},
						"template_id": schema.StringAttribute{
							Description:         "The instance exoscale_template ID.",
							MarkdownDescription: "The instance [exoscale_template](./template.md) ID.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							Description:         "The instance type.",
							MarkdownDescription: "The instance type.",
							Computed:            true,
						},
						"user_data": schema.StringAttribute{
							Description:         "The instance cloud-init configuration.",
							MarkdownDescription: "The instance [cloud-init](http://cloudinit.readthedocs.io/en/latest/) configuration.",
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

			// A filter exists for every bool, int, string and map of strings
			// attribute of an instance.
			"created_at": schema.StringAttribute{
				Description:         filterDescriptionString,
				MarkdownDescription: filterDescriptionString,
				Optional:            true,
			},
			"deploy_target_id": schema.StringAttribute{
				Description:         filterDescriptionString,
				MarkdownDescription: filterDescriptionString,
				Optional:            true,
			},
			"disk_size": schema.Int64Attribute{
				Description:         filterDescriptionInt,
				MarkdownDescription: filterDescriptionInt,
				Optional:            true,
			},
			"enable_secure_boot": schema.BoolAttribute{
				Description:         filterDescriptionBool,
				MarkdownDescription: filterDescriptionBool,
				Optional:            true,
			},
			"enable_tpm": schema.BoolAttribute{
				Description:         filterDescriptionBool,
				MarkdownDescription: filterDescriptionBool,
				Optional:            true,
			},
			"id": schema.StringAttribute{
				Description:         filterDescriptionString,
				MarkdownDescription: filterDescriptionString,
				Optional:            true,
				Computed:            true,
			},
			"ipv6": schema.BoolAttribute{
				Description:         filterDescriptionBool,
				MarkdownDescription: filterDescriptionBool,
				Optional:            true,
			},
			"ipv6_address": schema.StringAttribute{
				Description:         filterDescriptionString,
				MarkdownDescription: filterDescriptionString,
				Optional:            true,
			},
			"labels": schema.MapAttribute{
				Description:         filterDescriptionMap,
				MarkdownDescription: filterDescriptionMap,
				ElementType:         types.StringType,
				Optional:            true,
			},
			"manager_id": schema.StringAttribute{
				Description:         filterDescriptionString,
				MarkdownDescription: filterDescriptionString,
				Optional:            true,
			},
			"manager_type": schema.StringAttribute{
				Description:         filterDescriptionString,
				MarkdownDescription: filterDescriptionString,
				Optional:            true,
			},
			"name": schema.StringAttribute{
				Description:         filterDescriptionString,
				MarkdownDescription: filterDescriptionString,
				Optional:            true,
			},
			"public_ip_address": schema.StringAttribute{
				Description:         filterDescriptionString,
				MarkdownDescription: filterDescriptionString,
				Optional:            true,
			},
			"reverse_dns": schema.StringAttribute{
				Description:         filterDescriptionString,
				MarkdownDescription: filterDescriptionString,
				Optional:            true,
			},
			"ssh_key": schema.StringAttribute{
				Description:         filterDescriptionString,
				MarkdownDescription: filterDescriptionString,
				Optional:            true,
			},
			"state": schema.StringAttribute{
				Description:         filterDescriptionString,
				MarkdownDescription: filterDescriptionString,
				Optional:            true,
			},
			"template_id": schema.StringAttribute{
				Description:         filterDescriptionString,
				MarkdownDescription: filterDescriptionString,
				Optional:            true,
			},
			"type": schema.StringAttribute{
				Description:         filterDescriptionString,
				MarkdownDescription: filterDescriptionString,
				Optional:            true,
			},
			"user_data": schema.StringAttribute{
				Description:         filterDescriptionString,
				MarkdownDescription: filterDescriptionString,
				Optional:            true,
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
	var state DataSourceListModel

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

	zone := state.Zone.ValueString()

	client, err := utils.SwitchClientZone(ctx, d.client, exoscale.ZoneName(zone))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	filters, diags := state.filters(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	instances, err := client.ListInstances(ctx)
	if err != nil {
		resp.Diagnostics.AddError("API returned an error while fetching instances", err.Error())
		return
	}

	var (
		ids           = make([]string, 0, len(instances.Instances))
		instanceTypes = map[exoscale.UUID]string{}

		// To save time the reverse DNS is only fetched before filtering if it
		// is filtered on, and for the instances passing the filters otherwise.
		reverseDNSFiltered = state.ReverseDNS.ValueString() != ""
	)

	state.Instances = make([]InstanceModel, 0, len(instances.Instances))

	for _, listed := range instances.Instances {
		// we use ID to generate a resource ID, we cannot list instances without ID.
		if listed.ID == "" {
			continue
		}

		ids = append(ids, listed.ID.String())

		instance, err := client.GetInstance(ctx, listed.ID)
		if err != nil {
			resp.Diagnostics.AddError("unable to retrieve instance", err.Error())
			return
		}

		model, diags := instanceModel(ctx, instance, zone)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Unlike exoscale_compute_instance, the entries of the list have
		// always reported the attributes an instance does not have as empty
		// strings.
		for _, v := range []*types.String{
			&model.IPv6Address,
			&model.ManagerID,
			&model.ManagerType,
			&model.PublicIPAddress,
			&model.ReverseDNS,
			&model.SSHKey,
			&model.TemplateID,
			&model.Type,
			&model.UserData,
		} {
			if v.IsNull() {
				*v = types.StringValue("")
			}
		}

		if reverseDNSFiltered {
			rdns, err := reverseDNS(ctx, client, instance.ID)
			if err != nil {
				resp.Diagnostics.AddError("unable to retrieve instance reverse-dns", err.Error())
				return
			}
			model.ReverseDNS = types.StringValue(rdns)
		}

		// The API returns the instance type as a UUID.
		// We lazily convert it to the <family>.<size> format.
		if instance.InstanceType != nil && instance.InstanceType.ID != "" {
			tid := instance.InstanceType.ID
			if _, ok := instanceTypes[tid]; !ok {
				instanceType, err := client.GetInstanceType(ctx, tid)
				if err != nil {
					resp.Diagnostics.AddError("unable to retrieve instance type", err.Error())
					return
				}
				instanceTypes[tid] = instanceTypeName(instanceType)
			}

			model.Type = types.StringValue(instanceTypes[tid])
		}

		if !filter.CheckForMatch(filterData(model, instance.Labels), filters) {
			continue
		}

		if !reverseDNSFiltered {
			rdns, err := reverseDNS(ctx, client, instance.ID)
			if err != nil {
				resp.Diagnostics.AddError("unable to retrieve instance reverse-dns", err.Error())
				return
			}
			model.ReverseDNS = types.StringValue(rdns)
		}

		state.Instances = append(state.Instances, model)
	}

	// `id` is also a filter: when set it has to be returned as configured.
	if state.ID.IsNull() {
		// by sorting instance IDs we can generate the same resource ID regardless of the order in which
		// API returns instances in the list.
		sort.Strings(ids)

		state.ID = types.StringValue(fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(ids, "")))))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Trace(ctx, "datasource read done", map[string]any{"id": state.ID})
}

// filters creates a filter for each filter attribute set in the configuration.
// As with the SDKv2, a zero value (false, 0, "", {}) is no filter.
func (m DataSourceListModel) filters(ctx context.Context) ([]filter.FilterFunc, diag.Diagnostics) {
	var (
		filters []filter.FilterFunc
		diags   diag.Diagnostics
	)

	for name, value := range map[string]types.String{
		AttrCreatedAt:       m.CreatedAt,
		AttrDeployTargetID:  m.DeployTargetID,
		AttrID:              m.ID,
		AttrIPv6Address:     m.IPv6Address,
		AttrManagerID:       m.ManagerID,
		AttrManagerType:     m.ManagerType,
		AttrName:            m.Name,
		AttrPublicIPAddress: m.PublicIPAddress,
		AttrReverseDNS:      m.ReverseDNS,
		AttrSSHKey:          m.SSHKey,
		AttrState:           m.State,
		AttrTemplateID:      m.TemplateID,
		AttrType:            m.Type,
		AttrUserData:        m.UserData,
	} {
		if value.ValueString() == "" {
			continue
		}

		f, err := filter.NewStringFilter(name, value.ValueString())
		if err != nil {
			diags.AddError("failed to create filter", fmt.Sprintf("%s: %s", name, err))
			return nil, diags
		}
		filters = append(filters, f)
	}

	for name, value := range map[string]types.Bool{
		AttrEnableSecureBoot: m.EnableSecureBoot,
		AttrEnableTPM:        m.EnableTPM,
		AttrIPv6:             m.IPv6,
	} {
		if value.ValueBool() {
			filters = append(filters, filter.NewEqualityFilter(name, true))
		}
	}

	if v := m.DiskSize.ValueInt64(); v != 0 {
		filters = append(filters, filter.NewEqualityFilter(AttrDiskSize, v))
	}

	if len(m.Labels.Elements()) > 0 {
		labels := map[string]string{}
		diags.Append(m.Labels.ElementsAs(ctx, &labels, false)...)
		if diags.HasError() {
			return nil, diags
		}

		f, err := filter.NewMapFilter(ctx, AttrLabels, labels)
		if err != nil {
			diags.AddError("failed to create filter", fmt.Sprintf("%s: %s", AttrLabels, err))
			return nil, diags
		}
		filters = append(filters, f)
	}

	return filters, diags
}

// filterData returns the attributes of an instance the filters apply to.
func filterData(model InstanceModel, labels exoscale.Labels) map[string]any {
	return map[string]any{
		AttrCreatedAt:        model.CreatedAt.ValueString(),
		AttrDeployTargetID:   model.DeployTargetID.ValueString(),
		AttrDiskSize:         model.DiskSize.ValueInt64(),
		AttrEnableSecureBoot: model.EnableSecureBoot.ValueBool(),
		AttrEnableTPM:        model.EnableTPM.ValueBool(),
		AttrID:               model.ID.ValueString(),
		AttrIPv6:             model.IPv6.ValueBool(),
		AttrIPv6Address:      model.IPv6Address.ValueString(),
		AttrLabels:           map[string]string(labels),
		AttrManagerID:        model.ManagerID.ValueString(),
		AttrManagerType:      model.ManagerType.ValueString(),
		AttrName:             model.Name.ValueString(),
		AttrPublicIPAddress:  model.PublicIPAddress.ValueString(),
		AttrReverseDNS:       model.ReverseDNS.ValueString(),
		AttrSSHKey:           model.SSHKey.ValueString(),
		AttrState:            model.State.ValueString(),
		AttrTemplateID:       model.TemplateID.ValueString(),
		AttrType:             model.Type.ValueString(),
		AttrUserData:         model.UserData.ValueString(),
	}
}
