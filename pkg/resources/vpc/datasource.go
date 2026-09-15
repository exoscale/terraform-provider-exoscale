package vpc

import (
	"context"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const markdownDescriptionDatasource = `Fetch Exoscale [VPC](https://community.exoscale.com/product/networking/vpc/) data.

Corresponding resource: [exoscale_vpc](../resources/vpc.md).`

var _ datasource.DataSourceWithConfigure = (*DataSource)(nil)

type DataSource struct {
	client *exoscale.Client
}

func NewDataSource() datasource.DataSource {
	return &DataSource{}
}

type DataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Zone        types.String `tfsdk:"zone"`
	Description types.String `tfsdk:"description"`
	Labels      types.Map    `tfsdk:"labels"`
	Default     types.Bool   `tfsdk:"default"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *DataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc"
}

func (d *DataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: markdownDescriptionDatasource,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The VPC ID to match (conflicts with 'name').",
				MarkdownDescription: "The VPC ID to match (conflicts with `name`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("name"),
					}...),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The VPC name to match (conflicts with 'id').",
				MarkdownDescription: "The VPC name to match (conflicts with `id`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("id"),
					}...),
				},
			},
			"zone": schema.StringAttribute{
				Description:         "The Exoscale zone name.",
				MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"description": schema.StringAttribute{
				Description:         "The VPC description.",
				MarkdownDescription: "The VPC description.",
				Computed:            true,
			},
			"labels": schema.MapAttribute{
				Description:         "A map of key/value labels.",
				MarkdownDescription: "A map of key/value labels.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"default": schema.BoolAttribute{
				Description:         "Whether this is the organization's default VPC for the zone.",
				MarkdownDescription: "Whether this is the organization's default VPC for the zone.",
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

	client, err := utils.SwitchClientZone(ctx, d.client, exoscale.ZoneName(state.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	var vpc exoscale.Vpc
	switch {
	case !state.Name.IsNull():
		vpcs, err := client.ListVpcs(ctx)
		if err != nil {
			resp.Diagnostics.AddError("API returned an error while fetching VPCs", err.Error())
			return
		}
		entry, err := vpcs.FindListVpcEntry(state.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("VPC with name: "+state.Name.ValueString()+" not found", err.Error())
			return
		}
		got, err := client.GetVpc(ctx, entry.ID)
		if err != nil {
			resp.Diagnostics.AddError("API returned an error while fetching VPC", err.Error())
			return
		}
		vpc = *got

	case !state.ID.IsNull():
		id, err := exoscale.ParseUUID(state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("unable to parse ID", err.Error())
			return
		}
		got, err := client.GetVpc(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("API returned an error while fetching VPC", err.Error())
			return
		}
		vpc = *got

	default: // validation must prevent this, exit as a safe guard
		resp.Diagnostics.AddError("missing values", "name and id are missing")
		return
	}

	state = DataSourceModel{
		ID:          types.StringValue(vpc.ID.String()),
		Name:        types.StringValue(vpc.Name),
		Zone:        state.Zone,
		Description: types.StringValue(vpc.Description),
		Default:     types.BoolValue(vpc.Default != nil && *vpc.Default),
		Timeouts:    state.Timeouts,
	}
	state.Labels = types.MapNull(types.StringType)
	if vpc.Labels != nil {
		labels, dg := types.MapValueFrom(ctx, types.StringType, vpc.Labels)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}
		state.Labels = labels
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
