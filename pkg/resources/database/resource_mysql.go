package database

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/terraform-provider-exoscale/pkg/validators"
)

type ResourceMysqlModel struct {
	AdminPassword  types.String `tfsdk:"admin_password"`
	AdminUsername  types.String `tfsdk:"admin_username"`
	BackupSchedule types.String `tfsdk:"backup_schedule"`
	IpFilter       types.Set    `tfsdk:"ip_filter"`
	Settings       types.String `tfsdk:"mysql_settings"`
	Version        types.String `tfsdk:"version"`
}

var ResourceMysqlSchema = schema.SingleNestedBlock{
	MarkdownDescription: "*mysql* database service type specific arguments. Structure is documented below.",
	Attributes: map[string]schema.Attribute{
		"admin_password": schema.StringAttribute{
			MarkdownDescription: "A custom administrator account password (may only be set at creation time).",
			Optional:            true,
			Sensitive:           true,
		},
		"admin_username": schema.StringAttribute{
			MarkdownDescription: "A custom administrator account username (may only be set at creation time).",
			Optional:            true,
		},
		"backup_schedule": schema.StringAttribute{
			MarkdownDescription: "The automated backup schedule (`HH:MM`).",
			Optional:            true,
			Computed:            true,
		},
		"ip_filter": schema.SetAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "A list of CIDR blocks to allow incoming connections from.",
			Optional:            true,
			Validators: []validator.Set{
				setvalidator.ValueStringsAre(validators.IsCIDRNetworkValidator{Min: 0, Max: 128}),
			},
		},
		"mysql_settings": schema.StringAttribute{
			MarkdownDescription: "MySQL configuration settings in JSON format (`exo dbaas type show mysql --settings=mysql` for reference).",
			Optional:            true,
			Computed:            true,
		},
		"version": schema.StringAttribute{
			MarkdownDescription: "MySQL major version (`exo dbaas type show mysql` for reference; may only be set at creation time).",
			Optional:            true,
			Computed:            true,
		},
	},
}

// createMysql function handles MySQL specific part of database resource creation logic.
func (r *Resource) createMysql(ctx context.Context, data *ResourceModel, diagnostics *diag.Diagnostics) {
	mysqlData := &ResourceMysqlModel{}
	if data.Mysql != nil {
		mysqlData = data.Mysql
	}

	service := oapi.CreateDbaasServiceMysqlJSONRequestBody{
		Plan:                  data.Plan.ValueString(),
		TerminationProtection: data.TerminationProtection.ValueBoolPointer(),
		Version:               mysqlData.Version.ValueStringPointer(),
	}

	if !mysqlData.AdminPassword.IsUnknown() {
		service.AdminPassword = mysqlData.AdminPassword.ValueStringPointer()
	}

	if !mysqlData.AdminUsername.IsUnknown() {
		service.AdminUsername = mysqlData.AdminUsername.ValueStringPointer()
	}

	if !mysqlData.IpFilter.IsNull() {
		obj := []string{}
		dg := mysqlData.IpFilter.ElementsAs(ctx, &obj, false)
		if dg.HasError() {
			diagnostics.Append(dg...)
			return
		}

		service.IpFilter = &obj
	}

	if !data.MaintenanceDOW.IsNull() && !data.MaintenanceTime.IsNull() {
		service.Maintenance = &struct {
			Dow  oapi.CreateDbaasServiceMysqlJSONBodyMaintenanceDow `json:"dow"`
			Time string                                             `json:"time"`
		}{
			Dow:  oapi.CreateDbaasServiceMysqlJSONBodyMaintenanceDow(data.MaintenanceDOW.ValueString()),
			Time: data.MaintenanceTime.ValueString(),
		}
	}

	if !mysqlData.BackupSchedule.IsNull() {
		bh, bm, err := parseBackupSchedule(mysqlData.BackupSchedule.ValueString())
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("Unable to parse backup schedule, got error: %s", err))
			return
		}

		service.BackupSchedule = &struct {
			BackupHour   *int64 `json:"backup-hour,omitempty"`
			BackupMinute *int64 `json:"backup-minute,omitempty"`
		}{
			BackupHour:   &bh,
			BackupMinute: &bm,
		}
	}

	settingsSchema, err := r.client.GetDbaasSettingsMysqlWithResponse(ctx)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, got error: %s", err))
		return
	}
	if settingsSchema.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
		return
	}

	settings := mysqlData.Settings.ValueString()
	if settings != "" {
		obj, err := validateSettings(settings, settingsSchema.JSON200.Settings.Mysql)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return
		}
		service.MysqlSettings = &obj
	}

	res, err := r.client.CreateDbaasServiceMysqlWithResponse(
		ctx,
		oapi.DbaasServiceName(data.Name.ValueString()),
		service,
	)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service mysql, got error: %s", err))
		return
	}
	if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
		return
	}

	r.readMysql(ctx, data, diagnostics)
}

// readMysql function handles MySQL specific part of database resource Read logic.
// It is used in the dedicated Read action but also as a finishing step of Create, Update and Import.
func (r *Resource) readMysql(ctx context.Context, data *ResourceModel, diagnostics *diag.Diagnostics) {
	caCert, err := r.client.GetDatabaseCACertificate(context.Background(), data.Zone.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get CA Certificate: %s", err))
		return
	}
	data.CA = types.StringValue(caCert)

	res, err := r.client.GetDbaasServiceMysqlWithResponse(ctx, oapi.DbaasServiceName(data.Id.ValueString()))
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service mysql, got error: %s", err))
		return
	}
	if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service mysql, unexpected status: %s", res.Status()))
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

	if data.Mysql == nil {
		data.Mysql = &ResourceMysqlModel{}
	}

	if apiService.BackupSchedule == nil {
		data.Mysql.BackupSchedule = types.StringNull()
	} else {
		backupHour := types.Int64PointerValue(apiService.BackupSchedule.BackupHour)
		backupMinute := types.Int64PointerValue(apiService.BackupSchedule.BackupMinute)
		data.Mysql.BackupSchedule = types.StringValue(fmt.Sprintf(
			"%02d:%02d",
			backupHour.ValueInt64(),
			backupMinute.ValueInt64(),
		))
	}

	if apiService.IpFilter == nil || len(*apiService.IpFilter) == 0 {
		data.Mysql.IpFilter = types.SetNull(types.StringType)
	} else {
		v, dg := types.SetValueFrom(ctx, types.StringType, *apiService.IpFilter)
		if dg.HasError() {
			diagnostics.Append(dg...)
			return
		}

		data.Mysql.IpFilter = v
	}

	if apiService.Version == nil {
		data.Mysql.Version = types.StringNull()
	} else {
		data.Mysql.Version = types.StringValue(strings.SplitN(*apiService.Version, ".", 2)[0])
	}

	if apiService.MysqlSettings == nil {
		data.Mysql.Settings = types.StringNull()
	} else {
		settings, err := json.Marshal(*apiService.MysqlSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return
		}
		data.Mysql.Settings = types.StringValue(string(settings))
	}
}

// updateMysql function handles MySQL specific part of database resource Update logic.
func (r *Resource) updateMysql(ctx context.Context, stateData *ResourceModel, planData *ResourceModel, diagnostics *diag.Diagnostics) {
	var updated bool

	service := oapi.UpdateDbaasServiceMysqlJSONRequestBody{}

	settingsSchema, err := r.client.GetDbaasSettingsMysqlWithResponse(ctx)
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
			Dow  oapi.UpdateDbaasServiceMysqlJSONBodyMaintenanceDow `json:"dow"`
			Time string                                             `json:"time"`
		}{
			Dow:  oapi.UpdateDbaasServiceMysqlJSONBodyMaintenanceDow(planData.MaintenanceDOW.ValueString()),
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

	stateMysqlData := &ResourceMysqlModel{}
	if stateData.Mysql != nil {
		stateMysqlData = stateData.Mysql
	}
	planMysqlData := &ResourceMysqlModel{}
	if planData.Mysql != nil {
		planMysqlData = planData.Mysql
	}

	if !planMysqlData.BackupSchedule.Equal(stateMysqlData.BackupSchedule) {
		bh, bm, err := parseBackupSchedule(planMysqlData.BackupSchedule.ValueString())
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("Unable to parse backup schedule, got error: %s", err))
			return
		}

		service.BackupSchedule = &struct {
			BackupHour   *int64 `json:"backup-hour,omitempty"`
			BackupMinute *int64 `json:"backup-minute,omitempty"`
		}{
			BackupHour:   &bh,
			BackupMinute: &bm,
		}
		updated = true
	}

	if !planMysqlData.IpFilter.Equal(stateMysqlData.IpFilter) {
		obj := []string{}
		if len(planMysqlData.IpFilter.Elements()) > 0 {
			dg := planMysqlData.IpFilter.ElementsAs(ctx, &obj, false)
			if dg.HasError() {
				diagnostics.Append(dg...)
				return
			}
		}
		service.IpFilter = &obj
		updated = true
	}

	if !planMysqlData.Settings.Equal(stateMysqlData.Settings) {
		if planMysqlData.Settings.ValueString() != "" {
			obj, err := validateSettings(planMysqlData.Settings.ValueString(), settingsSchema.JSON200.Settings.Mysql)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid Mysql settings: %s", err))
				return
			}
			service.MysqlSettings = &obj
		}
		updated = true
	}

	if !updated {
		tflog.Info(ctx, "no updates detected", map[string]interface{}{})
		return
	}

	// Aiven would overwrite the backup schedule with random value if we don't specify it explicitly every time.
	if service.BackupSchedule == nil {
		bh, bm, err := parseBackupSchedule(planMysqlData.BackupSchedule.ValueString())
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("Unable to parse backup schedule, got error: %s", err))
			return
		}

		service.BackupSchedule = &struct {
			BackupHour   *int64 `json:"backup-hour,omitempty"`
			BackupMinute *int64 `json:"backup-minute,omitempty"`
		}{
			BackupHour:   &bh,
			BackupMinute: &bm,
		}
	}

	res, err := r.client.UpdateDbaasServiceMysqlWithResponse(
		ctx,
		oapi.DbaasServiceName(planData.Id.ValueString()),
		service,
	)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service mysql, got error: %s", err))
		return
	}
	if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
		return
	}

	r.readMysql(ctx, planData, diagnostics)
}
