package database

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type PGConnectionPoolResource struct {
	client *v3.Client
}

type PGConnectionPoolResourceModel struct {
	Id            types.String `tfsdk:"id"`
	Service       types.String `tfsdk:"service"`
	Name          types.String `tfsdk:"name"`
	DatabaseName  types.String `tfsdk:"database_name"`
	Username      types.String `tfsdk:"username"`
	Mode          types.String `tfsdk:"mode"`
	Size          types.Int64  `tfsdk:"size"`
	ConnectionURI types.String `tfsdk:"connection_uri"`
	Zone          types.String `tfsdk:"zone"`

	Timeouts timeouts.Value `tfsdk:"timeouts"`

	refreshUsername bool
	refreshMode     bool
	refreshSize     bool
}

var _ resource.Resource = &PGConnectionPoolResource{}
var _ resource.ResourceWithImportState = &PGConnectionPoolResource{}

func NewPGConnectionPoolResource() resource.Resource {
	return &PGConnectionPoolResource{}
}

func (r *PGConnectionPoolResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(*providerConfig.ExoscaleProviderConfig).ClientV3
}

func (r *PGConnectionPoolResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dbaas_pg_connection_pool"
}

func (r *PGConnectionPoolResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manage PostgreSQL PgBouncer connection pools for an Exoscale [Database Services (DBaaS)](https://community.exoscale.com/documentation/dbaas/).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "The ID of this resource, computed as `service/name`.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service": schema.StringAttribute{
				MarkdownDescription: "❗ The name of the PostgreSQL database service.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "❗ The connection pool name.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"database_name": schema.StringAttribute{
				MarkdownDescription: "❗ The PostgreSQL database name targeted by this pool.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "❗ The PostgreSQL username used by this pool.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplaceIfConfigured(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"mode": schema.StringAttribute{
				MarkdownDescription: "The PgBouncer pool mode (`transaction`, `statement`, or `session`).",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
				Validators: []validator.String{
					stringvalidator.OneOf("transaction", "statement", "session"),
				},
			},
			"size": schema.Int64Attribute{
				MarkdownDescription: "The connection pool size.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
				Validators: []validator.Int64{
					int64validator.Between(1, 10000),
				},
			},
			"connection_uri": schema.StringAttribute{
				MarkdownDescription: "The connection URI for this pool.",
				Computed:            true,
				Sensitive:           true,
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
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.BlockAll(ctx),
		},
	}
}

func (r *PGConnectionPoolResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PGConnectionPoolResourceModel
	ReadResource(ctx, req, resp, &data, r.client)
}

func (r *PGConnectionPoolResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PGConnectionPoolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.configurePostApplyRefresh(ctx, req.Config, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	t, diags := data.GetTimeouts().Create(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	data.GenerateID()

	client, err := utils.SwitchClientZone(ctx, r.client, v3.ZoneName(data.GetZone().ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to change exoscale client zone",
			err.Error(),
		)
		return
	}

	data.WaitForService(ctx, client, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	data.CreateResource(ctx, client, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	tflog.Trace(ctx, "resource created", map[string]interface{}{
		"id": data.GetID(),
	})
}

func (r *PGConnectionPoolResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var stateData, planData PGConnectionPoolResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}
	planData.configurePostApplyRefresh(ctx, req.Config, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	t, diags := stateData.GetTimeouts().Update(ctx, config.DefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, t)
	defer cancel()

	client, err := utils.SwitchClientZone(ctx, r.client, v3.ZoneName(planData.GetZone().ValueString()))
	if err != nil {
		resp.Diagnostics.AddError(
			"unable to change exoscale client zone",
			err.Error(),
		)
		return
	}

	planData.WaitForService(ctx, client, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	planData.UpdateResource(ctx, client, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)

	tflog.Trace(ctx, "resource updated", map[string]interface{}{
		"id": planData.GetID(),
	})
}

func (r *PGConnectionPoolResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PGConnectionPoolResourceModel
	DeleteResource(ctx, req, resp, &data, r.client)
}

func (r *PGConnectionPoolResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)

	idParts := strings.Split(req.ID, "@")
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: service/pool_name@zone. Got: %q", req.ID),
		)
		return
	}

	poolID := idParts[0]
	zone := idParts[1]
	id := strings.Split(poolID, "/")
	if len(id) != 2 || id[0] == "" || id[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: service/pool_name@zone. Got: %q", req.ID),
		)
		return
	}

	var data PGConnectionPoolResourceModel
	var resourceTimeouts timeouts.Value
	resp.Diagnostics.Append(resp.State.GetAttribute(ctx, path.Root("timeouts"), &resourceTimeouts)...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Timeouts = resourceTimeouts
	data.Id = types.StringValue(poolID)
	data.Service = types.StringValue(id[0])
	data.Name = types.StringValue(id[1])
	data.Zone = types.StringValue(zone)

	ReadResourceForImport(ctx, req, resp, &data, r.client)
}

func (data *PGConnectionPoolResourceModel) ReadResource(ctx context.Context, client *v3.Client, diagnostics *diag.Diagnostics) (clearState bool) {
	pool, clearState := data.getConnectionPool(ctx, client, diagnostics)
	if clearState || pool == nil {
		return clearState
	}

	data.applyConnectionPoolState(*pool)

	return false
}

func (data *PGConnectionPoolResourceModel) getConnectionPool(ctx context.Context, client *v3.Client, diagnostics *diag.Diagnostics) (*v3.DBAASServicePGConnectionPools, bool) {
	svc, err := waitForDBAASServiceReadyForFn(ctx, client.GetDBAASServicePG, data.Service.ValueString(), func(t *v3.DBAASServicePG) bool {
		return t.State == v3.EnumServiceStateRunning
	})
	if err != nil {
		if errors.Is(err, v3.ErrNotFound) {
			return nil, true
		}
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read service pg connection pool, got error: %s", err))
		return nil, false
	}

	for _, pool := range svc.ConnectionPools {
		if string(pool.Name) != data.Name.ValueString() {
			continue
		}

		pool := pool
		return &pool, false
	}

	return nil, true
}

func (data *PGConnectionPoolResourceModel) applyConnectionPoolState(pool v3.DBAASServicePGConnectionPools) {
	data.DatabaseName = basetypes.NewStringValue(string(pool.Database))
	data.Mode = basetypes.NewStringValue(string(pool.Mode))
	data.Size = basetypes.NewInt64Value(int64(pool.Size))
	data.Username = basetypes.NewStringValue(string(pool.Username))
	data.ConnectionURI = basetypes.NewStringValue(pool.ConnectionURI)
}

func (data *PGConnectionPoolResourceModel) applyPostApplyConnectionPoolState(pool v3.DBAASServicePGConnectionPools) {
	if data.refreshMode {
		data.Mode = basetypes.NewStringValue(string(pool.Mode))
	}
	if data.refreshSize {
		data.Size = basetypes.NewInt64Value(int64(pool.Size))
	}
	if data.refreshUsername {
		data.Username = basetypes.NewStringValue(string(pool.Username))
	}
	data.ConnectionURI = basetypes.NewStringValue(pool.ConnectionURI)
}

func (data *PGConnectionPoolResourceModel) configurePostApplyRefresh(ctx context.Context, config tfsdk.Config, diagnostics *diag.Diagnostics) {
	var usernameConfig types.String
	var modeConfig types.String
	var sizeConfig types.Int64

	diagnostics.Append(config.GetAttribute(ctx, path.Root("username"), &usernameConfig)...)
	diagnostics.Append(config.GetAttribute(ctx, path.Root("mode"), &modeConfig)...)
	diagnostics.Append(config.GetAttribute(ctx, path.Root("size"), &sizeConfig)...)
	if diagnostics.HasError() {
		return
	}

	data.configurePostApplyRefreshFromConfig(
		!usernameConfig.IsNull(),
		!modeConfig.IsNull(),
		!sizeConfig.IsNull(),
	)
}

func (data *PGConnectionPoolResourceModel) configurePostApplyRefreshFromConfig(usernameConfigured, modeConfigured, sizeConfigured bool) {
	data.refreshUsername = !usernameConfigured && (data.Username.IsNull() || data.Username.IsUnknown())
	data.refreshMode = !modeConfigured && (data.Mode.IsNull() || data.Mode.IsUnknown())
	data.refreshSize = !sizeConfigured && (data.Size.IsNull() || data.Size.IsUnknown())
}

func (data *PGConnectionPoolResourceModel) refreshPostApplyState(ctx context.Context, client *v3.Client, diagnostics *diag.Diagnostics) (clearState bool) {
	pool, clearState := data.getConnectionPool(ctx, client, diagnostics)
	if clearState || pool == nil {
		return clearState
	}

	data.applyPostApplyConnectionPoolState(*pool)

	return false
}

func (data *PGConnectionPoolResourceModel) CreateResource(ctx context.Context, client *v3.Client, diagnostics *diag.Diagnostics) {
	createRequest := v3.CreateDBAASPGConnectionPoolRequest{
		DatabaseName: v3.DBAASDatabaseName(data.DatabaseName.ValueString()),
		Name:         v3.DBAASPGPoolName(data.Name.ValueString()),
	}

	if !data.Mode.IsNull() && !data.Mode.IsUnknown() {
		createRequest.Mode = v3.EnumPGPoolMode(data.Mode.ValueString())
	}

	if !data.Size.IsNull() && !data.Size.IsUnknown() {
		createRequest.Size = v3.DBAASPGPoolSize(data.Size.ValueInt64())
	}

	if !data.Username.IsNull() && !data.Username.IsUnknown() {
		createRequest.Username = v3.DBAASPGPoolUsername(data.Username.ValueString())
	}

	op, err := client.CreateDBAASPGConnectionPool(ctx, data.Service.ValueString(), createRequest)
	if err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create service connection pool, got error %s", err.Error()),
		)
		return
	}

	if _, err := client.Wait(ctx, op, v3.OperationStateSuccess); err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create service connection pool, got error %s", err.Error()),
		)
		return
	}

	if data.refreshPostApplyState(ctx, client, diagnostics) {
		diagnostics.AddError("Client Error", "Unable to find newly created connection pool for the service")
	}
}

func (data *PGConnectionPoolResourceModel) UpdateResource(ctx context.Context, client *v3.Client, diagnostics *diag.Diagnostics) {
	updateRequest := v3.UpdateDBAASPGConnectionPoolRequest{
		DatabaseName: v3.DBAASDatabaseName(data.DatabaseName.ValueString()),
	}

	if !data.Mode.IsNull() && !data.Mode.IsUnknown() {
		updateRequest.Mode = v3.EnumPGPoolMode(data.Mode.ValueString())
	}

	if !data.Size.IsNull() && !data.Size.IsUnknown() {
		updateRequest.Size = v3.DBAASPGPoolSize(data.Size.ValueInt64())
	}

	if !data.Username.IsNull() && !data.Username.IsUnknown() {
		updateRequest.Username = v3.DBAASPGPoolUsername(data.Username.ValueString())
	}

	op, err := client.UpdateDBAASPGConnectionPool(ctx, data.Service.ValueString(), data.Name.ValueString(), updateRequest)
	if err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to update service connection pool, got error %s", err.Error()),
		)
		return
	}

	if _, err := client.Wait(ctx, op, v3.OperationStateSuccess); err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to update service connection pool, got error %s", err.Error()),
		)
		return
	}

	if data.refreshPostApplyState(ctx, client, diagnostics) {
		diagnostics.AddError("Client Error", "Unable to find updated connection pool for the service")
	}
}

func (data *PGConnectionPoolResourceModel) DeleteResource(ctx context.Context, client *v3.Client, diagnostics *diag.Diagnostics) {
	op, err := client.DeleteDBAASPGConnectionPool(ctx, data.Service.ValueString(), data.Name.ValueString())
	if err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete service connection pool, got error %s", err.Error()),
		)
		return
	}

	if _, err := client.Wait(ctx, op, v3.OperationStateSuccess); err != nil {
		diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete service connection pool, got error %s", err.Error()),
		)
		return
	}
}

func (data *PGConnectionPoolResourceModel) WaitForService(ctx context.Context, client *v3.Client, diagnostics *diag.Diagnostics) {
	_, err := waitForDBAASServiceReadyForFn(ctx, client.GetDBAASServicePG, data.Service.ValueString(), func(t *v3.DBAASServicePG) bool {
		return t.State == v3.EnumServiceStateRunning
	})

	time.Sleep(SERVICE_READY_DELAY)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Database service PG %s", err.Error()))
	}
}

func (data *PGConnectionPoolResourceModel) GetTimeouts() timeouts.Value {
	return data.Timeouts
}

func (data *PGConnectionPoolResourceModel) SetTimeouts(t timeouts.Value) {
	data.Timeouts = t
}

func (data *PGConnectionPoolResourceModel) GetID() basetypes.StringValue {
	return data.Id
}

func (data *PGConnectionPoolResourceModel) GetZone() basetypes.StringValue {
	return data.Zone
}

func (data *PGConnectionPoolResourceModel) GenerateID() {
	data.Id = basetypes.NewStringValue(fmt.Sprintf("%s/%s", data.Service.ValueString(), data.Name.ValueString()))
}
