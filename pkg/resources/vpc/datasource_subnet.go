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

const markdownDescriptionSubnetDatasource = `Fetch Exoscale [VPC](https://community.exoscale.com/product/networking/vpc/) Subnet data.

Corresponding resource: [exoscale_vpc_subnet](../resources/vpc_subnet.md).`

var _ datasource.DataSourceWithConfigure = (*DataSourceSubnet)(nil)

type DataSourceSubnet struct {
	client *exoscale.Client
}

func NewDataSourceSubnet() datasource.DataSource {
	return &DataSourceSubnet{}
}

type DataSourceSubnetModel struct {
	ID            types.String `tfsdk:"id"`
	VpcID         types.String `tfsdk:"vpc_id"`
	Name          types.String `tfsdk:"name"`
	Zone          types.String `tfsdk:"zone"`
	Description   types.String `tfsdk:"description"`
	IPv4Block     types.String `tfsdk:"ipv4_block"`
	AddressFamily types.String `tfsdk:"address_family"`
	AddressSpace  types.String `tfsdk:"address_space"`
	Labels        types.Map    `tfsdk:"labels"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *DataSourceSubnet) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vpc_subnet"
}

func (d *DataSourceSubnet) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: markdownDescriptionSubnetDatasource,

		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The Subnet ID to match (conflicts with 'name').",
				MarkdownDescription: "The Subnet ID to match (conflicts with `name`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("name"),
					}...),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The Subnet name to match (conflicts with 'id').",
				MarkdownDescription: "The Subnet name to match (conflicts with `id`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("id"),
					}...),
				},
			},
			"vpc_id": schema.StringAttribute{
				Description:         "The parent VPC ID.",
				MarkdownDescription: "The parent [exoscale_vpc](./vpc.md) ID.",
				Required:            true,
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
				Description:         "The Subnet description.",
				MarkdownDescription: "The Subnet description.",
				Computed:            true,
			},
			"ipv4_block": schema.StringAttribute{
				Description:         "The Subnet IPv4 CIDR.",
				MarkdownDescription: "The Subnet IPv4 CIDR.",
				Computed:            true,
			},
			"address_family": schema.StringAttribute{
				Description:         "The Subnet address family.",
				MarkdownDescription: "The Subnet address family.",
				Computed:            true,
			},
			"address_space": schema.StringAttribute{
				Description:         "The Subnet address space.",
				MarkdownDescription: "The Subnet address space.",
				Computed:            true,
			},
			"labels": schema.MapAttribute{
				Description:         "A map of key/value labels.",
				MarkdownDescription: "A map of key/value labels.",
				ElementType:         types.StringType,
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

func (d *DataSourceSubnet) Configure(ctx context.Context, r datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if r.ProviderData == nil {
		return
	}

	d.client = r.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (d *DataSourceSubnet) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state DataSourceSubnetModel

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

	vpcID, err := exoscale.ParseUUID(state.VpcID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse parent VPC ID", err.Error())
		return
	}

	var subnet exoscale.Subnet
	switch {
	case !state.Name.IsNull(): //nolint:staticcheck // in this case De Morgan's law is more complex to read
		subnets, err := client.ListSubnets(ctx, vpcID)
		if err != nil {
			resp.Diagnostics.AddError("API returned an error while fetching Subnets", err.Error())
			return
		}
		entry, err := subnets.FindListSubnetEntry(state.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Subnet with name: "+state.Name.ValueString()+" not found", err.Error())
			return
		}
		got, err := client.GetSubnet(ctx, vpcID, entry.ID)
		if err != nil {
			resp.Diagnostics.AddError("API returned an error while fetching Subnet", err.Error())
			return
		}
		subnet = *got

	case !state.ID.IsNull(): //nolint:staticcheck // in this case De Morgan's law is more complex to read
		id, err := exoscale.ParseUUID(state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("unable to parse ID", err.Error())
			return
		}
		got, err := client.GetSubnet(ctx, vpcID, id)
		if err != nil {
			resp.Diagnostics.AddError("API returned an error while fetching Subnet", err.Error())
			return
		}
		subnet = *got

	default: // validation must prevent this, exit as a safe guard
		resp.Diagnostics.AddError("missing values", "name and id are missing")
		return
	}

	state = DataSourceSubnetModel{
		ID:            types.StringValue(subnet.ID.String()),
		VpcID:         state.VpcID,
		Name:          types.StringValue(subnet.Name),
		Zone:          state.Zone,
		Description:   types.StringValue(subnet.Description),
		IPv4Block:     types.StringValue(subnet.Ipv4Block),
		AddressFamily: types.StringValue(string(subnet.Addressfamily)),
		AddressSpace:  types.StringValue(string(subnet.AddressSpace)),
		Timeouts:      state.Timeouts,
	}
	state.Labels = types.MapNull(types.StringType)
	if subnet.Labels != nil {
		labels, dg := types.MapValueFrom(ctx, types.StringType, subnet.Labels)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}
		state.Labels = labels
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
