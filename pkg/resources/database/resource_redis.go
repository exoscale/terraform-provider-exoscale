package database

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/validators"
)

type ResourceRedisModel struct {
	IpFilter types.Set    `tfsdk:"ip_filter"`
	Settings types.String `tfsdk:"redis_settings"`
}

var ResourceRedisSchema = schema.SingleNestedBlock{
	MarkdownDescription: "*redis* database service type specific arguments. Structure is documented below.",
	Attributes: map[string]schema.Attribute{
		"ip_filter": schema.SetAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "A list of CIDR blocks to allow incoming connections from.",
			Optional:            true,
			Validators: []validator.Set{
				setvalidator.ValueStringsAre(validators.IsCIDRNetworkValidator{Min: 0, Max: 128}),
			},
		},
		"redis_settings": schema.StringAttribute{
			MarkdownDescription: "Redis configuration settings in JSON format (`exo dbaas type show redis --settings=redis` for reference).",
			Optional:            true,
			Computed:            true,
		},
	},
}

// createRedis function handles Redis specific part of database resource creation logic.
func (r *Resource) createRedis(ctx context.Context, data *ResourceModel, diagnostics *diag.Diagnostics) {
	redisData := &ResourceRedisModel{}
	if data.Redis != nil {
		redisData = data.Redis
	}

	service := oapi.CreateDbaasServiceRedisJSONRequestBody{
		Plan:                  data.Plan.ValueString(),
		TerminationProtection: data.TerminationProtection.ValueBoolPointer(),
	}

	if !redisData.IpFilter.IsNull() {
		obj := []string{}
		dg := redisData.IpFilter.ElementsAs(ctx, &obj, false)
		if dg.HasError() {
			diagnostics.Append(dg...)
			return
		}

		service.IpFilter = &obj
	}

	if !data.MaintenanceDOW.IsNull() && !data.MaintenanceTime.IsNull() {
		service.Maintenance = &struct {
			Dow  oapi.CreateDbaasServiceRedisJSONBodyMaintenanceDow `json:"dow"`
			Time string                                             `json:"time"`
		}{
			Dow:  oapi.CreateDbaasServiceRedisJSONBodyMaintenanceDow(data.MaintenanceDOW.ValueString()),
			Time: data.MaintenanceTime.ValueString(),
		}
	}

	settingsSchema, err := r.client.GetDbaasSettingsRedisWithResponse(ctx)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, got error: %s", err))
		return
	}
	if settingsSchema.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
		return
	}

	settings := redisData.Settings.ValueString()
	if settings != "" {
		obj, err := validateSettings(settings, settingsSchema.JSON200.Settings.Redis)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return
		}
		service.RedisSettings = &obj
	}

	res, err := r.client.CreateDbaasServiceRedisWithResponse(
		ctx,
		oapi.DbaasServiceName(data.Name.ValueString()),
		service,
	)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service redis, got error: %s", err))
		return
	}
	if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
		return
	}

	r.readRedis(ctx, data, diagnostics)
}

// readRedis function handles Redis specific part of database resource Read logic.
// It is used in the dedicated Read action but also as a finishing step of Create, Update and Import.
func (r *Resource) readRedis(ctx context.Context, data *ResourceModel, diagnostics *diag.Diagnostics) {
	caCert, err := r.client.GetDatabaseCACertificate(context.Background(), data.Zone.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get CA Certificate: %s", err))
		return
	}
	data.CA = types.StringValue(caCert)

	res, err := r.client.GetDbaasServiceRedisWithResponse(ctx, oapi.DbaasServiceName(data.Id.ValueString()))
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service redis, got error: %s", err))
		return
	}
	if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service redis, unexpected status: %s", res.Status()))
		return
	}

	apiService := res.JSON200

	data.CreatedAt = types.StringValue(apiService.CreatedAt.String())
	data.DiskSize = types.Int64PointerValue(apiService.DiskSize)
	data.NodeCPUs = types.Int64PointerValue(apiService.NodeCpuCount)
	data.NodeMemory = types.Int64PointerValue(apiService.NodeMemory)
	data.Nodes = types.Int64PointerValue(apiService.NodeCount)
	data.State = types.StringPointerValue((*string)(apiService.State))
	data.TerminationProtection = types.BoolPointerValue(apiService.TerminationProtection)
	data.UpdatedAt = types.StringValue(apiService.UpdatedAt.String())

	if data.Plan.IsNull() || data.Plan.IsUnknown() {
		data.Plan = types.StringValue(apiService.Plan)
	}

	if apiService.Maintenance == nil {
		data.MaintenanceDOW = types.StringNull()
		data.MaintenanceTime = types.StringNull()
	} else {
		data.MaintenanceDOW = types.StringValue(string(apiService.Maintenance.Dow))
		data.MaintenanceTime = types.StringValue(apiService.Maintenance.Time)
	}

	if data.Redis == nil {
		data.Redis = &ResourceRedisModel{}
	}

	if apiService.IpFilter == nil || len(*apiService.IpFilter) == 0 {
		data.Redis.IpFilter = types.SetNull(types.StringType)
	} else {
		v, dg := types.SetValueFrom(ctx, types.StringType, *apiService.IpFilter)
		if dg.HasError() {
			diagnostics.Append(dg...)
			return
		}

		data.Redis.IpFilter = v
	}

	if apiService.RedisSettings == nil {
		data.Redis.Settings = types.StringNull()
	} else {
		settings, err := json.Marshal(*apiService.RedisSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return
		}
		data.Redis.Settings = types.StringValue(string(settings))
	}
}

// updateRedis function handles Redis specific part of database resource Update logic.
func (r *Resource) updateRedis(ctx context.Context, stateData *ResourceModel, planData *ResourceModel, diagnostics *diag.Diagnostics) {
	var updated bool

	service := oapi.UpdateDbaasServiceRedisJSONRequestBody{}

	settingsSchema, err := r.client.GetDbaasSettingsRedisWithResponse(ctx)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, got error: %s", err))
		return
	}
	if settingsSchema.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
		return
	}

	if !planData.MaintenanceDOW.Equal(stateData.MaintenanceDOW) || !planData.MaintenanceTime.Equal(stateData.MaintenanceTime) {
		service.Maintenance = &struct {
			Dow  oapi.UpdateDbaasServiceRedisJSONBodyMaintenanceDow `json:"dow"`
			Time string                                             `json:"time"`
		}{
			Dow:  oapi.UpdateDbaasServiceRedisJSONBodyMaintenanceDow(planData.MaintenanceDOW.ValueString()),
			Time: planData.MaintenanceTime.ValueString(),
		}
		updated = true
	}

	if !planData.Plan.Equal(stateData.Plan) {
		service.Plan = planData.Plan.ValueStringPointer()
		updated = true
	}

	if !planData.TerminationProtection.Equal(stateData.TerminationProtection) {
		service.TerminationProtection = planData.TerminationProtection.ValueBoolPointer()
		updated = true
	}

	stateRedisData := &ResourceRedisModel{}
	if stateData.Redis != nil {
		stateRedisData = stateData.Redis
	}
	planRedisData := &ResourceRedisModel{}
	if planData.Redis != nil {
		planRedisData = planData.Redis
	}

	if !planRedisData.IpFilter.Equal(stateRedisData.IpFilter) {
		obj := []string{}
		if len(planRedisData.IpFilter.Elements()) > 0 {
			dg := planRedisData.IpFilter.ElementsAs(ctx, &obj, false)
			if dg.HasError() {
				diagnostics.Append(dg...)
				return
			}
		}
		service.IpFilter = &obj
		updated = true
	}

	if !planRedisData.Settings.Equal(stateRedisData.Settings) {
		if planRedisData.Settings.ValueString() != "" {
			obj, err := validateSettings(planRedisData.Settings.ValueString(), settingsSchema.JSON200.Settings.Redis)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid Redis settings: %s", err))
				return
			}
			service.RedisSettings = &obj
		}
		updated = true
	}

	if !updated {
		tflog.Info(ctx, "no updates detected", map[string]interface{}{})
		return
	}

	res, err := r.client.UpdateDbaasServiceRedisWithResponse(
		ctx,
		oapi.DbaasServiceName(planData.Id.ValueString()),
		service,
	)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service redis, got error: %s", err))
		return
	}
	if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
		return
	}

	r.readRedis(ctx, planData, diagnostics)
}
