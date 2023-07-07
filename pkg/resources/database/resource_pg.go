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

type ResourcePgModel struct {
	AdminPassword     types.String `tfsdk:"admin_password"`
	AdminUsername     types.String `tfsdk:"admin_username"`
	BackupSchedule    types.String `tfsdk:"backup_schedule"`
	IpFilter          types.Set    `tfsdk:"ip_filter"`
	Settings          types.String `tfsdk:"pg_settings"`
	Version           types.String `tfsdk:"version"`
	PgbouncerSettings types.String `tfsdk:"pgbouncer_settings"`
	PglookoutSettings types.String `tfsdk:"pglookout_settings"`
}

var ResourcePgSchema = schema.SingleNestedBlock{
	MarkdownDescription: "*pg* database service type specific arguments. Structure is documented below.",
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
		"pg_settings": schema.StringAttribute{
			MarkdownDescription: "PostgreSQL configuration settings in JSON format (`exo dbaas type show pg --settings=pg` for reference).",
			Optional:            true,
			Computed:            true,
		},
		"version": schema.StringAttribute{
			MarkdownDescription: "PostgreSQL major version (`exo dbaas type show pg` for reference; may only be set at creation time).",
			Optional:            true,
			Computed:            true,
		},
		"pgbouncer_settings": schema.StringAttribute{
			MarkdownDescription: "PgBouncer configuration settings in JSON format (`exo dbaas type show pg --settings=pgbouncer` for reference).",
			Optional:            true,
			Computed:            true,
		},
		"pglookout_settings": schema.StringAttribute{
			MarkdownDescription: "pglookout configuration settings in JSON format (`exo dbaas type show pg --settings=pglookout` for reference).",
			Optional:            true,
			Computed:            true,
		},
	},
}

// createPg function handles PostgreSQL specific part of database resource creation logic.
func (r *Resource) createPg(ctx context.Context, data *ResourceModel, diagnostics *diag.Diagnostics) {
	pgData := &ResourcePgModel{}
	if data.Pg != nil {
		pgData = data.Pg
	}

	service := oapi.CreateDbaasServicePgJSONRequestBody{
		Plan:                  data.Plan.ValueString(),
		TerminationProtection: data.TerminationProtection.ValueBoolPointer(),
		Version:               pgData.Version.ValueStringPointer(),
	}

	if !pgData.AdminPassword.IsUnknown() {
		service.AdminPassword = pgData.AdminPassword.ValueStringPointer()
	}

	if !pgData.AdminUsername.IsUnknown() {
		service.AdminUsername = pgData.AdminUsername.ValueStringPointer()
	}

	if !pgData.IpFilter.IsNull() {
		obj := []string{}
		dg := pgData.IpFilter.ElementsAs(ctx, &obj, false)
		if dg.HasError() {
			diagnostics.Append(dg...)
			return
		}

		service.IpFilter = &obj
	}

	if !data.MaintenanceDOW.IsNull() && !data.MaintenanceTime.IsNull() {
		service.Maintenance = &struct {
			Dow  oapi.CreateDbaasServicePgJSONBodyMaintenanceDow `json:"dow"`
			Time string                                          `json:"time"`
		}{
			Dow:  oapi.CreateDbaasServicePgJSONBodyMaintenanceDow(data.MaintenanceDOW.ValueString()),
			Time: data.MaintenanceTime.ValueString(),
		}
	}

	if !pgData.BackupSchedule.IsNull() {
		bh, bm, err := parseBackupSchedule(pgData.BackupSchedule.ValueString())
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

	settingsSchema, err := r.client.GetDbaasSettingsPgWithResponse(ctx)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, got error: %s", err))
		return
	}
	if settingsSchema.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
		return
	}

	settings := pgData.Settings.ValueString()
	if settings != "" {
		obj, err := validateSettings(settings, settingsSchema.JSON200.Settings.Pg)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return
		}
		service.PgSettings = &obj
	}

	bouncerSettings := pgData.PgbouncerSettings.ValueString()
	if bouncerSettings != "" {
		obj, err := validateSettings(bouncerSettings, settingsSchema.JSON200.Settings.Pgbouncer)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return
		}
		service.PgbouncerSettings = &obj
	}

	lookoutSettings := pgData.PglookoutSettings.ValueString()
	if lookoutSettings != "" {
		obj, err := validateSettings(lookoutSettings, settingsSchema.JSON200.Settings.Pglookout)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return
		}
		service.PglookoutSettings = &obj
	}

	res, err := r.client.CreateDbaasServicePgWithResponse(
		ctx,
		oapi.DbaasServiceName(data.Name.ValueString()),
		service,
	)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service pg, got error: %s", err))
		return
	}
	if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
		return
	}

	r.readPg(ctx, data, diagnostics)
}

// readPg function handles PostgreSQL specific part of database resource Read logic.
// It is used in the dedicated Read action but also as a finishing step of Create, Update and Import.
func (r *Resource) readPg(ctx context.Context, data *ResourceModel, diagnostics *diag.Diagnostics) {
	caCert, err := r.client.GetDatabaseCACertificate(context.Background(), data.Zone.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get CA Certificate: %s", err))
		return
	}
	data.CA = types.StringValue(caCert)

	res, err := r.client.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(data.Id.ValueString()))
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service pg, got error: %s", err))
		return
	}
	if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service pg, unexpected status: %s", res.Status()))
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

	if data.Pg == nil {
		data.Pg = &ResourcePgModel{}
	}

	if apiService.BackupSchedule == nil {
		data.Pg.BackupSchedule = types.StringNull()
	} else {
		backupHour := types.Int64PointerValue(apiService.BackupSchedule.BackupHour)
		backupMinute := types.Int64PointerValue(apiService.BackupSchedule.BackupMinute)
		data.Pg.BackupSchedule = types.StringValue(fmt.Sprintf(
			"%02d:%02d",
			backupHour.ValueInt64(),
			backupMinute.ValueInt64(),
		))
	}

	if apiService.IpFilter == nil || len(*apiService.IpFilter) == 0 {
		data.Pg.IpFilter = types.SetNull(types.StringType)
	} else {
		v, dg := types.SetValueFrom(ctx, types.StringType, *apiService.IpFilter)
		if dg.HasError() {
			diagnostics.Append(dg...)
			return
		}

		data.Pg.IpFilter = v
	}

	if apiService.Version == nil {
		data.Pg.Version = types.StringNull()
	} else {
		data.Pg.Version = types.StringValue(strings.SplitN(*apiService.Version, ".", 2)[0])
	}

	if apiService.PgSettings == nil {
		data.Pg.Settings = types.StringNull()
	} else {
		settings, err := json.Marshal(*apiService.PgSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return
		}
		data.Pg.Settings = types.StringValue(string(settings))
	}

	if apiService.PgbouncerSettings == nil {
		data.Pg.PgbouncerSettings = types.StringNull()
	} else {
		settings, err := json.Marshal(*apiService.PgbouncerSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return
		}
		data.Pg.PgbouncerSettings = types.StringValue(string(settings))
	}

	if apiService.PglookoutSettings == nil {
		data.Pg.PglookoutSettings = types.StringNull()
	} else {
		settings, err := json.Marshal(*apiService.PglookoutSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return
		}
		data.Pg.PglookoutSettings = types.StringValue(string(settings))
	}
}

// updatePg function handles PostgreSQL specific part of database resource Update logic.
func (r *Resource) updatePg(ctx context.Context, stateData *ResourceModel, planData *ResourceModel, diagnostics *diag.Diagnostics) {
	var updated bool

	service := oapi.UpdateDbaasServicePgJSONRequestBody{}

	settingsSchema, err := r.client.GetDbaasSettingsPgWithResponse(ctx)
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
			Dow  oapi.UpdateDbaasServicePgJSONBodyMaintenanceDow `json:"dow"`
			Time string                                          `json:"time"`
		}{
			Dow:  oapi.UpdateDbaasServicePgJSONBodyMaintenanceDow(planData.MaintenanceDOW.ValueString()),
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

	statePgData := &ResourcePgModel{}
	if stateData.Pg != nil {
		statePgData = stateData.Pg
	}
	planPgData := &ResourcePgModel{}
	if planData.Pg != nil {
		planPgData = planData.Pg
	}

	if !planPgData.BackupSchedule.Equal(statePgData.BackupSchedule) {
		bh, bm, err := parseBackupSchedule(planPgData.BackupSchedule.ValueString())
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

	if !planPgData.IpFilter.Equal(statePgData.IpFilter) {
		obj := []string{}
		if len(planPgData.IpFilter.Elements()) > 0 {
			dg := planPgData.IpFilter.ElementsAs(ctx, &obj, false)
			if dg.HasError() {
				diagnostics.Append(dg...)
				return
			}
		}
		service.IpFilter = &obj
		updated = true
	}

	if !planPgData.Settings.Equal(statePgData.Settings) {
		if planPgData.Settings.ValueString() != "" {
			obj, err := validateSettings(planPgData.Settings.ValueString(), settingsSchema.JSON200.Settings.Pg)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid Pg settings: %s", err))
				return
			}
			service.PgSettings = &obj
		}
		updated = true
	}

	if !planPgData.PgbouncerSettings.Equal(statePgData.PgbouncerSettings) {
		if planPgData.PgbouncerSettings.ValueString() != "" {
			obj, err := validateSettings(planPgData.PgbouncerSettings.ValueString(), settingsSchema.JSON200.Settings.Pgbouncer)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid Pgbouncer settings: %s", err))
				return
			}
			service.PgbouncerSettings = &obj
		}
		updated = true
	}

	if !planPgData.PglookoutSettings.Equal(statePgData.PglookoutSettings) {
		if planPgData.PglookoutSettings.ValueString() != "" {
			obj, err := validateSettings(planPgData.PglookoutSettings.ValueString(), settingsSchema.JSON200.Settings.Pglookout)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid Pglookout settings: %s", err))
				return
			}
			service.PglookoutSettings = &obj
		}
		updated = true
	}

	if !updated {
		tflog.Info(ctx, "no updates detected", map[string]interface{}{})
		return
	}

	// Aiven would overwrite the backup schedule with random value if we don't specify it explicitly every time.
	if service.BackupSchedule == nil {
		bh, bm, err := parseBackupSchedule(planPgData.BackupSchedule.ValueString())
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

	res, err := r.client.UpdateDbaasServicePgWithResponse(
		ctx,
		oapi.DbaasServiceName(planData.Id.ValueString()),
		service,
	)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service pg, got error: %s", err))
		return
	}
	if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
		return
	}

	r.readPg(ctx, planData, diagnostics)
}
