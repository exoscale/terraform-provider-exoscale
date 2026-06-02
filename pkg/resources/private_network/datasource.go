package privatenetwork

import (
	"context"
	"fmt"
	"slices"

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

const markdownDescriptionDatasource = `Fetch Exoscale [Private Networks](https://community.exoscale.com/product/networking/private-network/) data.

Corresponding resource: [exoscale_private_network](../resources/private_network.md).`

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
	StartIP     types.String `tfsdk:"start_ip"`
	EndIP       types.String `tfsdk:"end_ip"`
	Netmask     types.String `tfsdk:"netmask"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *DataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_network"
}

func (d *DataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: markdownDescriptionDatasource,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The private network ID to match (conflicts with `name`).",
				MarkdownDescription: "The private network ID to match (conflicts with `name`).",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("name"),
					}...),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The network name to match (conflicts with `id`).",
				MarkdownDescription: "The network name to match (conflicts with `id`).",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("id"),
					}...),
				},
			},
			"zone": schema.StringAttribute{
				Description:         "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"description": schema.StringAttribute{
				Description:         "The private network description.",
				MarkdownDescription: "The private network description.",
				Optional:            true,
			},
			"labels": schema.MapAttribute{
				Description:         "A map of key/value labels.",
				MarkdownDescription: "A map of key/value labels.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"start_ip": schema.StringAttribute{
				Description:         "The first/last IPv4 addresses used by the DHCP service for dynamic leases.",
				MarkdownDescription: "The first/last IPv4 addresses used by the DHCP service for dynamic leases.",
				Computed:            true,
			},
			"end_ip": schema.StringAttribute{
				Description:         "The first/last IPv4 addresses used by the DHCP service for dynamic leases.",
				MarkdownDescription: "The first/last IPv4 addresses used by the DHCP service for dynamic leases.",
				Computed:            true,
			},
			"netmask": schema.StringAttribute{
				Description:         "The network mask defining the IPv4 network allowed for static leases.",
				MarkdownDescription: "The network mask defining the IPv4 network allowed for static leases.",
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

	zone := state.Zone.ValueString()
	if !slices.Contains(config.Zones, zone) {
		resp.Diagnostics.AddError("invalid value", "zone must be a valid exoscale zone")
	}

	timeout, diags := state.Timeouts.Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := utils.SwitchClientZone(
		ctx,
		d.client,
		exoscale.ZoneName(zone),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to change exoscale client zone",
			err.Error(),
		)
		return
	}

	var privateNetwork exoscale.PrivateNetwork
	switch {
	case !(state.Name.IsNull() || state.Name.IsUnknown()): //nolint:staticcheck // in this case De Morgan's law is more complex to read
		privateNetworks, err := client.ListPrivateNetworks(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"API returned an error while fetching private networks",
				err.Error(),
			)
			return
		}
		privateNetwork, err = privateNetworks.FindPrivateNetwork(state.Name.ValueString())
		for i := range privateNetworks.PrivateNetworks {
			fmt.Println("->", privateNetworks.PrivateNetworks[i].Name)
		}
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf("private network with name: %s not found", state.Name.ValueString()),
				err.Error(),
			)
			return
		}

	case !(state.ID.IsNull() && state.ID.IsUnknown()): //nolint:staticcheck // in this case De Morgan's law is more complex to read
		id, err := exoscale.ParseUUID(state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"unable to parse ID",
				err.Error(),
			)
			return
		}
		network, err := client.GetPrivateNetwork(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError(
				"API returned an error while fetching private network",
				err.Error(),
			)
			return
		}
		privateNetwork = *network
	default: // validation must prevents this, exit as a safe guard
		resp.Diagnostics.AddError("missing values", "name and id are missing")
		return
	}

	state = DataSourceModel{
		ID:          types.StringValue(privateNetwork.ID.String()),
		Name:        types.StringValue(privateNetwork.Name),
		Zone:        state.Zone,
		Description: types.StringValue(privateNetwork.Description),
		StartIP:     types.StringValue(privateNetwork.StartIP.String()),
		EndIP:       types.StringValue(privateNetwork.EndIP.String()),
		Netmask:     types.StringValue(privateNetwork.Netmask.String()),
		Timeouts:    state.Timeouts,
	}
	state.Labels = types.MapNull(types.StringType)
	if privateNetwork.Labels != nil {
		labels, dg := types.MapValueFrom(
			ctx,
			types.StringType,
			privateNetwork.Labels,
		)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		state.Labels = labels
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
