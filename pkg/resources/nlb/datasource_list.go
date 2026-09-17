package nlb

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
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const (
	NLBServiceListAttrNLBID          = "nlb_id"
	NLBServiceListAttrNLBName        = "nlb_name"
	NLBServiceListAttrNLBServiceList = "services"
	NLBServiceListAttrZone           = "zone"

	NLBServiceAttrDescription    = "description"
	NLBServiceAttrID             = "id"
	NLBServiceAttrHealthcheck    = "healthcheck"
	NLBServiceAttrInstancePoolID = "instance_pool_id"
	NLBServiceAttrName           = "name"
	NLBServiceAttrPort           = "port"
	NLBServiceAttrProtocol       = "protocol"
	NLBServiceAttrStrategy       = "strategy"
	NLBServiceAttrState          = "state"
	NLBServiceAttrTargetPort     = "target_port"

	NLBServiceHealthcheckAttrInterval = "interval"
	NLBServiceHealthcheckAttrMode     = "mode"
	NLBServiceHealthcheckAttrPort     = "port"
	NLBServiceHealthcheckAttrRetries  = "retries"
	NLBServiceHealthcheckAttrTimeout  = "timeout"
	NLBServiceHealthcheckAttrTLSSNI   = "tls_sni"
	NLBServiceHealthcheckAttrURI      = "uri"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ datasource.DataSource              = &DataSourceServiceList{}
	_ datasource.DataSourceWithConfigure = &DataSourceServiceList{}
)

// DataSourceServiceList is the exoscale_nlb_service_list data source implementation.
type DataSourceServiceList struct {
	client *exoscale.Client
}

// NewDataSourceServiceList creates an instance of DataSourceServiceList.
func NewDataSourceServiceList() datasource.DataSource {
	return &DataSourceServiceList{}
}

// DataSourceServiceListModel defines the exoscale_nlb_service_list data model.
type DataSourceServiceListModel struct {
	ID             types.String `tfsdk:"id"`
	NLBID          types.String `tfsdk:"nlb_id"`
	NLBName        types.String `tfsdk:"nlb_name"`
	NLBServiceList []Service    `tfsdk:"services"`
	Zone           types.String `tfsdk:"zone"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// Service is a single entry of the `services` list.
type Service struct {
	Description    types.String     `tfsdk:"description"`
	Healthcheck    HealthcheckModel `tfsdk:"healthcheck"`
	ID             types.String     `tfsdk:"id"`
	InstancePoolID types.String     `tfsdk:"instance_pool_id"`
	Name           types.String     `tfsdk:"name"`
	Port           types.Int64      `tfsdk:"port"`
	Protocol       types.String     `tfsdk:"protocol"`
	State          types.String     `tfsdk:"state"`
	Strategy       types.String     `tfsdk:"strategy"`
	TargetPort     types.Int64      `tfsdk:"target_port"`
}

func (d *DataSourceServiceList) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (d *DataSourceServiceList) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nlb_service_list"
}

func (d *DataSourceServiceList) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Fetch Exoscale [Network Load Balancers (NLB)](https://community.exoscale.com/product/networking/nlb/) Services.

Corresponding resource: [exoscale_nlb](../resources/nlb.md).`,
		Attributes: map[string]schema.Attribute{
			NLBServiceAttrID: schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
			},
			NLBServiceListAttrZone: schema.StringAttribute{
				MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			NLBServiceListAttrNLBID: schema.StringAttribute{
				MarkdownDescription: "The NLB ID to match (conflicts with `nlb_name`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot(NLBServiceListAttrNLBName),
					}...),
				},
			},
			NLBServiceListAttrNLBName: schema.StringAttribute{
				MarkdownDescription: "The NLB name to match (conflicts with `nlb_id`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot(NLBServiceListAttrNLBID),
					}...),
				},
			},
			NLBServiceListAttrNLBServiceList: schema.ListNestedAttribute{
				MarkdownDescription: "The list of [exoscale_nlb_service](./nlb_service_list.md).",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						NLBServiceAttrDescription: schema.StringAttribute{
							MarkdownDescription: "NLB service description.",
							Computed:            true,
						},
						NLBServiceAttrID: schema.StringAttribute{
							MarkdownDescription: "NLB service ID.",
							Computed:            true,
						},
						NLBServiceAttrHealthcheck: schema.ObjectAttribute{
							Computed:       true,
							AttributeTypes: HealthcheckModel{}.Types(),
						},
						NLBServiceAttrInstancePoolID: schema.StringAttribute{
							MarkdownDescription: "The [exoscale_instance_pool](./instance_pool.md) (ID) to forward traffic to.",
							Computed:            true,
						},
						NLBServiceAttrName: schema.StringAttribute{
							MarkdownDescription: "NLB Service name.",
							Computed:            true,
						},
						NLBServiceAttrPort: schema.Int64Attribute{
							MarkdownDescription: "Port exposed on the NLB's public IP.",
							Computed:            true,
						},
						NLBServiceAttrProtocol: schema.StringAttribute{
							MarkdownDescription: "Network traffic protocol.",
							Computed:            true,
						},
						NLBServiceAttrState: schema.StringAttribute{
							MarkdownDescription: "NLB Service State.",
							Computed:            true,
						},
						NLBServiceAttrStrategy: schema.StringAttribute{
							MarkdownDescription: "The strategy (`round-robin`|`source-hash`).",
							Computed:            true,
						},
						NLBServiceAttrTargetPort: schema.Int64Attribute{
							MarkdownDescription: "Port on which the network traffic will be forwarded to on the receiving instance.",
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

func (d *DataSourceServiceList) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceServiceListModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	t, diags := data.Timeouts.Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	client, err := utils.SwitchClientZone(ctx, d.client, exoscale.ZoneName(data.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	var nlb *exoscale.LoadBalancer

	switch {
	case !data.NLBID.IsNull():
		id, err := exoscale.ParseUUID(data.NLBID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("unable to parse NLB ID", err.Error())
			return
		}
		nlb, err = client.GetLoadBalancer(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("unable to find Network Load Balancer", err.Error())
			return
		}

	case !data.NLBName.IsNull():
		nlbs, err := client.ListLoadBalancers(ctx)
		if err != nil {
			resp.Diagnostics.AddError("API returned an error while fetching NLBs", err.Error())
			return
		}
		found, err := nlbs.FindLoadBalancer(data.NLBName.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("unable to find Network Load Balancer", err.Error())
			return
		}
		// The list endpoint does not return the services, fetch the NLB itself.
		nlb, err = client.GetLoadBalancer(ctx, found.ID)
		if err != nil {
			resp.Diagnostics.AddError("unable to find Network Load Balancer", err.Error())
			return
		}

	default: // validation must prevent this, exit as a safe guard
		resp.Diagnostics.AddError("missing values", "either nlb_name or nlb_id must be specified")
		return
	}

	data.NLBID = types.StringValue(nlb.ID.String())
	data.NLBName = types.StringValue(nlb.Name)

	// Use NLB ID as data source ID since it is unique.
	data.ID = data.NLBID

	for i := range nlb.Services {
		service := nlb.Services[i]

		serviceState := Service{
			Description: optionalString(service.Description),
			Healthcheck: healthcheckFromAPI(service.Healthcheck),
			ID:          types.StringValue(service.ID.String()),
			Name:        types.StringValue(service.Name),
			Port:        types.Int64Value(service.Port),
			Protocol:    types.StringValue(string(service.Protocol)),
			State:       types.StringValue(string(service.State)),
			Strategy:    types.StringValue(string(service.Strategy)),
			TargetPort:  types.Int64Value(service.TargetPort),

			InstancePoolID: types.StringNull(),
		}
		if service.InstancePool != nil {
			serviceState.InstancePoolID = types.StringValue(service.InstancePool.ID.String())
		}

		data.NLBServiceList = append(data.NLBServiceList, serviceState)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	tflog.Trace(ctx, "datasource read done", map[string]any{"id": data.ID})
}
