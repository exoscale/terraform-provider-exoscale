package ssh_key

import (
	"context"
	"errors"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const markdownDescriptionResource = `Manage Exoscale [SSH Keys](https://community.exoscale.com/product/compute/instances/how-to/ssh-keypairs/).`

var _ resource.ResourceWithConfigure = (*Resource)(nil)
var _ resource.ResourceWithImportState = (*Resource)(nil)

type Resource struct {
	client *exoscale.Client
}

func NewResource() resource.Resource {
	return &Resource{}
}

type ResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	PublicKey   types.String `tfsdk:"public_key"`
	Fingerprint types.String `tfsdk:"fingerprint"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *Resource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_key"
}

func (r *Resource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manage Exoscale SSH Keys.",
		MarkdownDescription: markdownDescriptionResource,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description:         "The SSH key name.",
				MarkdownDescription: "The SSH key name.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description:         "❗ The SSH key name.",
				MarkdownDescription: "❗ The SSH key name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public_key": schema.StringAttribute{
				Description:         "❗ The SSH public key that will be authorized in compute instances.",
				MarkdownDescription: "❗ The SSH *public* key that will be authorized in compute instances.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"fingerprint": schema.StringAttribute{
				Description:         "The SSH key unique identifier.",
				MarkdownDescription: "The SSH key unique identifier.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *Resource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *Resource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceModel

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

	op, err := r.client.RegisterSSHKey(ctx, exoscale.RegisterSSHKeyRequest{
		Name:      plan.Name.ValueString(),
		PublicKey: plan.PublicKey.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("unable to register SSH key", err.Error())
		return
	}

	if _, err := r.client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("register SSH key operation failed", err.Error())
		return
	}

	// SSH keys are identified by their name.
	plan.ID = plan.Name
	plan.Fingerprint = types.StringNull()

	// Save the key right away: if the fingerprint lookup below fails, the
	// key is kept in the state (tainted) instead of being orphaned.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sshKey, err := r.client.GetSSHKey(ctx, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to get SSH key", err.Error())
		return
	}

	plan.Fingerprint = types.StringValue(sshKey.Fingerprint)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Trace(ctx, "resource created", map[string]any{
		"id": plan.ID.ValueString(),
	})
}

func (r *Resource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceModel

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

	sshKey, err := r.client.GetSSHKey(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("unable to get SSH key", err.Error())
		return
	}

	// The API doesn't return the public key: it is kept as configured
	// (and stays null after an import).
	utils.RefreshString(&state.Name, sshKey.Name)
	utils.RefreshString(&state.Fingerprint, sshKey.Fingerprint)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Trace(ctx, "resource read", map[string]any{
		"id": state.ID.ValueString(),
	})
}

// Update only saves the plan: every configurable attribute of an SSH key
// replaces it, this is reached only for a timeouts change.
func (r *Resource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *Resource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceModel

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

	op, err := r.client.DeleteSSHKey(ctx, state.ID.ValueString())
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("unable to delete SSH key", err.Error())
		return
	}

	if _, err := r.client.Wait(ctx, op, exoscale.OperationStateSuccess); err != nil {
		resp.Diagnostics.AddError("delete SSH key operation failed", err.Error())
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]any{
		"id": state.ID.ValueString(),
	})
}

func (r *Resource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
