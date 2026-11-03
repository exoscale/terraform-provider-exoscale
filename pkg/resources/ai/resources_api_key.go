package ai

import (
	"context"
	"errors"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

var _ resource.Resource = (*ResourceAIAPIKey)(nil)
var _ resource.ResourceWithImportState = (*ResourceAIAPIKey)(nil)

const markdownDescriptionResourceAIAPIKey = `Manage Exoscale [AI API Keys](https://community.exoscale.com/product/concrete-ai/dedicated-inference/overview/), used to call AI deployments and public models.`

type ResourceAIAPIKey struct {
	client *exoscale.Client
}

func NewResourceAIAPIKey() resource.Resource {
	return &ResourceAIAPIKey{}
}

type ResourceAIAPIKeyModel struct {
	ID             types.String `tfsdk:"id"`
	Zone           types.String `tfsdk:"zone"`
	Name           types.String `tfsdk:"name"`
	AllDeployments types.Bool   `tfsdk:"all_deployments"`
	AllModels      types.Bool   `tfsdk:"all_models"`
	DeploymentIDs  types.Set    `tfsdk:"deployment_ids"`
	Models         types.Set    `tfsdk:"models"`
	Secret         types.String `tfsdk:"secret"`
	CreatedAt      types.String `tfsdk:"created_at"`
	UpdatedAt      types.String `tfsdk:"updated_at"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *ResourceAIAPIKey) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ai_api_key"
}

func (r *ResourceAIAPIKey) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *ResourceAIAPIKey) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manage Exoscale AI API Keys, used to call AI deployments and public models.",
		MarkdownDescription: markdownDescriptionResourceAIAPIKey,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The AI API key ID.",
				MarkdownDescription: "The AI API key ID.",
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
				Description:         "❗ The AI API key name (1-50 characters).",
				MarkdownDescription: "❗ The AI API key name (`1`-`50` characters).",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 50),
				},
			},
			"all_deployments": schema.BoolAttribute{
				Description:         "Grant access to all the deployments of the organization. Takes precedence over deployment_ids, which is ignored when set (default: false).",
				MarkdownDescription: "Grant access to all the deployments of the organization. Takes precedence over `deployment_ids`, which is ignored when set (default: `false`).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"all_models": schema.BoolAttribute{
				Description:         "Grant access to all the public models. Takes precedence over models, which is ignored when set (default: false).",
				MarkdownDescription: "Grant access to all the public models. Takes precedence over `models`, which is ignored when set (default: `false`).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"deployment_ids": schema.SetAttribute{
				Description:         "The IDs of the AI deployments the key has access to. Unset denies access to all deployments.",
				MarkdownDescription: "The IDs of the [`exoscale_ai_deployment`](./ai_deployment.md) the key has access to. Unset denies access to all deployments.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"models": schema.SetAttribute{
				Description:         "The names of the public models the key has access to. Unset denies access to all public models.",
				MarkdownDescription: "The names of the public models the key has access to. Unset denies access to all public models.",
				ElementType:         types.StringType,
				Optional:            true,
			},
			"secret": schema.StringAttribute{
				Description:         "The AI API key secret. It is only returned at creation: it stays null after an import.",
				MarkdownDescription: "The AI API key secret. It is only returned at creation: it stays null after an import.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				Description:         "The AI API key creation date.",
				MarkdownDescription: "The AI API key creation date.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Description:         "The AI API key last update date.",
				MarkdownDescription: "The AI API key last update date.",
				Computed:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *ResourceAIAPIKey) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceAIAPIKeyModel

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

	request := exoscale.CreateAIAPIKeyRequest{
		Name: plan.Name.ValueString(),
	}

	request.AllDeployments = plan.AllDeployments.ValueBoolPointer()
	request.AllModels = plan.AllModels.ValueBoolPointer()

	if !plan.DeploymentIDs.IsUnknown() && !plan.DeploymentIDs.IsNull() {
		deployments, dg := deploymentRefs(ctx, plan.DeploymentIDs)
		resp.Diagnostics.Append(dg...)
		if resp.Diagnostics.HasError() {
			return
		}
		request.Deployments = &deployments
	}

	if !plan.Models.IsUnknown() && !plan.Models.IsNull() {
		models := exoscale.AIAPIKeyModels{}
		resp.Diagnostics.Append(plan.Models.ElementsAs(ctx, &models, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		request.Models = &models
	}

	// The key is created synchronously: there is no operation to wait for.
	apiKey, err := client.CreateAIAPIKey(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("unable to create AI API key", err.Error())
		return
	}

	plan.ID = types.StringValue(apiKey.ID.String())
	plan.Secret = types.StringValue(apiKey.Value)
	plan.CreatedAt = types.StringValue(apiKey.CreatedAT.String())
	plan.UpdatedAt = types.StringValue(apiKey.UpdatedAT.String())

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Trace(ctx, "resource created", map[string]any{
		"id": plan.ID.ValueString(),
	})
}

func (r *ResourceAIAPIKey) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceAIAPIKeyModel

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
		resp.Diagnostics.AddError("unable to parse AI API key ID", err.Error())
		return
	}

	apiKey, err := client.GetAIAPIKey(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("unable to get AI API key", err.Error())
		return
	}

	// A revoked key can't be used anymore: it is handled as deleted.
	if apiKey.RevokedAT != nil {
		resp.State.RemoveResource(ctx)
		return
	}

	// The API doesn't return the zone and the key secret: they are kept as is
	// (and the secret stays null after an import).
	utils.RefreshString(&state.Name, apiKey.Name)
	if apiKey.AllDeployments != nil {
		state.AllDeployments = types.BoolValue(*apiKey.AllDeployments)
	}
	if apiKey.AllModels != nil {
		state.AllModels = types.BoolValue(*apiKey.AllModels)
	}

	deploymentIDs := []string{}
	if apiKey.Deployments != nil {
		for _, deployment := range *apiKey.Deployments {
			deploymentIDs = append(deploymentIDs, deployment.ID.String())
		}
	}
	utils.RefreshStringSet(ctx, &resp.Diagnostics, &state.DeploymentIDs, deploymentIDs)

	models := []string{}
	if apiKey.Models != nil {
		models = *apiKey.Models
	}
	utils.RefreshStringSet(ctx, &resp.Diagnostics, &state.Models, models)
	if resp.Diagnostics.HasError() {
		return
	}

	state.CreatedAt = types.StringValue(apiKey.CreatedAT.String())
	state.UpdatedAt = types.StringValue(apiKey.UpdatedAT.String())

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Trace(ctx, "resource read", map[string]any{
		"id": state.ID.ValueString(),
	})
}

func (r *ResourceAIAPIKey) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ResourceAIAPIKeyModel

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
		resp.Diagnostics.AddError("unable to parse AI API key ID", err.Error())
		return
	}

	var updated bool
	request := exoscale.UpdateAIAPIKeyRequest{}

	if !plan.AllDeployments.Equal(state.AllDeployments) {
		request.AllDeployments = plan.AllDeployments.ValueBoolPointer()
		updated = true
	}

	if !plan.AllModels.Equal(state.AllModels) {
		request.AllModels = plan.AllModels.ValueBoolPointer()
		updated = true
	}

	// An unset allowlist is sent as an empty one, which denies access.
	if !plan.DeploymentIDs.Equal(state.DeploymentIDs) {
		deployments := exoscale.AIAPIKeyDeployments{}
		if !plan.DeploymentIDs.IsNull() && !plan.DeploymentIDs.IsUnknown() {
			refs, dg := deploymentRefs(ctx, plan.DeploymentIDs)
			resp.Diagnostics.Append(dg...)
			if resp.Diagnostics.HasError() {
				return
			}
			deployments = refs
		}
		request.Deployments = &deployments
		updated = true
	}

	if !plan.Models.Equal(state.Models) {
		models := exoscale.AIAPIKeyModels{}
		if !plan.Models.IsNull() && !plan.Models.IsUnknown() {
			resp.Diagnostics.Append(plan.Models.ElementsAs(ctx, &models, false)...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
		request.Models = &models
		updated = true
	}

	if updated {
		// The key is updated synchronously: there is no operation to wait for.
		apiKey, err := client.UpdateAIAPIKey(ctx, id, request)
		if err != nil {
			resp.Diagnostics.AddError("unable to update AI API key", err.Error())
			return
		}

		plan.UpdatedAt = types.StringValue(apiKey.UpdatedAT.String())
	} else {
		plan.UpdatedAt = state.UpdatedAt
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Trace(ctx, "resource updated", map[string]any{
		"id": plan.ID.ValueString(),
	})
}

func (r *ResourceAIAPIKey) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceAIAPIKeyModel

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
		resp.Diagnostics.AddError("unable to parse AI API key ID", err.Error())
		return
	}

	// The API has no delete: the key is revoked.
	op, err := client.RevokeAIAPIKey(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("unable to revoke AI API key", err.Error())
		return
	}

	if _, err := client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("revoke AI API key operation failed", err.Error())
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]any{
		"id": state.ID.ValueString(),
	})
}

// ImportState only sets the ID and zone (<ID>@<zone>): Read refreshes the
// other attributes from the API, except the key secret which is never returned.
func (r *ResourceAIAPIKey) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	utils.ImportStatePassthroughZonedID(ctx, req, resp)
}

// deploymentRefs converts a set of deployment IDs into the API allowlist.
func deploymentRefs(ctx context.Context, set types.Set) (exoscale.AIAPIKeyDeployments, diag.Diagnostics) {
	ids := []string{}
	diags := set.ElementsAs(ctx, &ids, false)
	if diags.HasError() {
		return nil, diags
	}

	deployments := exoscale.AIAPIKeyDeployments{}
	for _, id := range ids {
		uuid, err := exoscale.ParseUUID(id)
		if err != nil {
			diags.AddError("unable to parse AI deployment ID", err.Error())
			return nil, diags
		}
		deployments = append(deployments, exoscale.AIAPIKeyDeploymentRef{ID: uuid})
	}

	return deployments, diags
}
