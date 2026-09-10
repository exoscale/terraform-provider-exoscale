package template

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	exoscale "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const ResourceTemplateDescription = `Manage Exoscale [Compute Instance Custom Templates](https://community.exoscale.com/product/compute/instances/how-to/custom-templates/).

Registering a custom template that can be used as the initial setup of a Compute instance.

Corresponding data source: [exoscale_template](../data-sources/template.md).`

var _ resource.Resource = &ResourceTemplate{}
var _ resource.ResourceWithImportState = &ResourceTemplate{}

type ResourceTemplate struct {
	client *exoscale.Client
}

func NewResourceTemplate() resource.Resource {
	return &ResourceTemplate{}
}

type ResourceTemplateModel struct {
	ID          types.String `tfsdk:"id"`
	Zone        types.String `tfsdk:"zone"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	URL         types.String `tfsdk:"url"`
	Checksum    types.String `tfsdk:"checksum"`

	DefaultUser types.String `tfsdk:"default_user"`
	BootMode    types.String `tfsdk:"boot_mode"`
	Build       types.String `tfsdk:"build"`
	Maintainer  types.String `tfsdk:"maintainer"`
	Version     types.String `tfsdk:"version"`
	Size        types.Int64  `tfsdk:"size"`

	PasswordEnabled                      types.Bool `tfsdk:"password_enabled"`
	SSHKeyEnabled                        types.Bool `tfsdk:"ssh_key_enabled"`
	ApplicationConsistentSnapshotEnabled types.Bool `tfsdk:"application_consistent_snapshot_enabled"`

	Family     types.String `tfsdk:"family"`
	Visibility types.String `tfsdk:"visibility"`
	CreatedAt  types.String `tfsdk:"created_at"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

func (r *ResourceTemplate) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template"
}

func (r *ResourceTemplate) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: ResourceTemplateDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
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
			"name": schema.StringAttribute{
				MarkdownDescription: "The template name.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A free-form text describing the template.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "❗ The URL to download the template image (qcow2/raw disk image) from.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"checksum": schema.StringAttribute{
				MarkdownDescription: "❗ The MD5 checksum of the template image referenced by `url`.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"default_user": schema.StringAttribute{
				MarkdownDescription: "❗ Username used to log into a Compute instance based on this template.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"boot_mode": schema.StringAttribute{
				MarkdownDescription: "❗ The template boot mode: `legacy` (default) or `uefi`.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf(
						string(exoscale.RegisterTemplateRequestBootModeLegacy),
						string(exoscale.RegisterTemplateRequestBootModeUefi),
					),
				},
			},
			"build": schema.StringAttribute{
				MarkdownDescription: "❗ The template build.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"maintainer": schema.StringAttribute{
				MarkdownDescription: "❗ The template maintainer.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "❗ The template version.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "❗ The template size in GB. Defaults to the size of the downloaded image.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplaceIfConfigured(),
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"password_enabled": schema.BoolAttribute{
				MarkdownDescription: "❗ Whether to enable password-based login on Compute instances based on this template.",
				Required:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"ssh_key_enabled": schema.BoolAttribute{
				MarkdownDescription: "❗ Whether to enable SSH key-based login on Compute instances based on this template.",
				Required:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"application_consistent_snapshot_enabled": schema.BoolAttribute{
				MarkdownDescription: "❗ Whether the template supports application-consistent snapshots (requires the QEMU guest agent to be installed).",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplaceIfConfigured(),
				},
			},
			"family": schema.StringAttribute{
				MarkdownDescription: "The template family.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"visibility": schema.StringAttribute{
				MarkdownDescription: "The template visibility (custom templates are always `private`).",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"created_at": schema.StringAttribute{
				MarkdownDescription: "The template creation date.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Read:   true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (r *ResourceTemplate) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func apply(model *ResourceTemplateModel, tpl *exoscale.Template) {
	model.ID = types.StringValue(tpl.ID.String())
	model.Name = types.StringValue(tpl.Name)
	model.Description = types.StringValue(tpl.Description)
	model.URL = types.StringValue(tpl.URL)
	model.Checksum = types.StringValue(tpl.Checksum)
	model.DefaultUser = types.StringValue(tpl.DefaultUser)
	model.BootMode = types.StringValue(string(tpl.BootMode))
	model.Build = types.StringValue(tpl.Build)
	model.Maintainer = types.StringValue(tpl.Maintainer)
	model.Version = types.StringValue(tpl.Version)
	model.Size = types.Int64Value(tpl.Size)
	model.Family = types.StringValue(tpl.Family)
	model.Visibility = types.StringValue(string(tpl.Visibility))
	model.CreatedAt = types.StringValue(tpl.CreatedAT.String())

	model.PasswordEnabled = types.BoolValue(tpl.PasswordEnabled != nil && *tpl.PasswordEnabled)
	model.SSHKeyEnabled = types.BoolValue(tpl.SSHKeyEnabled != nil && *tpl.SSHKeyEnabled)
	model.ApplicationConsistentSnapshotEnabled = types.BoolValue(
		tpl.ApplicationConsistentSnapshotEnabled != nil && *tpl.ApplicationConsistentSnapshotEnabled,
	)
}

func (r *ResourceTemplate) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ResourceTemplateModel

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

	passwordEnabled := plan.PasswordEnabled.ValueBool()
	sshKeyEnabled := plan.SSHKeyEnabled.ValueBool()

	request := exoscale.RegisterTemplateRequest{
		Name:            plan.Name.ValueString(),
		URL:             plan.URL.ValueString(),
		Checksum:        plan.Checksum.ValueString(),
		Description:     plan.Description.ValueString(),
		DefaultUser:     plan.DefaultUser.ValueString(),
		Build:           plan.Build.ValueString(),
		Maintainer:      plan.Maintainer.ValueString(),
		Version:         plan.Version.ValueString(),
		PasswordEnabled: &passwordEnabled,
		SSHKeyEnabled:   &sshKeyEnabled,
	}

	if !plan.BootMode.IsNull() && !plan.BootMode.IsUnknown() {
		request.BootMode = exoscale.RegisterTemplateRequestBootMode(plan.BootMode.ValueString())
	}

	if !plan.Size.IsNull() && !plan.Size.IsUnknown() {
		request.Size = plan.Size.ValueInt64()
	}

	if !plan.ApplicationConsistentSnapshotEnabled.IsNull() && !plan.ApplicationConsistentSnapshotEnabled.IsUnknown() {
		v := plan.ApplicationConsistentSnapshotEnabled.ValueBool()
		request.ApplicationConsistentSnapshotEnabled = &v
	}

	op, err := client.RegisterTemplate(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("unable to register template", err.Error())
		return
	}

	op, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("failed to register template", err.Error())
		return
	}

	if op.Reference == nil {
		resp.Diagnostics.AddError("failed to register template", "operation completed without a resource reference")
		return
	}

	tpl, err := client.GetTemplate(ctx, op.Reference.ID)
	if err != nil {
		resp.Diagnostics.AddError("unable to get template", err.Error())
		return
	}

	apply(&plan, tpl)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Trace(ctx, "resource created", map[string]any{
		"id": plan.ID,
	})
}

func (r *ResourceTemplate) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ResourceTemplateModel

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
		resp.Diagnostics.AddError("unable to parse template ID", err.Error())
		return
	}

	tpl, err := client.GetTemplate(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("unable to get template", err.Error())
		return
	}

	apply(&state, tpl)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	tflog.Trace(ctx, "resource read done", map[string]any{
		"id": state.ID,
	})
}

func (r *ResourceTemplate) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state, plan ResourceTemplateModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
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

	client, err := utils.SwitchClientZone(ctx, r.client, exoscale.ZoneName(state.Zone.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("unable to change exoscale client zone", err.Error())
		return
	}

	id, err := exoscale.ParseUUID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("unable to parse template ID", err.Error())
		return
	}

	if !plan.Name.Equal(state.Name) || !plan.Description.Equal(state.Description) {
		op, err := client.UpdateTemplate(ctx, id, exoscale.UpdateTemplateRequest{
			Name:        plan.Name.ValueString(),
			Description: plan.Description.ValueString(),
		})
		if err != nil {
			resp.Diagnostics.AddError("unable to update template", err.Error())
			return
		}

		_, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
		if err != nil {
			resp.Diagnostics.AddError("failed to update template", err.Error())
			return
		}
	}

	tpl, err := client.GetTemplate(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("unable to get template", err.Error())
		return
	}

	apply(&plan, tpl)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)

	tflog.Trace(ctx, "resource update done", map[string]any{
		"id": state.ID,
	})
}

func (r *ResourceTemplate) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ResourceTemplateModel

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
		resp.Diagnostics.AddError("unable to parse template ID", err.Error())
		return
	}

	op, err := client.DeleteTemplate(ctx, id)
	if err != nil {
		if errors.Is(err, exoscale.ErrNotFound) {
			return
		}
		resp.Diagnostics.AddError("unable to delete template", err.Error())
		return
	}

	_, err = client.Wait(ctx, op, exoscale.OperationStateSuccess)
	if err != nil {
		resp.Diagnostics.AddError("failed to delete template", err.Error())
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]any{
		"id": state.ID,
	})
}

func (r *ResourceTemplate) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, "@")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"unexpected import identifier",
			fmt.Sprintf("Expected import identifier with format: id@zone. Got: %q", req.ID),
		)
		return
	}

	var t timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &t)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &ResourceTemplateModel{
		ID:       types.StringValue(idParts[0]),
		Zone:     types.StringValue(idParts[1]),
		Timeouts: t,
	})...)

	tflog.Trace(ctx, "resource imported", map[string]any{
		"id": idParts[0],
	})
}
