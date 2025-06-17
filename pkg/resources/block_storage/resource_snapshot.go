package block_storage

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const ResourceSnapshotDescription = `Manage [Exoscale Block Storage](https://community.exoscale.com/product/storage/block-storage/) Volume Snapshot.

Block Storage offers persistent externally attached volumes for your workloads.
`

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceSnapshot{}
var _ resource.ResourceWithImportState = &ResourceSnapshot{}

// ResourceSnapshot defines the resource implementation.
type ResourceSnapshot struct {
	client *exoscale.Client
}

// NewResourceSnapshot creates instance of ResourceSnapshot.
func NewResourceSnapshot() resource.Resource {
	return &ResourceSnapshot{}
}

// ResourceSnapshotModel defines the resource data model.
type ResourceSnapshotModel struct {
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
func (r *ResourceSnapshot) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_block_storage_volume_snapshot"
}

// Schema defines resource attributes.
func (r *ResourceSnapshot) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: ResourceSnapshotDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Volume snapshot name.",
				Required:            true,
			},
			"volume": schema.SingleNestedAttribute{
				MarkdownDescription: "Volume from which to create a snapshot.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "Snapshot ID.",
						Required:            true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
				},
			},
			"zone": schema.StringAttribute{
				MarkdownDescription: "❗ The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"labels": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Resource labels.",
				Optional:            true,
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "Snapshot size in GB.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Snapshot creation date.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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

// Configure sets up resource dependencies.
func (r *ResourceSnapshot) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

// Create resources by receiving Terraform configuration and plan data, performing creation logic, and saving Terraform state data.
func (r *ResourceSnapshot) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceSnapshotModel

	// Load Terraform plan plan into the model.
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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
		r.client,
		exoscale.ZoneName(plan.Zone.ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to change exoscale client zone",
			err.Error(),
		)
		return
	}

	// Prepare API request to create a resource.
	// All required attributes must be present in the request.
	// Optional attributes not defined in the plan should leave default value in request,
	// for egoscale to interpret it as unset value and handle it accordingly.

	request := exoscale.CreateBlockStorageSnapshotRequest{
		Name: plan.Name.ValueString(),
	}

	volume := SnapshotVolumeModel{}

	dg := plan.Volume.As(ctx, &volume, basetypes.ObjectAsOptions{})
	if dg.HasError() {
		resp.Diagnostics.Append(dg...)
		return
	}

	vid, err := exoscale.ParseUUID(volume.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to parse volume ID",
			err.Error(),
		)
		return
	}

	if len(plan.Labels.Elements()) > 0 {
		labels := exoscale.Labels{}

		dg := plan.Labels.ElementsAs(ctx, &labels, false)
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		request.Labels = labels
	}

	op, err := client.CreateBlockStorageSnapshot(ctx, vid, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to create volume snapshot",
			err.Error(),
		)
		return
	}

	_, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to create volume snapshot",
			err.Error(),
		)
		return
	}

	plan.ID = types.StringValue(op.Reference.ID.String())
	id, err := exoscale.ParseUUID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to parse volume ID",
			err.Error(),
		)
		return
	}

	// Update computed attributes before saving the state.
	// Compute attributes cannot be undefined in the state, but may be null.

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

	plan.Name = types.StringValue(snapshot.Name)
	plan.Size = types.Int64Value(snapshot.Size)
	plan.CreatedAt = types.StringValue(snapshot.CreatedAT.String())
	plan.State = types.StringValue(string(snapshot.State))

	// Save plan into Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Trace(ctx, "resource created", map[string]interface{}{
		"id": plan.ID,
	})
}

// Read (refresh) resources by receiving Terraform prior state data, performing read logic, and saving refreshed Terraform state data.
func (r *ResourceSnapshot) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceSnapshotModel

	// Load Terraform prior data into the model.
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout.
	timeout, diags := state.Timeouts.Create(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Use API endpoint in selected zone.
	client, err := utils.SwitchClientZone(
		ctx,
		r.client,
		exoscale.ZoneName(state.Zone.ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to change exoscale client zone",
			err.Error(),
		)
		return
	}

	// Read remote state.
	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to parse snapshot ID",
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
			"unable to get storage snapshot",
			err.Error(),
		)
		return
	}

	// Update state model.
	state.Name = types.StringValue(snapshot.Name)
	state.Size = types.Int64Value(snapshot.Size)
	state.CreatedAt = types.StringValue(snapshot.CreatedAT.String())
	state.State = types.StringValue(string(snapshot.State))

	state.Volume = types.ObjectNull(SnapshotVolumeModel{}.Types())
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

		state.Volume = t
	}

	if !state.Labels.IsNull() {
		state.Labels = types.MapNull(types.StringType)

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

			state.Labels = t
		}
	}

	// Save updated state into Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Trace(ctx, "resource read done", map[string]interface{}{
		"id": state.ID,
	})
}

// Update resources in-place by receiving Terraform prior state, configuration, and plan data, performing update logic, and saving updated Terraform state data.
func (r *ResourceSnapshot) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan ResourceSnapshotModel

	// Read Terraform prior state data (for comparison) into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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
		r.client,
		exoscale.ZoneName(plan.Zone.ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to change exoscale client zone",
			err.Error(),
		)
		return
	}

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to parse volume ID",
			err.Error(),
		)
		return
	}

	update := false
	updateReq := exoscale.UpdateBlockStorageSnapshotRequest{}

	if !plan.Name.Equal(state.Name) {
		update = true

		updateReq.Name = plan.Name.ValueStringPointer()
	}

	if !plan.Labels.Equal(state.Labels) {
		update = true

		if !plan.Labels.IsNull() {
			resp.Diagnostics.Append(plan.Labels.ElementsAs(ctx, &updateReq.Labels, false)...)
		}
	}

	if update {
		op, err := client.UpdateBlockStorageSnapshot(ctx, id, updateReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"unable to update block storage snapshot",
				err.Error(),
			)
			return
		}

		_, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
		if err != nil {
			resp.Diagnostics.AddError(
				"unable to update block storage snapshot",
				err.Error(),
			)
			return
		}
	}

	state.Labels = plan.Labels
	state.Name = plan.Name

	// Save updated state into Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Trace(ctx, "resource update done", map[string]interface{}{
		"id": state.ID,
	})
}

// Delete resources by receiving Terraform prior state data and performing deletion logic.
func (r *ResourceSnapshot) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceSnapshotModel

	// Load Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout.
	timeout, diags := state.Timeouts.Create(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Use API endpoint in selected zone.
	client, err := utils.SwitchClientZone(
		ctx,
		r.client,
		exoscale.ZoneName(state.Zone.ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to change exoscale client zone",
			err.Error(),
		)
		return
	}

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to parse snapshot ID",
			err.Error(),
		)
		return
	}

	// Delete remote resource.
	op, err := client.DeleteBlockStorageSnapshot(
		ctx,
		id,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete snapshot",
			err.Error(),
		)
		return
	}

	_, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to delete snapshot",
			err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]interface{}{
		"id": state.ID,
	})
}

// ImportState lets Terraform begin managing existing infrastructure resources.
func (r *ResourceSnapshot) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, "@")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"unexpected import identifier",
			fmt.Sprintf("Expected import identifier with format: id@zone. Got: %q", req.ID),
		)
		return
	}

	var state ResourceSnapshotModel

	// Set timeouts (quirk https://github.com/hashicorp/terraform-plugin-framework-timeouts/issues/46)
	var timeouts timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &timeouts)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Timeouts = timeouts

	state.ID = types.StringValue(idParts[0])
	state.Zone = types.StringValue(idParts[1])

	// Set null values
	state.Labels = types.MapNull(types.StringType)
	state.Volume = types.ObjectNull(SnapshotVolumeModel{}.Types())

	// Save state into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Trace(ctx, "resource imported", map[string]interface{}{
		"id": state.ID,
	})
}
