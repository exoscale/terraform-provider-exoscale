package domain

import (
	"context"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const DataSourceDescription = `Fetch Exoscale [DNS](https://community.exoscale.com/product/networking/dns/) Domains data.

Corresponding resource: [exoscale_domain](../resources/domain.md).`

var _ datasource.DataSourceWithConfigure = &DataSource{}

type DataSource struct {
	client *exoscale.Client
}

func NewDataSource() datasource.DataSource {
	return &DataSource{}
}

type DataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *DataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (d *DataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: DataSourceDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Description:         "The ID of this resource.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The DNS domain name to match.",
				Description:         "The DNS domain name to match.",
				Required:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Read: true,
			}),
		},
	}
}

func (d *DataSource) Configure(ctx context.Context, r datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if r.ProviderData == nil {
		return
	}

	d.client = r.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
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

	client, err := utils.SwitchClientZone(ctx, d.client, exoscale.ZoneName(config.DefaultZone))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	// Normalize to unicode so both punycode and unicode inputs match against
	// the unicode names returned by the API.
	domainName := domainNameToUnicode(state.Name.ValueString())

	domains, err := client.ListDNSDomains(ctx)
	if err != nil {
		resp.Diagnostics.AddError("API returned error when listing domains", err.Error())
		return
	}

	domain, err := domains.FindDNSDomain(domainName)
	if err != nil {
		resp.Diagnostics.AddError("API returned error when reading domain", err.Error())
		return
	}

	state.ID = types.StringValue(domain.ID.String())
	state.Name = types.StringValue(domain.UnicodeName)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
