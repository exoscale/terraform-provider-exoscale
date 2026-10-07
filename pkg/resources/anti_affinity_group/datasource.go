package anti_affinity_group

import (
	"context"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
)

const markdownDescriptionDataSource = `Fetch Exoscale [Anti-Affinity Groups](https://community.exoscale.com/product/compute/instances/how-to/anti-affinity/) data.

Corresponding resource: [exoscale_anti_affinity_group](../resources/anti_affinity_group.md).`

var _ datasource.DataSourceWithConfigure = (*DataSource)(nil)

type DataSource struct {
	client *exoscale.Client
}

func NewDataSource() datasource.DataSource {
	return &DataSource{}
}

type DataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Instances types.Set    `tfsdk:"instances"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (d *DataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_anti_affinity_group"
}

func (d *DataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Fetch Exoscale Anti-Affinity Groups data.",
		MarkdownDescription: markdownDescriptionDataSource,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The anti-affinity group ID to match (conflicts with 'name').",
				MarkdownDescription: "The anti-affinity group ID to match (conflicts with `name`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("name")),
				},
			},
			"name": schema.StringAttribute{
				Description:         "The group name to match (conflicts with 'id').",
				MarkdownDescription: "The group name to match (conflicts with `id`).",
				Optional:            true,
				Computed:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(path.MatchRoot("id")),
				},
			},
			"instances": schema.SetAttribute{
				Description:         "The list of attached exoscale_compute_instance (IDs).",
				MarkdownDescription: "The list of attached [exoscale_compute_instance](../resources/compute_instance.md) (IDs).",
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

func (d *DataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	d.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (d *DataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := data.Timeouts.Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	id := exoscale.UUID(data.ID.ValueString())
	if data.ID.IsNull() {
		list, err := d.client.ListAntiAffinityGroups(ctx)
		if err != nil {
			resp.Diagnostics.AddError("unable to list anti-affinity groups", err.Error())
			return
		}

		found, err := list.FindAntiAffinityGroup(data.Name.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("unable to find anti-affinity group", err.Error())
			return
		}
		id = found.ID
	}

	// The list doesn't return the group instances.
	aag, err := d.client.GetAntiAffinityGroup(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("unable to get anti-affinity group", err.Error())
		return
	}

	data.ID = types.StringValue(aag.ID.String())
	data.Name = types.StringValue(aag.Name)

	// A group without instances gets an empty set, as with the SDKv2 implementation.
	instanceIDs := make([]string, 0, len(aag.Instances))
	for _, instance := range aag.Instances {
		instanceIDs = append(instanceIDs, instance.ID.String())
	}

	data.Instances, diags = types.SetValueFrom(ctx, types.StringType, instanceIDs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "data source read", map[string]any{
		"id": data.ID.ValueString(),
	})
}
