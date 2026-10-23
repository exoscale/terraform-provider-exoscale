package ai

import (
	"context"
	"errors"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

var _ resource.Resource = (*ResourceAIDeployment)(nil)
var _ resource.ResourceWithImportState = (*ResourceAIDeployment)(nil)

const markdownDescriptionResourceAIDeployment = `Manage Exoscale [AI Deployments](https://community.exoscale.com/product/concrete-ai/dedicated-inference/overview/).`

type ResourceAIDeployment struct {
	client *exoscale.Client
}

type ResourceAIDeploymentModel struct {
	ID                        types.String `tfsdk:"id"`
	Zone                      types.String `tfsdk:"zone"`
	Name                      types.String `tfsdk:"name"`
	ModelID                   types.String `tfsdk:"model_id"`
	GpuType                   types.String `tfsdk:"gpu_type"`
	GpuCount                  types.Int64  `tfsdk:"gpu_count"`
	Replicas                  types.Int64  `tfsdk:"replicas"`
	InferenceEngineVersion    types.String `tfsdk:"inference_engine_version"`
	InferenceEngineParameters types.Set    `tfsdk:"inference_engine_parameters"`
	DeploymentURL             types.String `tfsdk:"deployment_url"`
	ServiceLevel              types.String `tfsdk:"service_level"`
	State                     types.String `tfsdk:"state"`
	StateDetails              types.String `tfsdk:"state_details"`
	CreatedAt                 types.String `tfsdk:"created_at"`
	UpdatedAt                 types.String `tfsdk:"updated_at"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func NewResourceAIDeployment() resource.Resource {
	return &ResourceAIDeployment{}
}

func (r *ResourceAIDeployment) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_deployment"
}

func (r *ResourceAIDeployment) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ResourceAIDeployment) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manage Exoscale AI Deployments.",
		MarkdownDescription: markdownDescriptionResourceAIDeployment,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The AI deployment ID.",
				MarkdownDescription: "The AI deployment ID.",
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
				Description:         "The deployment name.",
				MarkdownDescription: "The deployment name.",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"model_id": schema.StringAttribute{
				Description:         "❗ The ID of the AI model to deploy.",
				MarkdownDescription: "❗ The ID of the [`exoscale_ai_model`](./ai_model.md) to deploy.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"gpu_type": schema.StringAttribute{
				Description:         "❗ The GPU type family (e.g. gpua5000, gpu3080ti). The available GPU types are listed by: exo dedicated-inference deployment instance-type. Your organization GPU limits must allow the deployment: check them with exo limits.",
				MarkdownDescription: "❗ The GPU type family (e.g. `gpua5000`, `gpu3080ti`). The available GPU types are listed by `exo dedicated-inference deployment instance-type`. Your organization GPU limits must allow the deployment: check them with `exo limits`.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"gpu_count": schema.Int64Attribute{
				Description:         "❗ The number of GPUs per replica (1-8).",
				MarkdownDescription: "❗ The number of GPUs per replica (`1`-`8`).",
				Required:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
				Validators: []validator.Int64{
					int64validator.Between(1, 8),
				},
			},
			"replicas": schema.Int64Attribute{
				Description:         "The number of replicas.",
				MarkdownDescription: "The number of replicas.",
				Required:            true,
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"inference_engine_version": schema.StringAttribute{
				Description:         "The inference engine (vLLM) version (e.g. 0.30.0). Defaults to the API default version. The supported versions and the default are listed in the Exoscale API reference: https://community.exoscale.com/reference/api/ai/dedicated-inference/deployment/#create-deployment.",
				MarkdownDescription: "The inference engine (vLLM) version (e.g. `0.30.0`). Defaults to the API default version. The supported versions and the default are listed in the [Exoscale API reference](https://community.exoscale.com/reference/api/ai/dedicated-inference/deployment/#create-deployment).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"inference_engine_parameters": schema.SetAttribute{
				Description:         "Extra inference engine server arguments (e.g. [\"--max-tokens=4096\", \"--enable-prefix-caching\"]). The supported parameters are listed by: exo dedicated-inference deployment create --inference-engine-parameter-help.",
				MarkdownDescription: "Extra inference engine server arguments (e.g. `[\"--max-tokens=4096\", \"--enable-prefix-caching\"]`). The supported parameters are listed by `exo dedicated-inference deployment create --inference-engine-parameter-help`.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"deployment_url": schema.StringAttribute{
				Description:         "The deployment inference endpoint URL.",
				MarkdownDescription: "The deployment inference endpoint URL.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_level": schema.StringAttribute{
				Description:         "The deployment service level.",
				MarkdownDescription: "The deployment service level.",
				Computed:            true,
			},
			"state": schema.StringAttribute{
				Description:         "The deployment state (creating, preparing, deploying, ready, scaling, updating, error).",
				MarkdownDescription: "The deployment state (`creating`, `preparing`, `deploying`, `ready`, `scaling`, `updating`, `error`).",
				Computed:            true,
			},
			"state_details": schema.StringAttribute{
				Description:         "The deployment state details.",
				MarkdownDescription: "The deployment state details.",
				Computed:            true,
			},
			"created_at": schema.StringAttribute{
				Description:         "The deployment creation date.",
				MarkdownDescription: "The deployment creation date.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description:         "The deployment last update date.",
				MarkdownDescription: "The deployment last update date.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}
func (r *ResourceAIDeployment) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceAIDeploymentModel

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

	modelID, err := exoscale.ParseUUID(plan.ModelID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse AI model ID", err.Error())
		return
	}

	request := exoscale.CreateDeploymentRequest{
		Name:     plan.Name.ValueString(),
		Model:    &exoscale.ModelRef{ID: modelID},
		GpuType:  plan.GpuType.ValueString(),
		GpuCount: plan.GpuCount.ValueInt64(),
		Replicas: plan.Replicas.ValueInt64(),
		// Unknown when not configured: left empty for the API to pick its default.
		InferenceEngineVersion: exoscale.InferenceEngineVersion(plan.InferenceEngineVersion.ValueString()),
	}

	if !plan.InferenceEngineParameters.IsNull() && !plan.InferenceEngineParameters.IsUnknown() {
		resp.Diagnostics.Append(plan.InferenceEngineParameters.ElementsAs(ctx, &request.InferenceEngineParameters, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	op, err := client.CreateDeployment(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("unable to create AI deployment", err.Error())
		return
	}

	op, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("create AI deployment operation failed", err.Error())
		return
	}

	plan.ID = types.StringValue(op.Reference.ID.String())
	if plan.InferenceEngineVersion.IsUnknown() {
		plan.InferenceEngineVersion = types.StringNull()
	}
	plan.DeploymentURL = types.StringNull()
	plan.ServiceLevel = types.StringNull()
	plan.State = types.StringNull()
	plan.StateDetails = types.StringNull()
	plan.CreatedAt = types.StringNull()
	plan.UpdatedAt = types.StringNull()

	// Save the deployment right away: if the lookup below fails, the
	// deployment is kept in the state (tainted) instead of being orphaned.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deployment, err := client.GetDeployment(ctx, op.Reference.ID)
	if err != nil {
		resp.Diagnostics.AddError("unable to get AI deployment", err.Error())
		return
	}

	utils.RefreshString(&plan.InferenceEngineVersion, string(deployment.InferenceEngineVersion))
	plan.DeploymentURL = types.StringValue(deployment.DeploymentURL)
	plan.ServiceLevel = types.StringValue(deployment.ServiceLevel)
	plan.State = types.StringValue(string(deployment.State))
	plan.StateDetails = types.StringValue(deployment.StateDetails)
	plan.CreatedAt = types.StringValue(deployment.CreatedAT.String())
	plan.UpdatedAt = types.StringValue(deployment.UpdatedAT.String())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Trace(ctx, "resource created", map[string]any{
		"id": plan.ID.ValueString(),
	})
}
func (r *ResourceAIDeployment) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceAIDeploymentModel

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
		resp.Diagnostics.AddError("unable to parse AI deployment ID", err.Error())
		return
	}

	deployment, err := client.GetDeployment(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("unable to get AI deployment", err.Error())
		return
	}

	utils.RefreshString(&state.Name, deployment.Name)
	if deployment.Model != nil {
		utils.RefreshString(&state.ModelID, deployment.Model.ID.String())
	}
	utils.RefreshString(&state.GpuType, deployment.GpuType)
	utils.RefreshInt64(&state.GpuCount, deployment.GpuCount)
	utils.RefreshInt64(&state.Replicas, deployment.Replicas)
	utils.RefreshString(&state.InferenceEngineVersion, string(deployment.InferenceEngineVersion))

	utils.RefreshStringSet(ctx, &resp.Diagnostics, &state.InferenceEngineParameters, deployment.InferenceEngineParameters)
	if resp.Diagnostics.HasError() {
		return
	}

	state.DeploymentURL = types.StringValue(deployment.DeploymentURL)
	state.ServiceLevel = types.StringValue(deployment.ServiceLevel)
	state.State = types.StringValue(string(deployment.State))
	state.StateDetails = types.StringValue(deployment.StateDetails)
	state.CreatedAt = types.StringValue(deployment.CreatedAT.String())
	state.UpdatedAt = types.StringValue(deployment.UpdatedAT.String())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Trace(ctx, "resource read", map[string]any{
		"id": state.ID.ValueString(),
	})
}

func (r *ResourceAIDeployment) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ResourceAIDeploymentModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Update(ctx, config.DefaultTimeout)
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

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse AI deployment ID", err.Error())
		return
	}

	// Name, inference engine version and parameters are updated in place,
	// replicas are changed by scaling the deployment.
	var updated bool
	request := exoscale.UpdateDeploymentRequest{}

	if !plan.Name.Equal(state.Name) {
		request.Name = plan.Name.ValueString()
		updated = true
	}

	if !plan.InferenceEngineVersion.IsUnknown() && !plan.InferenceEngineVersion.Equal(state.InferenceEngineVersion) {
		request.InferenceEngineVersion = exoscale.InferenceEngineVersion(plan.InferenceEngineVersion.ValueString())
		updated = true
	}

	if !plan.InferenceEngineParameters.Equal(state.InferenceEngineParameters) {
		// TODO(egoscale): the parameters can't be cleared, an empty list is
		// omitted from the request (omitempty).
		if !plan.InferenceEngineParameters.IsNull() && !plan.InferenceEngineParameters.IsUnknown() {
			resp.Diagnostics.Append(plan.InferenceEngineParameters.ElementsAs(ctx, &request.InferenceEngineParameters, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
			updated = true
		}
	}

	if updated {
		op, err := client.UpdateDeployment(ctx, id, request)
		if err != nil {
			resp.Diagnostics.AddError("unable to update AI deployment", err.Error())
			return
		}

		if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
			resp.Diagnostics.AddError("update AI deployment operation failed", err.Error())
			return
		}
	}

	if !plan.Replicas.Equal(state.Replicas) {
		op, err := client.ScaleDeployment(ctx, id, exoscale.ScaleDeploymentRequest{
			Replicas: plan.Replicas.ValueInt64(),
		})
		if err != nil {
			resp.Diagnostics.AddError("unable to scale AI deployment", err.Error())
			return
		}

		if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
			resp.Diagnostics.AddError("scale AI deployment operation failed", err.Error())
			return
		}
	}

	deployment, err := client.GetDeployment(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("unable to get AI deployment", err.Error())
		return
	}

	utils.RefreshString(&plan.InferenceEngineVersion, string(deployment.InferenceEngineVersion))
	plan.DeploymentURL = types.StringValue(deployment.DeploymentURL)
	plan.ServiceLevel = types.StringValue(deployment.ServiceLevel)
	plan.State = types.StringValue(string(deployment.State))
	plan.StateDetails = types.StringValue(deployment.StateDetails)
	plan.CreatedAt = types.StringValue(deployment.CreatedAT.String())
	plan.UpdatedAt = types.StringValue(deployment.UpdatedAT.String())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Trace(ctx, "resource updated", map[string]any{
		"id": plan.ID.ValueString(),
	})
}
func (r *ResourceAIDeployment) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceAIDeploymentModel

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
		resp.Diagnostics.AddError("unable to parse AI deployment ID", err.Error())
		return
	}

	op, err := client.DeleteDeployment(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("unable to delete AI deployment", err.Error())
		return
	}

	if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("delete AI deployment operation failed", err.Error())
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]any{
		"id": state.ID.ValueString(),
	})
}

// ImportState only sets the ID and zone (<ID>@<zone>): Read refreshes the
// other attributes from the API.
func (r *ResourceAIDeployment) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	utils.ImportStatePassthroughZonedID(ctx, req, resp)
}
