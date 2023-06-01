package zones

import (
	"context"

	exov1 "github.com/exoscale/egoscale"
	"github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"

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
	clientV1 *exov1.Client
}

type ZonesDataSourceModel struct {
	Zones types.List `tfsdk:"zones"`
}

func (d *ZonesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.clientV1 = req.ProviderData.(*config.ExoscaleProviderConfig).ClientV1
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

	exoZonesList, err := d.clientV1.ListWithContext(ctx, &exov1.Zone{})
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "")

		return
	}

	attrs := make([]attr.Value, 0, len(exoZonesList))
	for _, key := range exoZonesList {
		zone := key.(*exov1.Zone)

		attrs = append(attrs, basetypes.NewStringValue(zone.Name))
	}

	zonesList, listDiags := types.ListValue(types.StringType, attrs)
	resp.Diagnostics.Append(listDiags...)
	data.Zones = zonesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
