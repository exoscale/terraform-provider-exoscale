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

const ResourceVolumeDescription = `Manage [Exoscale Block Storage](https://community.exoscale.com/documentation/block-storage/) Volume.

Block Storage offers persistent externally attached volumes for your workloads.
`

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceVolume{}
var _ resource.ResourceWithImportState = &ResourceVolume{}

// ResourceVolume defines the resource implementation.
type ResourceVolume struct {
	client *exoscale.Client
}

// NewResourceVolume creates instance of ResourceVolume.
func NewResourceVolume() resource.Resource {
	return &ResourceVolume{}
}

// ResourceVolumeModel defines the resource data model.
type ResourceVolumeModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Size           types.Int64  `tfsdk:"size"`
	Blocksize      types.Int64  `tfsdk:"blocksize"`
	CreatedAt      types.String `tfsdk:"created_at"`
	Labels         types.Map    `tfsdk:"labels"`
	SnapshotTarget types.Object `tfsdk:"snapshot_target"`
	State          types.String `tfsdk:"state"`
	Zone           types.String `tfsdk:"zone"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

// Metadata specifies resource name.
func (r *ResourceVolume) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_block_storage_volume"
}

// Schema defines resource attributes.
func (r *ResourceVolume) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: ResourceVolumeDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Volume name.",
				Required:            true,
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
			"size": schema.Int64Attribute{
				MarkdownDescription: "Volume size in GB (default 10). If volume is attached, instance must be stopped to update this value. Volume can only grow, cannot be shrunk.",
				Optional:            true,
			},
			"labels": schema.MapAttribute{
				ElementType:         types.StringType,
				MarkdownDescription: "Resource labels.",
				Optional:            true,
			},
			"snapshot_target": schema.SingleNestedAttribute{
				MarkdownDescription: "Block storage snapshot to use when creating a volume. Read-only after creation.",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "Snapshot ID.",
						Optional:            true,
					},
				},
			},
			"blocksize": schema.Int64Attribute{
				MarkdownDescription: "Volume block size.",
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "Volume creation date.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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

// Configure sets up resource dependencies.
func (r *ResourceVolume) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

// Create resources by receiving Terraform configuration and plan data, performing creation logic, and saving Terraform state data.
func (r *ResourceVolume) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceVolumeModel

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

	request := exoscale.CreateBlockStorageVolumeRequest{
		Name: plan.Name.ValueString(),
	}

	if !plan.Size.IsNull() {
		request.Size = plan.Size.ValueInt64()
	} else if plan.SnapshotTarget.IsNull() {
		request.Size = 10
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

	if !plan.SnapshotTarget.IsNull() {
		snapshot := VolumeSnapshotTargetModel{}

		dg := plan.SnapshotTarget.As(ctx, &snapshot, basetypes.ObjectAsOptions{})
		if dg.HasError() {
			resp.Diagnostics.Append(dg...)
			return
		}

		if !snapshot.ID.IsNull() {
			uuid, err := exoscale.ParseUUID(snapshot.ID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"unable to parse snapshot ID",
					err.Error(),
				)
				return
			}

			request.BlockStorageSnapshot = &exoscale.BlockStorageSnapshotTarget{
				ID: uuid,
			}
		}
	}

	op, err := client.CreateBlockStorageVolume(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create Volume",
			err.Error(),
		)
		return
	}

	_, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to create block storage",
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

	plan.Blocksize = types.Int64Value(volume.Blocksize)
	plan.CreatedAt = types.StringValue(volume.CreatedAT.String())
	plan.State = types.StringValue(string(volume.State))

	// Save plan into Terraform state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Trace(ctx, "resource created", map[string]interface{}{
		"id": plan.ID,
	})
}

// Read (refresh) resources by receiving Terraform prior state data, performing read logic, and saving refreshed Terraform state data.
func (r *ResourceVolume) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceVolumeModel

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
	state.Name = types.StringValue(volume.Name)
	state.Blocksize = types.Int64Value(volume.Blocksize)
	state.CreatedAt = types.StringValue(volume.CreatedAT.String())
	state.State = types.StringValue(string(volume.State))

	if !state.Size.IsNull() {
		state.Size = types.Int64Value(volume.Size)
	}

	if !state.Labels.IsNull() {
		state.Labels = types.MapNull(types.StringType)

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
func (r *ResourceVolume) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan ResourceVolumeModel

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

	// Resize Volume
	if !plan.Size.Equal(state.Size) {
		request := exoscale.ResizeBlockStorageVolumeRequest{
			Size: plan.Size.ValueInt64(),
		}

		volume, err := client.ResizeBlockStorageVolume(ctx, id, request)
		if err != nil {
			resp.Diagnostics.AddError(
				"unable to resize block storage volume",
				err.Error(),
			)
			return
		}

		state.Size = types.Int64Value(volume.Size)
	}

	update := false
	updateReq := exoscale.UpdateBlockStorageVolumeRequest{}

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
		op, err := client.UpdateBlockStorageVolume(ctx, id, updateReq)
		if err != nil {
			resp.Diagnostics.AddError(
				"unable to update block storage volume",
				err.Error(),
			)
			return
		}

		_, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
		if err != nil {
			resp.Diagnostics.AddError(
				"unable to update block storage volume",
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
func (r *ResourceVolume) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceVolumeModel

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
			"unable to parse volume ID",
			err.Error(),
		)
		return
	}

	// Detach volume if attached.
	op, err := client.DetachBlockStorageVolume(
		ctx,
		id,
	)
	if err != nil {
		// Ideally we would have a custom error defined in OpenAPI spec & egoscale.
		// For now we just check the error text.
		if strings.HasSuffix(err.Error(), "Volume not attached") {
			tflog.Debug(ctx, "volume not attached")
		} else {
			resp.Diagnostics.AddError(
				"unable to detach volume",
				err.Error(),
			)
			return
		}
	} else {
		_, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
		if err != nil {
			resp.Diagnostics.AddError(
				"failed to create block storage",
				err.Error(),
			)
			return
		}
	}

	// Delete remote resource.
	op, err = client.DeleteBlockStorageVolume(
		ctx,
		id,
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete volume",
			err.Error(),
		)
		return
	}

	_, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError(
			"failed to delete block storage",
			err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]interface{}{
		"id": state.ID,
	})
}

// ImportState lets Terraform begin managing existing infrastructure resources.
func (r *ResourceVolume) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, "@")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"unexpected import identifier",
			fmt.Sprintf("Expected import identifier with format: id@zone. Got: %q", req.ID),
		)
		return
	}

	var state ResourceVolumeModel

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
	state.SnapshotTarget = types.ObjectNull(VolumeSnapshotTargetModel{}.Types())

	// Save state into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Trace(ctx, "resource imported", map[string]interface{}{
		"id": state.ID,
	})
}
