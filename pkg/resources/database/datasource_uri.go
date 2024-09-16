package database

import (
	"context"
	"fmt"
	"strconv"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const DataSourceURIDescription = `Fetch Exoscale [Database](https://community.exoscale.com/documentation/dbaas/) connection URI data.

This data source returns database conection details of the default (admin) user only.

URI parts are also available individually for convenience.

Corresponding resource: [exoscale_database](../resources/database.md).`

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSourceWithConfigure = &DataSourceURI{}

// DataSourceURI defines the resource implementation.
type DataSourceURI struct {
	client *exoscale.Client
	env    string
}

// NewDataSourceURI creates instance of DataSourceURI.
func NewDataSourceURI() datasource.DataSource {
	return &DataSourceURI{}
}

// DataSourceURIModel defines the data model.
type DataSourceURIModel struct {
	Id types.String `tfsdk:"id"`

	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`

	URI types.String `tfsdk:"uri"`

	// URI components for convenience
	Schema   types.String `tfsdk:"schema"`
	Host     types.String `tfsdk:"host"`
	Port     types.Int64  `tfsdk:"port"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	DbName   types.String `tfsdk:"db_name"`

	Zone types.String `tfsdk:"zone"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// Metadata specifies resource name.
func (d *DataSourceURI) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_database_uri"
}

// Schema defines resource attributes.
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
				MarkdownDescription: "Name of database service to match.",
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
				MarkdownDescription: "Database service connection URI.",
				Computed:            true,
				Sensitive:           true,
			},
			"schema": schema.StringAttribute{
				MarkdownDescription: "Database service connection schema",
				Computed:            true,
			},
			"host": schema.StringAttribute{
				MarkdownDescription: "Database service hostname",
				Computed:            true,
			},
			"port": schema.Int64Attribute{
				MarkdownDescription: "Database service port",
				Computed:            true,
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Admin user username",
				Computed:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Admin user password",
				Computed:            true,
				Sensitive:           true,
			},
			"db_name": schema.StringAttribute{
				MarkdownDescription: "Default database name",
				Computed:            true,
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

// Configure sets up data source dependencies.
func (d *DataSourceURI) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

// Read defines how the data source updates Terraform's state to reflect the retrieved data.
func (d *DataSourceURI) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceURIModel

	// Load Terraform plan into the model.
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

	// Use API endpoint in selected zone.
	client, err := utils.SwitchClientZone(
		ctx,
		d.client,
		exoscale.ZoneName(data.Zone.ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to change exoscale client zone",
			err.Error(),
		)
		return
	}

	// Read remote state.
	data.Id = data.Name

	var uri string
	var params map[string]interface{}

	switch data.Type.ValueString() {
	case "kafka": // kafka has: schema, host & port
		res, err := client.GetDBAASServiceKafka(ctx, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf("Unable to read Database Service kafka: %s", err),
			)
			return
		}

		uri = res.URI
		params = res.URIParams
	case "mysql":
		res, err := client.GetDBAASServiceMysql(ctx, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf("Unable to read Database Service kafka: %s", err),
			)
			return
		}

		data.Schema = types.StringValue("mysql")

		uri = res.URI
		params = res.URIParams
	case "pg":
		res, err := client.GetDBAASServicePG(ctx, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf("Unable to read Database Service kafka: %s", err),
			)
			return
		}

		data.Schema = types.StringValue("postgres")

		uri = res.URI
		params = res.URIParams
	case "redis":
		res, err := client.GetDBAASServiceRedis(ctx, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf("Unable to read Database Service kafka: %s", err),
			)
			return
		}

		data.Schema = types.StringValue("rediss")

		uri = res.URI
		params = res.URIParams
	case "opensearch":
		res, err := client.GetDBAASServiceOpensearch(ctx, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf("Unable to read Database Service kafka: %s", err),
			)
			return
		}

		data.Schema = types.StringValue("https")

		uri = res.URI
		params = res.URIParams
	case "grafana":
		res, err := client.GetDBAASServiceGrafana(ctx, data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf("Unable to read Database Service kafka: %s", err),
			)
			return
		}

		data.Schema = types.StringValue("https")

		uri = res.URI
		params = res.URIParams
	}

	data.URI = types.StringValue(uri)

	if i, ok := params["host"]; ok {
		if s, ok := i.(string); ok {
			data.Host = types.StringValue(s)
		}
	}
	if i, ok := params["port"]; ok {
		if s, ok := i.(string); ok {
			if n, err := strconv.ParseInt(s, 10, 64); err == nil {
				data.Port = types.Int64Value(n)
			}
		}
	}
	if i, ok := params["user"]; ok {
		if s, ok := i.(string); ok {
			data.Username = types.StringValue(s)
		}
	}
	if i, ok := params["password"]; ok {
		if s, ok := i.(string); ok {
			data.Password = types.StringValue(s)
		}
	}
	if i, ok := params["dbname"]; ok {
		if s, ok := i.(string); ok {
			data.DbName = types.StringValue(s)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
