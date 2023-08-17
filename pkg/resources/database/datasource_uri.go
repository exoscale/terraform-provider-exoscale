package database

import (
	"context"
	"fmt"
	"net/http"

	exoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/egoscale/v2/oapi"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
)

const DataSourceURIDescription = `Fetch Exoscale [Database](https://community.exoscale.com/documentation/dbaas/) URI data.

Corresponding resource: [exoscale_database](../resources/database.md).`

var _ datasource.DataSourceWithConfigure = &DataSourceURI{}

func NewDataSourceURI() datasource.DataSource {
	return &DataSourceURI{}
}

type DataSourceURI struct {
	client *exoscale.Client
	env    string
}

type DataSourceURIModel struct {
	Id   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
	URI  types.String `tfsdk:"uri"`
	Zone types.String `tfsdk:"zone"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *DataSourceURI) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_database_uri"
}

func (d *DataSourceURI) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: DataSourceURIDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The database name to match.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The type of the database service (`kafka`, `mysql`, `opensearch`, `pg`, `redis`).",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(ServicesList...),
				},
			},
			"uri": schema.StringAttribute{
				MarkdownDescription: "The database service connection URI.",
				Computed:            true,
				Sensitive:           true,
			},
			"zone": schema.StringAttribute{
				MarkdownDescription: "The Exoscale Zone name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
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

func (d *DataSourceURI) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV2
	d.env = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).Environment
}

func (d *DataSourceURI) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceURIModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout
	t, diags := data.Timeouts.Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(d.env, data.Zone.ValueString()))
	data.Id = data.Name

	switch data.Type.ValueString() {
	case "kafka":
		res, err := d.client.GetDbaasServiceKafkaWithResponse(ctx, oapi.DbaasServiceName(data.Name.ValueString()))
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Database Service kafka: %s", err))
			return
		}
		if res.StatusCode() != http.StatusOK {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Database Service kafka, unexpected status: %s", res.Status()))
			return
		}
		data.URI = types.StringPointerValue(res.JSON200.Uri)
	case "mysql":
		res, err := d.client.GetDbaasServiceMysqlWithResponse(ctx, oapi.DbaasServiceName(data.Name.ValueString()))
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Database Service mysql: %s", err))
			return
		}
		if res.StatusCode() != http.StatusOK {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Database Service mysql, unexpected status: %s", res.Status()))
			return
		}
		data.URI = types.StringPointerValue(res.JSON200.Uri)
	case "pg":
		res, err := d.client.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(data.Name.ValueString()))
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Database Service pg: %s", err))
			return
		}
		if res.StatusCode() != http.StatusOK {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Database Service pg, unexpected status: %s", res.Status()))
			return
		}
		data.URI = types.StringPointerValue(res.JSON200.Uri)
	case "redis":
		res, err := d.client.GetDbaasServiceRedisWithResponse(ctx, oapi.DbaasServiceName(data.Name.ValueString()))
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Database Service redis: %s", err))
			return
		}
		if res.StatusCode() != http.StatusOK {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Database Service redis, unexpected status: %s", res.Status()))
			return
		}
		data.URI = types.StringPointerValue(res.JSON200.Uri)
	case "opensearch":
		res, err := d.client.GetDbaasServiceOpensearchWithResponse(ctx, oapi.DbaasServiceName(data.Name.ValueString()))
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Database Service opensearch: %s", err))
			return
		}
		if res.StatusCode() != http.StatusOK {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Database Service opensearch, unexpected status: %s", res.Status()))
			return
		}
		data.URI = types.StringPointerValue(res.JSON200.Uri)
	case "grafana":
		res, err := d.client.GetDbaasServiceGrafanaWithResponse(ctx, oapi.DbaasServiceName(data.Name.ValueString()))
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Database Service grafana: %s", err))
			return
		}
		if res.StatusCode() != http.StatusOK {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Database Service grafana, unexpected status: %s", res.Status()))
			return
		}
		data.URI = types.StringPointerValue(res.JSON200.Uri)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
