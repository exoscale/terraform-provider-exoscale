package nlb

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
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const markdownDescriptionDatasource = `Fetch Exoscale [Network Load Balancers (NLB)](https://community.exoscale.com/product/networking/nlb/) data.

Corresponding resource: [exoscale_nlb](../resources/nlb.md).`

var _ datasource.DataSource = (*DataSource)(nil)
var _ datasource.DataSourceWithConfigure = (*DataSource)(nil)

type DataSource struct {
	client *exoscale.Client
}

// NewDataSource creates an instance of DataSource.
func NewDataSource() datasource.DataSource {
	return &DataSource{}
}

// DataSourceModel defines the exoscale_nlb data source data model.
type DataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Labels      types.Map    `tfsdk:"labels"`
	Zone        types.String `tfsdk:"zone"`
	CreatedAt   types.String `tfsdk:"created_at"`
	IPAddress   types.String `tfsdk:"ip_address"`
	State       types.String `tfsdk:"state"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *DataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_nlb"
}

func (d *DataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Fetch Exoscale Network Load Balancers (NLB) data.",
		MarkdownDescription: markdownDescriptionDatasource,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The Network Load Balancers (NLB) ID to match (conflicts with 'name').",
				MarkdownDescription: "The Network Load Balancers (NLB) ID to match (conflicts with `name`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("name"),
					}...),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The NLB name to match (conflicts with 'id').",
				MarkdownDescription: "The NLB name to match (conflicts with `id`).",
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
				Description:         "The Network Load Balancers (NLB) description.",
				MarkdownDescription: "The Network Load Balancers (NLB) description.",
				Computed:            true,
			},
			"labels": schema.MapAttribute{
				Description:         "A map of key/value labels.",
				MarkdownDescription: "A map of key/value labels.",
				ElementType:         types.StringType,
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				Description:         "The NLB creation date.",
				MarkdownDescription: "The NLB creation date.",
				Computed:            true,
			},
			"ip_address": schema.StringAttribute{
				Description:         "The NLB public IPv4 address.",
				MarkdownDescription: "The NLB public IPv4 address.",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				Description:         "The current NLB state.",
				MarkdownDescription: "The current NLB state.",
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

	var nlb exoscale.LoadBalancer

	switch {
	case !state.Name.IsNull():
		nlbs, err := client.ListLoadBalancers(ctx)
		if err != nil {
			resp.Diagnostics.AddError("API returned an error while fetching NLBs", err.Error())
			return
		}
		found, err := nlbs.FindLoadBalancer(state.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("NLB with name: "+state.Name.ValueString()+" not found", err.Error())
			return
		}
		// The list endpoint does not return every attribute, fetch the NLB itself.
		got, err := client.GetLoadBalancer(ctx, found.ID)
		if err != nil {
			resp.Diagnostics.AddError("API returned an error while fetching NLB", err.Error())
			return
		}
		nlb = *got

	case !state.ID.IsNull():
		id, err := exoscale.ParseUUID(state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("unable to parse ID", err.Error())
			return
		}
		got, err := client.GetLoadBalancer(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError("API returned an error while fetching NLB", err.Error())
			return
		}
		nlb = *got

	default: // validation must prevent this, exit as a safe guard
		resp.Diagnostics.AddError("missing values", "name and id are missing")
		return
	}

	state.ID = types.StringValue(nlb.ID.String())
	state.Name = types.StringValue(nlb.Name)
	state.Description = types.StringValue(nlb.Description)
	state.CreatedAt = types.StringValue(nlb.CreatedAT.String())
	state.State = types.StringValue(string(nlb.State))

	state.IPAddress = types.StringNull()
	if len(nlb.IP) > 0 {
		state.IPAddress = types.StringValue(nlb.IP.String())
	}

	state.Labels = types.MapNull(types.StringType)
	if len(nlb.Labels) > 0 {
		labels, dg := types.MapValueFrom(ctx, types.StringType, nlb.Labels)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}
		state.Labels = labels
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	tflog.Trace(ctx, "datasource read done", map[string]any{"id": state.ID})
}
