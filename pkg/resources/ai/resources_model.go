package ai

import (
	"context"
	"errors"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = (*ResourceAIModel)(nil)
var _ resource.ResourceWithImportState = (*ResourceAIModel)(nil)

const markdownDescriptionResourceAIModel = `Manage Exoscale [AI Models](https://community.exoscale.com/product/concrete-ai/dedicated-inference/overview/).`

type ResourceAIModel struct {
	client *exoscale.Client
}

type ResourceAIModelModel struct {
	ID               types.String `tfsdk:"id"`
	Zone             types.String `tfsdk:"zone"`
	Name             types.String `tfsdk:"name"`
	HuggingfaceToken types.String `tfsdk:"huggingface_token"`
	ModelSize        types.Int64  `tfsdk:"model_size"`
	State            types.String `tfsdk:"state"`
	LifecycleStatus  types.String `tfsdk:"lifecycle_status"`
	Visibility       types.String `tfsdk:"visibility"`
	DeprecationDate  types.String `tfsdk:"deprecation_date"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func NewResourceAIModel() resource.Resource {
	return &ResourceAIModel{}
}

func (r *ResourceAIModel) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_model"
}

func (r *ResourceAIModel) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ResourceAIModel) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manage Exoscale AI Models.",
		MarkdownDescription: markdownDescriptionResourceAIModel,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The AI model ID.",
				MarkdownDescription: "The AI model ID.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"zone": schema.StringAttribute{
				Description:         "❗ The Exoscale Zone name.",
				MarkdownDescription: "❗ The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(config.Zones...),
				},
			},
			"name": schema.StringAttribute{
				Description:         "❗ The model name (e.g. the Hugging Face repository).",
				MarkdownDescription: "❗ The model name (e.g. the Hugging Face repository).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"huggingface_token": schema.StringAttribute{
				Description:         "❗ The Hugging Face token used to download the model. It is not returned by the API: it is kept as configured and stays null after an import.",
				MarkdownDescription: "❗ The Hugging Face token used to download the model. It is not returned by the API: it is kept as configured and stays null after an import.",
				Optional:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"model_size": schema.Int64Attribute{
				Description:         "The model size in bytes.",
				MarkdownDescription: "The model size in bytes.",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				Description:         "The model state (creating, downloading, created, ready, error).",
				MarkdownDescription: "The model state (`creating`, `downloading`, `created`, `ready`, `error`).",
				Computed:            true,
			},
			"lifecycle_status": schema.StringAttribute{
				Description:         "The model lifecycle status (preview, active, deprecated, eol).",
				MarkdownDescription: "The model lifecycle status (`preview`, `active`, `deprecated`, `eol`).",
				Computed:            true,
			},
			"visibility": schema.StringAttribute{
				Description:         "The model visibility (private, public).",
				MarkdownDescription: "The model visibility (`private`, `public`).",
				Computed:            true,
			},
			"deprecation_date": schema.StringAttribute{
				Description:         "The model deprecation date.",
				MarkdownDescription: "The model deprecation date.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				Description:         "The model creation date.",
				MarkdownDescription: "The model creation date.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description:         "The model last update date.",
				MarkdownDescription: "The model last update date.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *ResourceAIModel) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceAIModelModel

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

	op, err := client.CreateModel(ctx, exoscale.CreateModelRequest{
		Name:             plan.Name.ValueString(),
		HuggingfaceToken: plan.HuggingfaceToken.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("unable to create AI model", err.Error())
		return
	}

	op, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("create AI model operation failed", err.Error())
		return
	}

	tflog.Warn(ctx, "FAULTY OPERATION", map[string]any{
		"operation": op,
	})
	plan.ID = types.StringValue(op.Reference.ID.String())
	plan.ModelSize = types.Int64Null()
	plan.State = types.StringNull()
	plan.LifecycleStatus = types.StringNull()
	plan.Visibility = types.StringNull()
	plan.DeprecationDate = types.StringNull()
	plan.CreatedAt = types.StringNull()
	plan.UpdatedAt = types.StringNull()

	// Save the model right away: if the lookup below fails, the model is
	// kept in the state (tainted) instead of being orphaned.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	model, err := client.GetModel(ctx, op.Reference.ID)
	if err != nil {
		resp.Diagnostics.AddError("unable to get AI model", err.Error())
		return
	}

	plan.ModelSize = types.Int64Value(model.ModelSize)
	plan.State = types.StringValue(string(model.State))
	plan.LifecycleStatus = types.StringValue(string(model.LifecycleStatus))
	plan.Visibility = types.StringValue(string(model.Visibility))
	if model.DeprecationDate != nil {
		plan.DeprecationDate = types.StringValue(model.DeprecationDate.String())
	}
	plan.CreatedAt = types.StringValue(model.CreatedAT.String())
	plan.UpdatedAt = types.StringValue(model.UpdatedAT.String())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Trace(ctx, "resource created", map[string]any{
		"id": plan.ID.ValueString(),
	})
}

func (r *ResourceAIModel) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceAIModelModel

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
		resp.Diagnostics.AddError("unable to parse AI model ID", err.Error())
		return
	}

	model, err := client.GetModel(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("unable to get AI model", err.Error())
		return
	}

	// The API doesn't return the zone and the Hugging Face token: they are kept
	// as configured (and the token stays null after an import).
	utils.RefreshString(&state.Name, model.Name)
	state.ModelSize = types.Int64Value(model.ModelSize)
	state.State = types.StringValue(string(model.State))
	state.LifecycleStatus = types.StringValue(string(model.LifecycleStatus))
	state.Visibility = types.StringValue(string(model.Visibility))
	state.DeprecationDate = types.StringNull()
	if model.DeprecationDate != nil {
		state.DeprecationDate = types.StringValue(model.DeprecationDate.String())
	}
	state.CreatedAt = types.StringValue(model.CreatedAT.String())
	state.UpdatedAt = types.StringValue(model.UpdatedAT.String())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Trace(ctx, "resource read", map[string]any{
		"id": state.ID.ValueString(),
	})
}

func (r *ResourceAIModel) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All attributes require replace but we still need to update timeouts block.
	var plan ResourceAIModelModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceAIModel) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceAIModelModel

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
		resp.Diagnostics.AddError("unable to parse AI model ID", err.Error())
		return
	}

	op, err := client.DeleteModel(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("unable to delete AI model", err.Error())
		return
	}

	if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("delete AI model operation failed", err.Error())
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]any{
		"id": state.ID.ValueString(),
	})
}

func (r *ResourceAIModel) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	utils.ImportStatePassthroughZonedID(ctx, req, resp)
}
