package nlb_service

import (
	"context"
	"time"

	exoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
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
	_ datasource.DataSource              = &NLBServiceListDataSource{}
	_ datasource.DataSourceWithConfigure = &NLBServiceListDataSource{}
)

// NLBServiceListDataSource is the data source implementation.
type NLBServiceListDataSource struct {
	client *exoscale.Client
	env    string
}

type DataSourceModel struct {
	ID             types.String `tfsdk:"id"`
	NLBID          types.String `tfsdk:"nlb_id"`
	NLBName        types.String `tfsdk:"nlb_name"`
	NLBServiceList []Service    `tfsdk:"services"`
	Zone           types.String `tfsdk:"zone"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type Service struct {
	Description    types.String `tfsdk:"description"`
	Healthcheck    Healthcheck  `tfsdk:"healthcheck"`
	ID             types.String `tfsdk:"id"`
	InstancePoolID types.String `tfsdk:"instance_pool_id"`
	Name           types.String `tfsdk:"name"`
	Port           types.Int64  `tfsdk:"port"`
	Protocol       types.String `tfsdk:"protocol"`
	State          types.String `tfsdk:"state"`
	Strategy       types.String `tfsdk:"strategy"`
	TargetPort     types.Int64  `tfsdk:"target_port"`
}

type Healthcheck struct {
	Interval types.Int64  `tfsdk:"interval"`
	Mode     types.String `tfsdk:"mode"`
	Port     types.Int64  `tfsdk:"port"`
	Retries  types.Int64  `tfsdk:"retries"`
	TLSSNI   types.String `tfsdk:"tls_sni"`
	Timeout  types.Int64  `tfsdk:"timeout"`
	URI      types.String `tfsdk:"uri"`
}

func (d *NLBServiceListDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV2
	d.env = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).Environment
}

func (d *NLBServiceListDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nlb_service_list"
}

func (d *NLBServiceListDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Fetch Exoscale [Network Load Balancers (NLB)](https://community.exoscale.com/documentation/compute/network-load-balancer/) Services.

Corresponding resource: [exoscale_nlb](../resources/nlb.md).`,
		Attributes: map[string]schema.Attribute{
			NLBServiceAttrID: schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
			},
			NLBServiceListAttrZone: schema.StringAttribute{
				Required: true,
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			NLBServiceListAttrNLBID: schema.StringAttribute{
				Optional: true,
			},
			NLBServiceListAttrNLBName: schema.StringAttribute{
				Optional: true,
			},
			NLBServiceListAttrNLBServiceList: schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						NLBServiceAttrDescription: schema.StringAttribute{
							Computed: true,
						},
						NLBServiceAttrID: schema.StringAttribute{
							Computed: true,
						},
						NLBServiceAttrHealthcheck: schema.ObjectAttribute{
							Computed: true,
							AttributeTypes: map[string]attr.Type{
								NLBServiceHealthcheckAttrInterval: types.Int64Type,
								NLBServiceHealthcheckAttrMode:     types.StringType,
								NLBServiceHealthcheckAttrPort:     types.Int64Type,
								NLBServiceHealthcheckAttrRetries:  types.Int64Type,
								NLBServiceHealthcheckAttrTimeout:  types.Int64Type,
								NLBServiceHealthcheckAttrTLSSNI:   types.StringType,
								NLBServiceHealthcheckAttrURI:      types.StringType,
							},
						},
						NLBServiceAttrInstancePoolID: schema.StringAttribute{
							Computed: true,
						},
						NLBServiceAttrName: schema.StringAttribute{
							Computed: true,
						},
						NLBServiceAttrPort: schema.Int64Attribute{
							Computed: true,
						},
						NLBServiceAttrProtocol: schema.StringAttribute{
							Computed: true,
						},
						NLBServiceAttrState: schema.StringAttribute{
							Computed: true,
						},
						NLBServiceAttrStrategy: schema.StringAttribute{
							Computed: true,
						},
						NLBServiceAttrTargetPort: schema.Int64Attribute{
							Computed: true,
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

func (d *NLBServiceListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceModel

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

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(d.env, data.Zone.ValueString()))

	var x string
	switch {
	case !data.NLBID.IsNull():
		x = data.NLBID.ValueString()
	case !data.NLBName.IsNull():
		x = data.NLBName.ValueString()
	default:
		resp.Diagnostics.AddError(
			"Either nlb_name or nlb_id must be specified",
			"",
		)
		return
	}

	nlb, err := d.client.FindNetworkLoadBalancer(ctx, data.Zone.ValueString(), x)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to find Network Load Balancer",
			err.Error(),
		)
		return
	}

	if nlb.ID != nil && data.NLBID.IsNull() {
		data.NLBID = types.StringValue(*nlb.ID)
	}
	if nlb.Name != nil && data.NLBName.IsNull() {
		data.NLBName = types.StringValue(*nlb.Name)
	}

	// Use NLB ID as data source ID since it is unique.
	data.ID = data.NLBID

	for _, service := range nlb.Services {
		var serviceState Service

		if service.Description != nil {
			serviceState.Description = types.StringValue(*service.Description)
		}
		if service.ID != nil {
			serviceState.ID = types.StringValue(*service.ID)
		}
		if service.InstancePoolID != nil {
			serviceState.InstancePoolID = types.StringValue(*service.InstancePoolID)
		}
		if service.Name != nil {
			serviceState.Name = types.StringValue(*service.Name)
		}
		if service.Port != nil {
			serviceState.Port = types.Int64Value(int64(*service.Port))
		}
		if service.Protocol != nil {
			serviceState.Protocol = types.StringValue(*service.Protocol)
		}
		if service.State != nil {
			serviceState.State = types.StringValue(*service.State)
		}
		if service.Strategy != nil {
			serviceState.Strategy = types.StringValue(*service.Strategy)
		}
		if service.TargetPort != nil {
			serviceState.TargetPort = types.Int64Value(int64(*service.TargetPort))
		}

		var healtcheckState Healthcheck

		if service.Healthcheck.Interval != nil {
			healtcheckState.Interval = types.Int64Value(int64(*service.Healthcheck.Interval / time.Second))
		}
		if service.Healthcheck.Mode != nil {
			healtcheckState.Mode = types.StringValue(*service.Healthcheck.Mode)
		}
		if service.Healthcheck.Port != nil {
			healtcheckState.Port = types.Int64Value(int64(*service.Healthcheck.Port))
		}
		if service.Healthcheck.Retries != nil {
			healtcheckState.Retries = types.Int64Value(int64(*service.Healthcheck.Retries))
		}
		if service.Healthcheck.TLSSNI != nil {
			healtcheckState.TLSSNI = types.StringValue(*service.Healthcheck.TLSSNI)
		}
		if service.Healthcheck.Timeout != nil {
			healtcheckState.Timeout = types.Int64Value(int64(*service.Healthcheck.Timeout / time.Second))
		}
		if service.Healthcheck.URI != nil {
			healtcheckState.URI = types.StringValue(*service.Healthcheck.URI)
		}

		serviceState.Healthcheck = healtcheckState

		data.NLBServiceList = append(data.NLBServiceList, serviceState)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
