package zones

import (
	"context"

	exoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	ZonesAttrName = "zones"
)

var _ datasource.DataSourceWithConfigure = &ZonesDataSource{}

type ZonesDataSource struct {
	client *exoscale.Client
	env    string
}

type ZonesDataSourceModel struct {
	Zones types.List `tfsdk:"zones"`
}

func (d *ZonesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV2
	d.env = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).Environment
}

func (d *ZonesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_zones"
}

func (d *ZonesDataSource) GetSchema() schema.Schema {
	return schema.Schema{
		Description: "Lists all zones.",
		Attributes: map[string]schema.Attribute{
			ZonesAttrName: schema.ListAttribute{
				Description: `List of zones`,
				ElementType: types.StringType,
				Computed:    true,
			},
		},
	}
}

func (d *ZonesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = d.GetSchema()
}

func (d *ZonesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ZonesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(d.env, config.DefaultZone))
	exoZonesList, err := d.client.ListZones(ctx)
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "")

		return
	}

	attrs := make([]attr.Value, 0, len(exoZonesList))
	for _, zone := range exoZonesList {

		attrs = append(attrs, basetypes.NewStringValue(zone))
	}

	zonesList, listDiags := types.ListValue(types.StringType, attrs)
	resp.Diagnostics.Append(listDiags...)
	data.Zones = zonesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
