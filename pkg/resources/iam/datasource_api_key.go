package iam

import (
	"context"

	exoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
)

const DataSourceAPIKeyDescription = `Fetch Exoscale [IAM](https://community.exoscale.com/documentation/iam/) API Key.

Corresponding resource: [exoscale_iam_role](../resources/iam_api_key.md).`

var _ datasource.DataSourceWithConfigure = &DataSourceAPIKey{}

func NewDataSourceAPIKey() datasource.DataSource {
	return &DataSourceAPIKey{}
}

type DataSourceAPIKey struct {
	client *exoscale.Client
	env    string
}

type DataSourceAPIKeyModel struct {
	ID   types.String `tfsdk:"id"`
	Key  types.String `tfsdk:"key"`
	Name types.String `tfsdk:"name"`

	RoleID types.String `tfsdk:"role_id"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *DataSourceAPIKey) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_iam_api_key"
}

func (d *DataSourceAPIKey) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: DataSourceAPIKeyDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "The IAM API Key to match.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "IAM API Key name.",
				Computed:            true,
			},
			"role_id": schema.StringAttribute{
				MarkdownDescription: "IAM API Key role ID.",
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

func (d *DataSourceAPIKey) Configure(
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

func (d *DataSourceAPIKey) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceAPIKeyModel

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

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(d.env, config.DefaultZone))

	apiKey, err := d.client.GetAPIKey(
		ctx,
		config.DefaultZone,
		data.Key.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to get IAM Role",
			err.Error(),
		)
		return
	}

	data.ID = types.StringPointerValue(apiKey.Key)
	data.Name = types.StringPointerValue(apiKey.Name)
	data.RoleID = types.StringPointerValue(apiKey.RoleID)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
