package block_storage

import (
	"context"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/utils"
)

const DataSourceSnapshotDescription = `Fetch [Exoscale Block Storage](https://community.exoscale.com/documentation/block-storage/) Snapshot.

Block Storage offers persistent externally attached volumes for your workloads.

Corresponding resource: [exoscale_block_storage_snapshot](../resources/block_storage_volume_snapshot.md).`

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSourceWithConfigure = &DataSourceSnapshot{}

// DataSourceSnapshot defines the resource implementation.
type DataSourceSnapshot struct {
	client *exoscale.Client
}

// NewDataSourceSnapshot creates instance of DataSourceSnapshot.
func NewDataSourceSnapshot() datasource.DataSource {
	return &DataSourceSnapshot{}
}

// DataSourceSnapshotModel defines the resource data model.
type DataSourceSnapshotModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Size      types.Int64  `tfsdk:"size"`
	CreatedAt types.String `tfsdk:"created_at"`
	Labels    types.Map    `tfsdk:"labels"`
	State     types.String `tfsdk:"state"`
	Volume    types.Object `tfsdk:"volume"`
	Zone      types.String `tfsdk:"zone"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// Metadata specifies resource name.
func (d *DataSourceSnapshot) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_block_storage_volume_snapshot"
}

// Schema defines resource attributes.
func (d *DataSourceSnapshot) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: DataSourceSnapshotDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Snapshot ID to match.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Snapshot name.",
				Computed:            true,
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "Snapshot size in GB.",
				Computed:            true,
			},
			"zone": schema.StringAttribute{
				MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Snapshot creation date.",
				Computed:            true,
			},
			"volume": schema.SingleNestedAttribute{
				MarkdownDescription: "Block Storage Volume.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "Volume ID.",
						Computed:            true,
					},
				},
			},
			"labels": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Labels.",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "Snapshot state.",
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
func (d *DataSourceSnapshot) Configure(
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
func (d *DataSourceSnapshot) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var plan DataSourceSnapshotModel

	// Load Terraform plan into the model.
	resp.Diagnostics.Append(req.Config.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout.
	timeout, diags := plan.Timeouts.Create(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Use API endpoint in selected zone.
	client, err := utils.SwitchClientZone(
		ctx,
		d.client,
		exoscale.ZoneName(plan.Zone.ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to change exoscale client zone",
			err.Error(),
		)
		return
	}

	// Read remote state.
	id, err := exoscale.ParseUUID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to parse volume ID",
			err.Error(),
		)
		return
	}

	snapshot, err := client.GetBlockStorageSnapshot(
		ctx,
		id,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to get volume snapshot",
			err.Error(),
		)
		return
	}

	// Update state model.
	plan.Name = types.StringValue(snapshot.Name)
	plan.Size = types.Int64Value(snapshot.Size)
	plan.CreatedAt = types.StringValue(snapshot.CreatedAT.String())
	plan.State = types.StringValue(string(snapshot.State))

	plan.Labels = types.MapNull(types.StringType)
	if snapshot.Labels != nil {
		t, dg := types.MapValueFrom(
			ctx,
			types.StringType,
			snapshot.Labels,
		)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		plan.Labels = t
	}

	plan.Volume = types.ObjectNull(SnapshotVolumeModel{}.Types())
	if snapshot.BlockStorageVolume != nil {
		volume := SnapshotVolumeModel{}
		volume.ID = types.StringValue(snapshot.BlockStorageVolume.ID.String())

		t, dg := types.ObjectValueFrom(
			ctx,
			SnapshotVolumeModel{}.Types(),
			volume,
		)

		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		plan.Volume = t
	}

	// Save updated state into Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Trace(ctx, "datasource read done", map[string]interface{}{
		"id": plan.ID,
	})
}
