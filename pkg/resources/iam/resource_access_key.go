package iam

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	// The legacy IAM access keys (/v2/access-key) are not part of the
	// egoscale v3 API spec.
	exoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const markdownDescriptionResourceAccessKey = `Manage Exoscale [IAM Access Keys](https://community.exoscale.com/documentation/iam/)`

var _ resource.ResourceWithConfigure = (*ResourceAccessKey)(nil)
var _ resource.ResourceWithImportState = (*ResourceAccessKey)(nil)

func NewResourceAccessKey() resource.Resource {
	return &ResourceAccessKey{}
}

// ResourceAccessKey defines the IAM access key resource implementation.
type ResourceAccessKey struct {
	client *exoscale.Client
	env    string
}

// ResourceAccessKeyModel describes the IAM access key resource data model.
type ResourceAccessKeyModel struct {
	ID             types.String `tfsdk:"id"`
	Key            types.String `tfsdk:"key"`
	Name           types.String `tfsdk:"name"`
	Operations     types.Set    `tfsdk:"operations"`
	Resources      types.Set    `tfsdk:"resources"`
	Secret         types.String `tfsdk:"secret"`
	Tags           types.Set    `tfsdk:"tags"`
	TagsOperations types.Set    `tfsdk:"tags_operations"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *ResourceAccessKey) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_access_key"
}

func (r *ResourceAccessKey) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manage Exoscale IAM Access Keys",
		MarkdownDescription: markdownDescriptionResourceAccessKey,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The IAM access key (identifier).",
				MarkdownDescription: "The IAM access key (identifier).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key": schema.StringAttribute{
				Description:         "The IAM access key (identifier).",
				MarkdownDescription: "The IAM access key (identifier).",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description:         "❗ The IAM access key name.",
				MarkdownDescription: "❗ The IAM access key name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"operations": schema.SetAttribute{
				Description:         "❗ A list of API operations to restrict the key to.",
				MarkdownDescription: "❗ A list of API operations to restrict the key to.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					operationsFromTags(),
					setplanmodifier.UseStateForUnknown(),
					setRequiresReplace(),
				},
			},
			"resources": schema.SetAttribute{
				Description:         "❗ A list of API resources to restrict the key to (<domain>/<type>:<name>).",
				MarkdownDescription: "❗ A list of API [resources](https://community.exoscale.com/documentation/iam/quick-start/#restricting-api-access-keys-to-resources) to restrict the key to (`<domain>/<type>:<name>`).",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					emptySetIsNull{},
					setRequiresReplace(),
				},
			},
			"secret": schema.StringAttribute{
				Description:         "The key secret.",
				MarkdownDescription: "The key secret.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"tags": schema.SetAttribute{
				Description:         "❗ A list of tags to restrict the key to.",
				MarkdownDescription: "❗ A list of tags to restrict the key to.",
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					emptySetIsNull{},
					setRequiresReplace(),
				},
			},
			"tags_operations": schema.SetAttribute{
				Description:         "The API operations granted by the tags.",
				MarkdownDescription: "The API operations granted by the `tags`.",
				ElementType:         types.StringType,
				Computed:            true,
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *ResourceAccessKey) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV2
	r.env = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).Environment
}

func (r *ResourceAccessKey) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceAccessKeyModel

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

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(r.env, config.DefaultZone))

	opts := []exoscale.CreateIAMAccessKeyOpt{}

	// Unknown when not configured.
	if !plan.Operations.IsUnknown() {
		var operations []string
		resp.Diagnostics.Append(plan.Operations.ElementsAs(ctx, &operations, false)...)
		if len(operations) > 0 {
			opts = append(opts, exoscale.CreateIAMAccessKeyWithOperations(operations))
		}
	}

	var resources []string
	resp.Diagnostics.Append(plan.Resources.ElementsAs(ctx, &resources, false)...)
	if len(resources) > 0 {
		parsedResources := make([]exoscale.IAMAccessKeyResource, len(resources))
		for i, resourceDescription := range resources {
			parsedResource, err := utils.ParseIAMAccessKeyResource(resourceDescription)
			if err != nil {
				resp.Diagnostics.AddAttributeError(
					path.Root("resources"),
					"invalid IAM access key resource",
					fmt.Sprintf("%q: %s, expected <domain>/<type>:<name>", resourceDescription, err),
				)
				return
			}
			parsedResources[i] = *parsedResource
		}

		opts = append(opts, exoscale.CreateIAMAccessKeyWithResources(parsedResources))
	}

	var tags []string
	resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
	if len(tags) > 0 {
		opts = append(opts, exoscale.CreateIAMAccessKeyWithTags(tags))
	}

	if resp.Diagnostics.HasError() {
		return
	}

	accessKey, err := r.client.CreateIAMAccessKey(ctx, config.DefaultZone, plan.Name.ValueString(), opts...)
	if err != nil {
		resp.Diagnostics.AddError("unable to create IAM access key", err.Error())
		return
	}

	plan.ID = types.StringPointerValue(accessKey.Key)
	plan.Key = types.StringPointerValue(accessKey.Key)
	// The secret is only returned by the creation call.
	plan.Secret = types.StringPointerValue(accessKey.Secret)

	operationsUnknown := plan.Operations.IsUnknown()
	if operationsUnknown {
		plan.Operations = types.SetNull(types.StringType)
	}
	plan.TagsOperations = types.SetNull(types.StringType)

	// Save the key right away: if a lookup below fails, the key (and its
	// secret) is kept in the state (tainted) instead of being orphaned.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessKey, err = r.client.GetIAMAccessKey(ctx, config.DefaultZone, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to get IAM access key", err.Error())
		return
	}

	knownOperations, err := r.client.ListIAMAccessKeyOperations(ctx, config.DefaultZone)
	if err != nil {
		resp.Diagnostics.AddError("unable to list IAM access key operations", err.Error())
		return
	}

	// Computed attributes only: the configured ones are kept as planned.
	if operationsUnknown {
		utils.RefreshStringSet(ctx, &resp.Diagnostics, &plan.Operations, stringsValue(accessKey.Operations))
	}
	utils.RefreshStringSet(ctx, &resp.Diagnostics, &plan.TagsOperations, tagsOperations(accessKey, knownOperations))
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Trace(ctx, "resource created", map[string]any{
		"id": plan.ID.ValueString(),
	})
}

func (r *ResourceAccessKey) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceAccessKeyModel

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

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(r.env, config.DefaultZone))

	accessKey, err := r.client.GetIAMAccessKey(ctx, config.DefaultZone, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("unable to get IAM access key", err.Error())
		return
	}

	knownOperations, err := r.client.ListIAMAccessKeyOperations(ctx, config.DefaultZone)
	if err != nil {
		resp.Diagnostics.AddError("unable to list IAM access key operations", err.Error())
		return
	}

	resp.Diagnostics.Append(applyAccessKey(ctx, &state, accessKey, knownOperations)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Trace(ctx, "resource read", map[string]any{
		"id": state.ID.ValueString(),
	})
}

// Update only saves the plan: every configurable attribute of an access key
// replaces it, this is reached for a timeouts change or a set going from
// null to empty (or back).
func (r *ResourceAccessKey) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceAccessKeyModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ResourceAccessKey) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceAccessKeyModel

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

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(r.env, config.DefaultZone))

	err := r.client.RevokeIAMAccessKey(ctx, config.DefaultZone, &exoscale.IAMAccessKey{
		Key: state.ID.ValueStringPointer(),
	})
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("unable to revoke IAM access key", err.Error())
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]any{
		"id": state.ID.ValueString(),
	})
}

// ImportState imports a key by its identifier. The secret cannot be retrieved
// and stays null.
func (r *ResourceAccessKey) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// applyAccessKey refreshes the whole model from the API (Read only).
func applyAccessKey(
	ctx context.Context,
	state *ResourceAccessKeyModel,
	accessKey *exoscale.IAMAccessKey,
	knownOperations []*exoscale.IAMAccessKeyOperation,
) diag.Diagnostics {
	var diags diag.Diagnostics

	utils.RefreshStringPointer(&state.Key, accessKey.Key)
	utils.RefreshStringPointer(&state.Name, accessKey.Name)
	utils.RefreshStringSet(ctx, &diags, &state.Operations, stringsValue(accessKey.Operations))

	resources := []string{}
	if accessKey.Resources != nil {
		for _, r := range *accessKey.Resources {
			resources = append(resources, fmt.Sprintf("%s/%s:%s", r.Domain, r.ResourceType, r.ResourceName))
		}
	}
	utils.RefreshStringSet(ctx, &diags, &state.Resources, resources)

	utils.RefreshStringSet(ctx, &diags, &state.Tags, stringsValue(accessKey.Tags))
	utils.RefreshStringSet(ctx, &diags, &state.TagsOperations, tagsOperations(accessKey, knownOperations))

	return diags
}

// tagsOperations returns the operations granted by the tags of the key.
func tagsOperations(accessKey *exoscale.IAMAccessKey, knownOperations []*exoscale.IAMAccessKeyOperation) []string {
	if accessKey.Tags == nil {
		return nil
	}

	operationsByTag := map[string][]string{}
	for _, operation := range knownOperations {
		for _, tag := range operation.Tags {
			operationsByTag[tag] = append(operationsByTag[tag], operation.Name)
		}
	}

	seen := map[string]bool{}
	operations := []string{}
	for _, tag := range *accessKey.Tags {
		for _, operation := range operationsByTag[tag] {
			if !seen[operation] {
				seen[operation] = true
				operations = append(operations, operation)
			}
		}
	}

	return operations
}

func stringsValue(v *[]string) []string {
	if v == nil {
		return nil
	}

	return *v
}
