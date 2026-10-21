package kms

import (
	"context"
	"errors"
	"fmt"
	"strings"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

var _ resource.Resource = &ResourceKMSKey{}
var _ resource.ResourceWithImportState = &ResourceKMSKey{}

// kmsKeyDeletionDelayDays is the minimum scheduled deletion delay accepted by the KMS API.
// Keys cannot be deleted immediately; they enter a pending-deletion state for at least this many days.
const kmsKeyDeletionDelayDays = 7

// ResourceKMSKeyModel holds the Terraform state for a KMS key.
type ResourceKMSKeyModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Zone        types.String `tfsdk:"zone"`
	Description types.String `tfsdk:"description"`
	MultiZone   types.Bool   `tfsdk:"multi_zone"`
	Usage       types.String `tfsdk:"usage"`
	Status      types.String `tfsdk:"status"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

type ResourceKMSKey struct {
	client *exoscale.Client
}

func NewResourceKMSKey() resource.Resource {
	return &ResourceKMSKey{}
}

func (r *ResourceKMSKey) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kms_key"
}

func (r *ResourceKMSKey) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage Exoscale KMS Keys.",
		Description:         "Manage Exoscale KMS Keys.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Description:         "The ID of this resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "❗ The name of the KMS Key.",
				Description:         "The name of the KMS Key.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"zone": schema.StringAttribute{
				MarkdownDescription: "❗ The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Description:         "The Exoscale Zone name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "❗ A description of the KMS Key.",
				Description:         "A description of the KMS Key.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"multi_zone": schema.BoolAttribute{
				MarkdownDescription: "❗ Whether the key is replicated across multiple zones.",
				Description:         "Whether the key is replicated across multiple zones.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"usage": schema.StringAttribute{
				MarkdownDescription: "❗ The key usage purpose (e.g. `encrypt-decrypt`).",
				Description:         "The key usage purpose (e.g. encrypt-decrypt).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("encrypt-decrypt"),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the KMS Key.",
				Description:         "The current status of the KMS Key.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *ResourceKMSKey) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ResourceKMSKey) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceKMSKeyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Create(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(plan.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	createReq := exoscale.CreateKmsKeyRequest{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
	}

	if !plan.MultiZone.IsUnknown() {
		v := plan.MultiZone.ValueBool()
		createReq.MultiZone = &v
	}

	if !plan.Usage.IsUnknown() {
		createReq.Usage = exoscale.CreateKmsKeyRequestUsage(plan.Usage.ValueString())
	}

	key, err := client.CreateKmsKey(ctx, createReq)
	if err != nil {
		resp.Diagnostics.AddError("API error creating KMS key", err.Error())
		return
	}

	plan.ID = types.StringValue(key.ID.String())
	plan.Status = types.StringValue(string(key.Status))

	if plan.Usage.IsUnknown() {
		plan.Usage = types.StringValue(key.Usage)
	}
	if plan.MultiZone.IsUnknown() {
		plan.MultiZone = types.BoolValue(*key.MultiZone)
	}
	if plan.Description.IsUnknown() {
		plan.Description = types.StringValue(key.Description)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceKMSKey) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceKMSKeyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
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

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(state.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse resource ID", err.Error())
		return
	}

	key, err := client.GetKmsKey(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			tflog.Info(ctx, "KMS key not found, removing from state", map[string]any{})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("API error reading KMS key", err.Error())
		return
	}

	// pending-deletion means the key is scheduled for deletion. treating it as gone.
	if key.Status == exoscale.GetKmsKeyResponseStatusPendingDeletion {
		tflog.Info(ctx, "KMS key is pending deletion, removing from state", map[string]any{})
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(key.Name)
	state.Status = types.StringValue(string(key.Status))
	state.Usage = types.StringValue(key.Usage)
	state.MultiZone = types.BoolValue(*key.MultiZone)
	state.Description = types.StringValue(key.Description)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is a no-op: all mutable attributes use RequiresReplace.
func (r *ResourceKMSKey) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
}

func (r *ResourceKMSKey) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceKMSKeyModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := state.Timeouts.Delete(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(state.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse resource ID", err.Error())
		return
	}

	_, err = client.ScheduleKmsKeyDeletion(ctx, id, exoscale.ScheduleKmsKeyDeletionRequest{
		DelayDays: kmsKeyDeletionDelayDays,
	})
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("API error scheduling KMS key deletion", err.Error())
		return
	}
}

func (r *ResourceKMSKey) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, "@")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"unexpected import identifier",
			fmt.Sprintf("Expected import identifier with format: id@zone. Got: %q", req.ID),
		)
		return
	}

	// Set timeouts (quirk https://github.com/hashicorp/terraform-plugin-framework-timeouts/issues/46)
	var t timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &t)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &ResourceKMSKeyModel{
		ID:       types.StringValue(idParts[0]),
		Zone:     types.StringValue(idParts[1]),
		Timeouts: t,
	})...)
}
