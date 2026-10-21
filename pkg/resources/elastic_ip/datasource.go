package elasticip

import (
	"context"
	"net"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const markdownDescriptionDatasource = `Fetch Exoscale [Elastic IPs (EIP)](https://community.exoscale.com/product/networking/eip/) data.

Corresponding resource: [exoscale_elastic_ip](../resources/elastic_ip.md).`

var _ datasource.DataSourceWithConfigure = (*DataSource)(nil)

type DataSource struct {
	client *exoscale.Client
}

func NewDataSource() datasource.DataSource {
	return &DataSource{}
}

type DataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Zone          types.String `tfsdk:"zone"`
	AddressFamily types.String `tfsdk:"address_family"`
	CIDR          types.String `tfsdk:"cidr"`
	Description   types.String `tfsdk:"description"`
	Healthcheck   types.Object `tfsdk:"healthcheck"`
	IPAddress     types.String `tfsdk:"ip_address"`
	ReverseDNS    types.String `tfsdk:"reverse_dns"`
	Labels        types.Map    `tfsdk:"labels"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *DataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_elastic_ip"
}

func (d *DataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Fetch Exoscale Elastic IPs (EIP) data.",
		MarkdownDescription: markdownDescriptionDatasource,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The Elastic IP (EIP) ID to match (conflicts with 'ip_address' and 'labels').",
				MarkdownDescription: "The Elastic IP (EIP) ID to match (conflicts with `ip_address` and `labels`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("ip_address"),
						path.MatchRoot("labels"),
					),
				},
			},
			"ip_address": schema.StringAttribute{
				Description:         "The EIP IPv4 or IPv6 address to match (conflicts with 'id' and 'labels').",
				MarkdownDescription: "The EIP IPv4 or IPv6 address to match (conflicts with `id` and `labels`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("id"),
						path.MatchRoot("labels"),
					),
				},
			},
			"labels": schema.MapAttribute{
				Description:         "The EIP labels to match (conflicts with 'ip_address' and 'id').",
				MarkdownDescription: "The EIP labels to match (conflicts with `ip_address` and `id`).",
				ElementType:         types.StringType,
				Optional:            true,
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
			"address_family": schema.StringAttribute{
				Description:         "The Elastic IP (EIP) address family ('inet4' or 'inet6').",
				MarkdownDescription: "The Elastic IP (EIP) address family (`inet4` or `inet6`).",
				Computed:            true,
			},
			"cidr": schema.StringAttribute{
				Description:         "The Elastic IP (EIP) CIDR.",
				MarkdownDescription: "The Elastic IP (EIP) CIDR.",
				Computed:            true,
			},
			"description": schema.StringAttribute{
				Description:         "The Elastic IP (EIP) description.",
				MarkdownDescription: "The Elastic IP (EIP) description.",
				Computed:            true,
			},
			"healthcheck": schema.SingleNestedAttribute{
				Description:         "The *managed* EIP healthcheck configuration.",
				MarkdownDescription: "The *managed* EIP healthcheck configuration.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"interval": schema.Int64Attribute{
						Description:         "The healthcheck interval in seconds.",
						MarkdownDescription: "The healthcheck interval in seconds.",
						Computed:            true,
					},
					"mode": schema.StringAttribute{
						Description:         "The healthcheck mode.",
						MarkdownDescription: "The healthcheck mode.",
						Computed:            true,
					},
					"port": schema.Int64Attribute{
						Description:         "The healthcheck target port.",
						MarkdownDescription: "The healthcheck target port.",
						Computed:            true,
					},
					"timeout": schema.Int64Attribute{
						Description:         "The time in seconds before considering a healthcheck probing failed.",
						MarkdownDescription: "The time in seconds before considering a healthcheck probing failed.",
						Computed:            true,
					},
					"strikes_fail": schema.Int64Attribute{
						Description:         "The number of failed healthcheck attempts before considering the target unhealthy.",
						MarkdownDescription: "The number of failed healthcheck attempts before considering the target unhealthy.",
						Computed:            true,
					},
					"strikes_ok": schema.Int64Attribute{
						Description:         "The number of successful healthcheck attempts before considering the target healthy.",
						MarkdownDescription: "The number of successful healthcheck attempts before considering the target healthy.",
						Computed:            true,
					},
					"tls_skip_verify": schema.BoolAttribute{
						Description:         "Disable TLS certificate verification for healthcheck in 'https' mode.",
						MarkdownDescription: "Disable TLS certificate verification for healthcheck in `https` mode.",
						Computed:            true,
					},
					"tls_sni": schema.StringAttribute{
						Description:         "The healthcheck server name to present with SNI in 'https' mode.",
						MarkdownDescription: "The healthcheck server name to present with SNI in `https` mode.",
						Computed:            true,
					},
					"uri": schema.StringAttribute{
						Description:         "The healthcheck URI.",
						MarkdownDescription: "The healthcheck URI.",
						Computed:            true,
					},
				},
			},
			"reverse_dns": schema.StringAttribute{
				Description:         "Domain name for reverse DNS record.",
				MarkdownDescription: "Domain name for reverse DNS record.",
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
	var state DataSourceModel

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

	client, err := utils.SwitchClientZone(ctx, d.client, exoscale.ZoneName(state.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	var elasticIP *exoscale.ElasticIP
	switch {
	case !state.ID.IsNull():
		id, err := exoscale.ParseUUID(state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("unable to parse ID", err.Error())
			return
		}

		elasticIP, err = client.GetElasticIP(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("API returned an error while fetching Elastic IP", err.Error())
			return
		}

	case !state.IPAddress.IsNull():
		ip := net.ParseIP(state.IPAddress.ValueString())
		elasticIP, diags = findElasticIP(ctx, client, func(eip *exoscale.ElasticIP) bool {
			return ip != nil && ip.Equal(net.ParseIP(eip.IP))
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

	case !state.Labels.IsNull():
		labels := map[string]string{}
		resp.Diagnostics.Append(state.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		// As with the SDKv2, the first Elastic IP holding all the labels wins.
		elasticIP, diags = findElasticIP(ctx, client, func(eip *exoscale.ElasticIP) bool {
			for k, v := range labels {
				if value, ok := eip.Labels[k]; !ok || value != v {
					return false
				}
			}
			return true
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

	default: // validation must prevent this, exit as a safe guard
		resp.Diagnostics.AddError("missing values", "one of id, ip_address or labels must be specified")
		return
	}

	rdns, err := reverseDNS(ctx, client, elasticIP.ID)
	if err != nil {
		resp.Diagnostics.AddError("unable to retrieve Elastic IP reverse DNS", err.Error())
		return
	}

	// Filters are kept as configured: Terraform rejects a data source result
	// that differs from a configured value.
	if state.ID.IsNull() {
		state.ID = types.StringValue(elasticIP.ID.String())
	}
	if state.IPAddress.IsNull() {
		state.IPAddress = types.StringValue(elasticIP.IP)
	}
	if state.Labels.IsNull() {
		labels := elasticIP.Labels
		if labels == nil {
			labels = exoscale.Labels{}
		}
		state.Labels, diags = types.MapValueFrom(ctx, types.StringType, labels)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	state.AddressFamily = types.StringValue(string(elasticIP.Addressfamily))
	state.CIDR = types.StringValue(elasticIP.Cidr)
	state.Description = types.StringValue(elasticIP.Description)
	state.ReverseDNS = types.StringValue(rdns)

	state.Healthcheck, diags = healthcheckObject(ctx, elasticIP.Healthcheck)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Trace(ctx, "data source read done", map[string]any{"id": state.ID})
}

// findElasticIP returns the first Elastic IP of the zone matching the filter.
func findElasticIP(
	ctx context.Context,
	client *exoscale.Client,
	match func(*exoscale.ElasticIP) bool,
) (*exoscale.ElasticIP, diag.Diagnostics) {
	var diags diag.Diagnostics

	elasticIPs, err := client.ListElasticIPS(ctx)
	if err != nil {
		diags.AddError("API returned an error while listing Elastic IPs", err.Error())
		return nil, diags
	}

	for i := range elasticIPs.ElasticIPS {
		if match(&elasticIPs.ElasticIPS[i]) {
			return &elasticIPs.ElasticIPS[i], diags
		}
	}

	diags.AddError("unable to find matching Elastic IP", "no Elastic IP matches the given filter")
	return nil, diags
}
