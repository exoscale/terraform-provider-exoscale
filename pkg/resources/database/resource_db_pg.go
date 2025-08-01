package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type PGDatabaseResource struct {
	DBResource
}

type PGDatabaseResourceModel struct {
	DBResourceModel
	LcCtype   types.String `tfsdk:"lc_ctype"`
	LcCollate types.String `tfsdk:"lc_collate"`
}

var _ resource.Resource = &PGDatabaseResource{}
var _ resource.ResourceWithImportState = &PGDatabaseResource{}

func NewPGDatabaseResource() resource.Resource {
	return &PGDatabaseResource{}
}

func (r *PGDatabaseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

// ImportState implements resource.ResourceWithImportState.
func (p *PGDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	idParts := strings.Split(req.ID, "@")

	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {

		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: service/database_name@zone. Got: %q", req.ID),
		)

		return
	}

	databaseID := idParts[0]
	zone := idParts[1]

	id := strings.Split(databaseID, "/")

	if len(id) != 2 || id[0] == "" || id[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: service/database_name@zone. Got: %q", req.ID),
		)
	}

	serviceName := id[0]
	databaseName := id[1]

	var data PGDatabaseResourceModel

	// Set timeouts (quirk https://github.com/hashicorp/terraform-plugin-framework-timeouts/issues/46)
	var timeouts timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &timeouts)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Timeouts = timeouts

	data.Id = types.StringValue(databaseID)
	data.DatabaseName = types.StringValue(databaseName)
	data.Service = types.StringValue(serviceName)
	data.Zone = types.StringValue(zone)

	ReadResourceForImport(ctx, req, resp, &data, p.client)
}

// Create implements resource.Resource.
func (p *PGDatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PGDatabaseResourceModel
	CreateResource(ctx, req, resp, &data, p.client)
}

// Delete implements resource.Resource.
func (p *PGDatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PGDatabaseResourceModel
	DeleteResource(ctx, req, resp, &data, p.client)
}

// Metadata implements resource.Resource.
func (p *PGDatabaseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_pg_database"
}

// Read implements resource.Resource.
func (p *PGDatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PGDatabaseResourceModel
	ReadResource(ctx, req, resp, &data, p.client)

}

// Schema implements resource.Resource.
func (p *PGDatabaseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{

		MarkdownDescription: "❗ Manage service database for a PostgreSQL Exoscale [Database Services (DBaaS)](https://community.exoscale.com/documentation/dbaas/).",
		Attributes: map[string]schema.Attribute{
			// Computed attributes
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource, computed as service/database_name",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
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
			"database_name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "❗ The name of the database for this service.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"lc_collate": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Default string sort order (LC_COLLATE) for PostgreSQL database",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"lc_ctype": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Default character classification (LC_CTYPE) for PostgreSQL database",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

// Update implements resource.Resource.
func (p *PGDatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var stateData, planData PGDatabaseResourceModel
	UpdateResource(ctx, req, resp, &stateData, &planData, p.client)
}

// ReadResource reads resource from remote and populate the model accordingly
func (data PGDatabaseResourceModel) ReadResource(ctx context.Context, client *v3.Client, diagnostics *diag.Diagnostics) {
	svc, err := waitForDBAASServiceReadyForFn(ctx, client.GetDBAASServicePG, data.Service.ValueString(), func(t *v3.DBAASServicePG) bool { return t.State == v3.EnumServiceStateRunning })
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service pg, got error: %s", err))
		return
	}

	for _, db := range svc.Databases {
		if string(db) == data.DatabaseName.ValueString() {
			return
		}
	}
	diagnostics.AddError("Client Error", "Unable to find database for the service")
}

// CreateResource creates the resource according to the model, and then
// update computed fields if applicable
func (data PGDatabaseResourceModel) CreateResource(ctx context.Context, client *v3.Client, diagnostics *diag.Diagnostics) {

	createRequest := v3.CreateDBAASPGDatabaseRequest{
		DatabaseName: v3.DBAASDatabaseName(data.DatabaseName.ValueString()),
	}

	if !data.LcCollate.IsNull() {
		createRequest.LCCollate = data.LcCtype.ValueString()
	}
	if !data.LcCtype.IsNull() {
		createRequest.LCCollate = data.LcCollate.ValueString()
	}

	op, err := client.CreateDBAASPGDatabase(ctx, data.Service.ValueString(), createRequest)
	if err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create service database, got error %s", err.Error()),
		)
		return
	}

	if _, err := client.Wait(ctx, op, v3.OperationStateSuccess); err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create service database, got error %s", err.Error()),
		)
		return
	}

	svc, err := client.GetDBAASServicePG(ctx, data.Service.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service pg, got error: %s", err))
		return
	}

	for _, db := range svc.Databases {
		if string(db) == data.DatabaseName.ValueString() {
			return
		}
	}
	diagnostics.AddError("Client Error", "Unable to find newly created database for the service")
}

// DeleteResource deletes the resource
func (data PGDatabaseResourceModel) DeleteResource(ctx context.Context, client *v3.Client, diagnostics *diag.Diagnostics) {

	op, err := client.DeleteDBAASPGDatabase(ctx, data.Service.ValueString(), data.DatabaseName.ValueString())
	if err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete service database, got error %s", err.Error()),
		)
		return
	}
	_, err = client.Wait(ctx, op, v3.OperationStateSuccess)
	if err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete service database, got error %s", err.Error()),
		)
		return
	}

}

// UpdateResource updates the remote resource w/ the new model
func (PGDatabaseResourceModel) UpdateResource(ctx context.Context, client *v3.Client, diagnostics *diag.Diagnostics) {
	// Nothing to do as these resources are systematically recreated
}

func (data PGDatabaseResourceModel) WaitForService(ctx context.Context, client *v3.Client, diagnostics *diag.Diagnostics) {
	_, err := waitForDBAASServiceReadyForFn(ctx, client.GetDBAASServicePG, data.Service.ValueString(), func(t *v3.DBAASServicePG) bool { return t.State == v3.EnumServiceStateRunning })

	time.Sleep(SERVICE_READY_DELAY)

	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Database service PG %s", err.Error()))
	}
}

// Accessing and setting attributes
func (data *PGDatabaseResourceModel) GetTimeouts() timeouts.Value {
	return data.Timeouts
}

func (data *PGDatabaseResourceModel) SetTimeouts(t timeouts.Value) {
	data.Timeouts = t
}

func (data *PGDatabaseResourceModel) GetID() basetypes.StringValue {
	return data.Id
}

func (data *PGDatabaseResourceModel) GetZone() basetypes.StringValue {
	return data.Zone
}

// Should set the return value of .GetID() to service/database_name
func (data *PGDatabaseResourceModel) GenerateID() {
	data.Id = basetypes.NewStringValue(fmt.Sprintf("%s/%s", data.Service.ValueString(), data.DatabaseName.ValueString()))
}
