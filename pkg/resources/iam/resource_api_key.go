package iam

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	exoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
)

const ResourceAPIKeyDescription = `Manage Exoscale [IAM](https://community.exoscale.com/documentation/iam/) API Key.
`

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceAPIKey{}
var _ resource.ResourceWithImportState = &ResourceAPIKey{}

func NewResourceAPIKey() resource.Resource {
	return &ResourceAPIKey{}
}

// ResourceAPIKey defines the IAM Organization Policy resource implementation.
type ResourceAPIKey struct {
	client *exoscale.Client
	env    string
}

// ResourceAPIKeyModel describes the IAM Organization Policy resource data model.
type ResourceAPIKeyModel struct {
	ID     types.String `tfsdk:"id"`
	Key    types.String `tfsdk:"key"`
	Name   types.String `tfsdk:"name"`
	Secret types.String `tfsdk:"secret"`

	RoleID types.String `tfsdk:"role_id"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *ResourceAPIKey) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_iam_api_key"
}

func (r *ResourceAPIKey) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: ResourceAPIKeyDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "IAM API Key name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"role_id": schema.StringAttribute{
				MarkdownDescription: "IAM API Key role ID.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "The IAM API Key to match.",
				Computed:            true,
			},
			"secret": schema.StringAttribute{
				MarkdownDescription: "Secret for the IAM API Key.",
				Computed:            true,
				Sensitive:           true,
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Read: true,
			}),
		},
	}
}

func (r *ResourceAPIKey) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV2
	r.env = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).Environment
}

func (r *ResourceAPIKey) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ResourceAPIKeyModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout
	t, diags := data.Timeouts.Create(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(r.env, config.DefaultZone))

	key := exoscale.APIKey{
		Name:   data.Name.ValueStringPointer(),
		RoleID: data.RoleID.ValueStringPointer(),
	}

	newKey, secret, err := r.client.CreateAPIKey(ctx, config.DefaultZone, &key)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to create IAM API Key",
			err.Error(),
		)
		return
	}

	data.ID = types.StringPointerValue(newKey.Key)
	data.Secret = types.StringValue(secret)

	// Read created policy
	r.read(ctx, resp.Diagnostics, &data)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "resource created", map[string]interface{}{
		"id": data.ID,
	})
}

func (r *ResourceAPIKey) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceAPIKeyModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout
	t, diags := data.Timeouts.Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(r.env, config.DefaultZone))

	r.read(ctx, resp.Diagnostics, &data)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "resource read done", map[string]interface{}{
		"id": data.ID,
	})
}

// Update is NOOP becauses all arguments require restart..
func (r *ResourceAPIKey) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *ResourceAPIKey) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceAPIKeyModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout
	t, diags := data.Timeouts.Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(r.env, config.DefaultZone))

	err := r.client.DeleteAPIKey(
		ctx,
		config.DefaultZone,
		&exoscale.APIKey{Key: data.ID.ValueStringPointer()},
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to delete API Key",
			err.Error(),
		)
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]interface{}{
		"id": data.ID,
	})
}

func (r *ResourceAPIKey) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data ResourceAPIKeyModel

	// Set timeouts (quirk https://github.com/hashicorp/terraform-plugin-framework-timeouts/issues/46)
	var timeouts timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &timeouts)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Timeouts = timeouts

	data.ID = types.StringValue(req.ID)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "resource imported", map[string]interface{}{
		"id": data.ID,
	})
}

func (r *ResourceAPIKey) read(
	ctx context.Context,
	d diag.Diagnostics,
	data *ResourceAPIKeyModel,
) {
	apiKey, err := r.client.GetAPIKey(
		ctx,
		config.DefaultZone,
		data.ID.ValueString(),
	)
	if err != nil {
		d.AddError(
			"Unable to get IAM Role",
			err.Error(),
		)
		return
	}

	data.Key = types.StringPointerValue(apiKey.Key)
	data.Name = types.StringPointerValue(apiKey.Name)
	data.RoleID = types.StringPointerValue(apiKey.RoleID)
}
