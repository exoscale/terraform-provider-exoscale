package database

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

// UserResource defines the resource implementation.
type UserResource struct {
	client *exoscale.Client
}

// UserResourceModel describes the resource data model.
type UserResourceModel struct {
	Id       types.String `tfsdk:"id"`
	Service  types.String `tfsdk:"service"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Zone     types.String `tfsdk:"zone"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`
}

var commonAttributes = map[string]schema.Attribute{
	// Attributes referencing the service
	"service": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "❗ The name of the database service.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
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

	// Variables
	"username": schema.StringAttribute{
		Required:            true,
		MarkdownDescription: "❗ The name of the user for this service.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	},

	// Computed attributes
	"id": schema.StringAttribute{
		MarkdownDescription: "The ID of this resource, as SERVICENAME/USERNAME",
		Computed:            true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	},
	"password": schema.StringAttribute{
		Description: "The password of the service user.",
		Computed:    true,
		Sensitive:   true,
	},
}

func buildUserAttributes(newAttributes map[string]schema.Attribute) map[string]schema.Attribute {

	newSchemas := map[string]schema.Attribute{}
	for k, v := range commonAttributes {
		newSchemas[k] = v
	}
	for k, v := range newAttributes {
		newSchemas[k] = v
	}

	return newSchemas

}

type DatabaseServiceUserModel interface {
	Read(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics)
	Create(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics)
	Delete(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics)
	Update(ctx context.Context, client *exoscale.Client, diagnostics *diag.Diagnostics)
	GetTimeouts() timeouts.Value
	SetTimeouts(timeouts.Value)
	GenerateID()
	GetID() basetypes.StringValue
	GetZone() basetypes.StringValue
}

func UserRead[T DatabaseServiceUserModel](ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse, data T, client *exoscale.Client) {

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout
	t, diags := data.GetTimeouts().Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	data.GenerateID()

	client, err := utils.SwitchClientZone(
		ctx,
		client,
		exoscale.ZoneName(data.GetZone().ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to change exoscale client zone",
			err.Error(),
		)
		return
	}

	data.Read(ctx, client, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "resource read done", map[string]interface{}{
		"id": data.GetID(),
	})

}

func UserReadForImport[T DatabaseServiceUserModel](ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse, data T, client *exoscale.Client) {

	// Set timeout
	t, diags := data.GetTimeouts().Read(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	data.GenerateID()

	client, err := utils.SwitchClientZone(
		ctx,
		client,
		exoscale.ZoneName(data.GetZone().ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to change exoscale client zone",
			err.Error(),
		)
		return
	}

	data.Read(ctx, client, &resp.Diagnostics)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "resource read done", map[string]interface{}{
		"id": data.GetID(),
	})

}

func UserCreate[T DatabaseServiceUserModel](ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse, data T, client *exoscale.Client) {

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout
	t, diags := data.GetTimeouts().Create(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	data.GenerateID()

	client, err := utils.SwitchClientZone(
		ctx,
		client,
		exoscale.ZoneName(data.GetZone().ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to change exoscale client zone",
			err.Error(),
		)
		return
	}

	data.Create(ctx, client, &diags)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "resource created", map[string]interface{}{
		"id": data.GetID(),
	})

}

func UserUpdate[T DatabaseServiceUserModel](ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse, stateData, planData T, client *exoscale.Client) {
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	// Read Terraform state data (for comparison) into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout
	t, diags := stateData.GetTimeouts().Update(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	client, err := utils.SwitchClientZone(
		ctx,
		client,
		exoscale.ZoneName(planData.GetZone().ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to change exoscale client zone",
			err.Error(),
		)
		return
	}

	planData.Update(ctx, client, &diags)

	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)

	tflog.Trace(ctx, "resource updated", map[string]interface{}{
		"id": planData.GetID(),
	})
}

func UserDelete[T DatabaseServiceUserModel](ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse, data T, client *exoscale.Client) {
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Set timeout
	t, diags := data.GetTimeouts().Delete(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	data.GenerateID()

	client, err := utils.SwitchClientZone(
		ctx,
		client,
		exoscale.ZoneName(data.GetZone().ValueString()),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to change exoscale client zone",
			err.Error(),
		)
		return
	}

	data.Delete(ctx, client, &diags)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "resource deleted", map[string]interface{}{
		"id": data.GetID(),
	})

}
