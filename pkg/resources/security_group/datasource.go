package security_group

import (
	"context"
	"fmt"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
)

const DataSourceDescription = `Fetch [Exoscale Security Groups](https://community.exoscale.com/product/compute/instances/quick-start/#firewall-rules---security-groups).

Security Groups are firewall rules. Each Security Group may be attached to one or many compute instances.

Corresponding resource: [exoscale_security_group](../resources/security_group.md).`

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSourceWithConfigure = &DataSource{}

// DataSource defines the data source implementation.
type DataSource struct {
	client *exoscale.Client
}

// NewDataSource creates instance of DataSource.
func NewDataSource() datasource.DataSource {
	return &DataSource{}
}

// DataSourceModel defines the data source data model.
type DataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	ExternalSources types.Set    `tfsdk:"external_sources"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// Metadata specifies data source name.
func (d *DataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_security_group"
}

// Schema defines data source attributes.
func (d *DataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: DataSourceDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "security group ID",
				MarkdownDescription: "The ID of the Security Group (required if `name` is not set).",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.Expressions{
						path.MatchRoot("name"),
					}...),
				},
			},
			"name": schema.StringAttribute{
				Description:         "security group name",
				MarkdownDescription: "The name of the Security Group (required if `id` is not set).",
				Optional:            true,
			},
			"description": schema.StringAttribute{
				Description:         "security group description",
				MarkdownDescription: "A free-form text describing the the Security Group.",
				Computed:            true,
			},
			"external_sources": schema.SetAttribute{
				ElementType:         types.StringType,
				Description:         "external network sources",
				MarkdownDescription: `A list of external network sources, in [CIDR](https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing#CIDR_notation) notation.`,
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

// Configure sets up datasource dependencies.
func (d *DataSource) Configure(
	ctx context.Context,
	r datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	if r.ProviderData == nil {
		return
	}

	d.client = r.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

// Read defines how the data source updates Terraform's state to reflect the retrieved data.
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

	var sg *exoscale.SecurityGroup

	if !state.Name.IsNull() {
		sgs, err := d.client.ListSecurityGroups(ctx)
		if err != nil {
			resp.Diagnostics.AddError(
				"API returned error reading security group",
				err.Error(),
			)

			return
		}
		t, err := sgs.FindSecurityGroup(state.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				fmt.Sprintf(
					"security group with name %q not found",
					state.Name.ValueString(),
				),
				err.Error(),
			)

			return
		}
		sg = &t
	} else if !state.ID.IsNull() {
		id, err := exoscale.ParseUUID(state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"unable to parse ID",
				err.Error(),
			)
			return
		}
		sg, err = d.client.GetSecurityGroup(ctx, id)
		if err != nil {
			resp.Diagnostics.AddError(
				"API returned error reading security group",
				err.Error(),
			)

			return
		}
	} else { // validation must prevents this, exit as a safe guard
		return
	}

	state.Name = types.StringValue(sg.Name)
	state.Description = types.StringValue(sg.Description)

	state.ExternalSources = types.SetNull(types.StringType)
	if len(sg.ExternalSources) > 0 {
		setElems := []attr.Value{}
		for _, cidr := range sg.ExternalSources {
			setElems = append(setElems, types.StringValue((cidr)))
		}

		dg := diag.Diagnostics{}
		state.ExternalSources, dg = types.SetValue(types.StringType, setElems)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
