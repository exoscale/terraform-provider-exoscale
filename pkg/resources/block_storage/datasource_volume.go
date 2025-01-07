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

const DataSourceVolumeDescription = `Fetch [Exoscale Block Storage](https://community.exoscale.com/documentation/block-storage/) Volume.

Block Storage offers persistent externally attached volumes for your workloads.

Corresponding resource: [exoscale_block_storage_volume](../resources/block_storage_volume.md).`

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSourceWithConfigure = &DataSourceVolume{}

// DataSourceVolume defines the resource implementation.
type DataSourceVolume struct {
	client *exoscale.Client
}

// NewDataSourceVolume creates instance of ResourceVolume.
func NewDataSourceVolume() datasource.DataSource {
	return &DataSourceVolume{}
}

// DataSourceVolumeModel defines the resource data model.
type DataSourceVolumeModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Size      types.Int64  `tfsdk:"size"`
	Blocksize types.Int64  `tfsdk:"blocksize"`
	CreatedAt types.String `tfsdk:"created_at"`
	Instance  types.Object `tfsdk:"instance"`
	Labels    types.Map    `tfsdk:"labels"`
	Snapshots types.Set    `tfsdk:"snapshots"`
	State     types.String `tfsdk:"state"`
	Zone      types.String `tfsdk:"zone"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// Metadata specifies resource name.
func (d *DataSourceVolume) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_block_storage_volume"
}

// Schema defines resource attributes.
func (d *DataSourceVolume) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: DataSourceVolumeDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Volume ID to match.",
				Required:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Volume name.",
				Computed:            true,
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "Volume size in GB.",
				Computed:            true,
			},
			"zone": schema.StringAttribute{
				MarkdownDescription: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"blocksize": schema.Int64Attribute{
				MarkdownDescription: "Volume block size.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Volume creation date.",
				Computed:            true,
			},
			"instance": schema.SingleNestedAttribute{
				MarkdownDescription: "Volume attached instance.",
				Computed:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "Instance ID.",
						Computed:            true,
					},
				},
			},
			"labels": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Resource labels.",
				Computed:            true,
			},
			"snapshots": schema.SetNestedAttribute{
				MarkdownDescription: "Volume snapshots.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Snapshot ID.",
							Computed:            true,
						},
					},
				},
			},
			"state": schema.StringAttribute{
				MarkdownDescription: "Volume state.",
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
func (d *DataSourceVolume) Configure(
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
func (d *DataSourceVolume) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var plan DataSourceVolumeModel

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

	volume, err := client.GetBlockStorageVolume(
		ctx,
		id,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to get storage volume",
			err.Error(),
		)
		return
	}

	// Update state model.
	plan.Name = types.StringValue(volume.Name)
	plan.Size = types.Int64Value(volume.Size)
	plan.Blocksize = types.Int64Value(volume.Blocksize)
	plan.CreatedAt = types.StringValue(volume.CreatedAT.String())
	plan.State = types.StringValue(string(volume.State))

	plan.Labels = types.MapNull(types.StringType)
	if volume.Labels != nil {
		t, dg := types.MapValueFrom(
			ctx,
			types.StringType,
			volume.Labels,
		)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		plan.Labels = t
	}

	plan.Instance = types.ObjectNull(VolumeInstanceModel{}.Types())
	if volume.Instance != nil {
		instance := VolumeInstanceModel{}
		instance.ID = types.StringValue(volume.Instance.ID.String())

		i, dg := types.ObjectValueFrom(
			ctx,
			VolumeInstanceModel{}.Types(),
			instance,
		)

		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		plan.Instance = i
	}

	plan.Snapshots = types.SetNull(types.ObjectType{AttrTypes: VolumeSnapshotModel{}.Types()})
	if volume.BlockStorageSnapshots != nil {
		snapshots := []VolumeSnapshotModel{}
		for _, s := range volume.BlockStorageSnapshots {
			snapshots = append(snapshots, VolumeSnapshotModel{
				ID: types.StringValue(s.ID.String()),
			})
		}

		t, dg := types.SetValueFrom(
			ctx,
			types.ObjectType{AttrTypes: VolumeSnapshotModel{}.Types()},
			snapshots,
		)

		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		plan.Snapshots = t
	}

	// Save updated state into Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Trace(ctx, "datasource read done", map[string]interface{}{
		"id": plan.ID,
	})
}
