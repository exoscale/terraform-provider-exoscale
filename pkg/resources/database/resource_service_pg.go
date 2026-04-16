package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	apiv2 "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/egoscale/v2/oapi"
	v3 "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/validators"
)

type ResourcePgModel struct {
	AdminPassword           types.String `tfsdk:"admin_password"`
	AdminUsername           types.String `tfsdk:"admin_username"`
	BackupSchedule          types.String `tfsdk:"backup_schedule"`
	IpFilter                types.Set    `tfsdk:"ip_filter"`
	Settings                types.String `tfsdk:"pg_settings"`
	Version                 types.String `tfsdk:"version"`
	PgbouncerSettings       types.String `tfsdk:"pgbouncer_settings"`
	PglookoutSettings       types.String `tfsdk:"pglookout_settings"`
	SharedBuffersPercentage types.Int64  `tfsdk:"shared_buffers_percentage"`
	TimescaledbSettings     types.String `tfsdk:"timescaledb_settings"`
	Variant                 types.String `tfsdk:"variant"`
	WorkMem                 types.Int64  `tfsdk:"work_mem"`
	RecoveryBackupTime      types.String `tfsdk:"recovery_backup_time"`
}

var ResourcePgSchema = schema.SingleNestedAttribute{
	Optional:            true,
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
			Computed:            true,
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
		"shared_buffers_percentage": schema.Int64Attribute{
			MarkdownDescription: "Percentage of total RAM that the database server uses for shared memory buffers. Valid range is 20-60, which corresponds to 20% - 60%. This setting adjusts the shared_buffers configuration value.",
			Optional:            true,
			Computed:            true,
			Validators: []validator.Int64{
				int64validator.Between(20, 60),
			},
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
		"timescaledb_settings": schema.StringAttribute{
			MarkdownDescription: "TimescaleDB extension configuration settings in JSON format (`exo dbaas type show pg --settings=timescaledb` for reference).",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"variant": schema.StringAttribute{
			MarkdownDescription: "PostgreSQL variant (`timescale` or `aiven`). May only be set at creation time.",
			Optional:            true,
			Validators: []validator.String{
				stringvalidator.OneOf("timescale", "aiven"),
			},
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"recovery_backup_time": schema.StringAttribute{
			MarkdownDescription: "ISO time of a backup to recover from. May only be set at creation time.",
			Optional:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"work_mem": schema.Int64Attribute{
			MarkdownDescription: "Sets the maximum amount of memory to be used by a query operation (such as a sort or hash table) before writing to temporary disk files, in MB. Default is 1MB + 0.075% of total RAM (up to 32MB).",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.Int64{
				int64planmodifier.UseStateForUnknown(),
			},
		},
	},
}

// createPg function handles PostgreSQL specific part of database resource creation logic.
func (r *ServiceResource) createPg(ctx context.Context, data *ServiceResourceModel, diagnostics *diag.Diagnostics) {
	service := oapi.CreateDbaasServicePgJSONRequestBody{
		Plan:                  data.Plan.ValueString(),
		TerminationProtection: data.TerminationProtection.ValueBoolPointer(),
	}
	var pgbouncerSettings *v3.JSONSchemaPgbouncer

	if !data.MaintenanceDOW.IsUnknown() && !data.MaintenanceTime.IsUnknown() {
		service.Maintenance = &struct {
			Dow  oapi.CreateDbaasServicePgJSONBodyMaintenanceDow `json:"dow"`
			Time string                                          `json:"time"`
		}{
			Dow:  oapi.CreateDbaasServicePgJSONBodyMaintenanceDow(data.MaintenanceDOW.ValueString()),
			Time: data.MaintenanceTime.ValueString(),
		}
	}

	if data.Pg != nil {
		if !data.Pg.Version.IsUnknown() {
			service.Version = data.Pg.Version.ValueStringPointer()
		}

		if !data.Pg.AdminPassword.IsNull() {
			service.AdminPassword = data.Pg.AdminPassword.ValueStringPointer()
		}

		if !data.Pg.AdminUsername.IsNull() {
			service.AdminUsername = data.Pg.AdminUsername.ValueStringPointer()
		}

		if !data.Pg.IpFilter.IsUnknown() {
			obj := []string{}
			if len(data.Pg.IpFilter.Elements()) > 0 {
				dg := data.Pg.IpFilter.ElementsAs(ctx, &obj, true)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
			}

			service.IpFilter = &obj
		}
		if !data.Pg.BackupSchedule.IsNull() && !data.Pg.BackupSchedule.IsUnknown() {
			bh, bm, err := parseBackupSchedule(data.Pg.BackupSchedule.ValueString())
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

		if !data.Pg.Settings.IsUnknown() {
			obj, err := validateSettings(data.Pg.Settings.ValueString(), settingsSchema.JSON200.Settings.Pg)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			service.PgSettings = &obj
		}

		if !data.Pg.PgbouncerSettings.IsUnknown() {
			settings, err := validateSettings(data.Pg.PgbouncerSettings.ValueString(), settingsSchema.JSON200.Settings.Pgbouncer)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}

			typedSettings, err := json.Marshal(settings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("unable to marshal pgbouncer settings: %s", err))
				return
			}

			pgbouncerSettings = &v3.JSONSchemaPgbouncer{}
			if err := json.Unmarshal(typedSettings, pgbouncerSettings); err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("unable to read pgbouncer settings: %s", err))
				return
			}
		}

		if !data.Pg.PglookoutSettings.IsUnknown() {
			obj, err := validateSettings(data.Pg.PglookoutSettings.ValueString(), settingsSchema.JSON200.Settings.Pglookout)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			service.PglookoutSettings = &obj
		}

		if !data.Pg.TimescaledbSettings.IsNull() && !data.Pg.TimescaledbSettings.IsUnknown() {
			obj, err := validateSettings(data.Pg.TimescaledbSettings.ValueString(), settingsSchema.JSON200.Settings.Timescaledb)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			service.TimescaledbSettings = &obj
		}

		if !data.Pg.SharedBuffersPercentage.IsNull() && !data.Pg.SharedBuffersPercentage.IsUnknown() {
			v := data.Pg.SharedBuffersPercentage.ValueInt64()
			service.SharedBuffersPercentage = &v
		}

		if !data.Pg.Variant.IsNull() {
			v := oapi.EnumPgVariant(data.Pg.Variant.ValueString())
			service.Variant = &v
		}

		if !data.Pg.RecoveryBackupTime.IsNull() {
			v := data.Pg.RecoveryBackupTime.ValueString()
			service.RecoveryBackupTime = &v
		}

		if !data.Pg.WorkMem.IsNull() && !data.Pg.WorkMem.IsUnknown() {
			v := data.Pg.WorkMem.ValueInt64()
			service.WorkMem = &v
		}
	}

	resp, err := r.client.CreateDbaasServicePgWithResponse(
		ctx,
		oapi.DbaasServiceName(data.Name.ValueString()),
		service,
	)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service pg, got error: %s", err))
		return
	}
	if resp.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable create database service pg, unexpected status: %s", resp.Status()))
		return
	}

	tflog.Info(ctx, "DB Service created, waiting for the service to be in 'running' state")
	apiService := &oapi.DbaasServicePg{}
pooling:
	for {
		select {
		case <-ctx.Done():
			diagnostics.AddError("Error", ctx.Err().Error())
			return
		default:
			res, err := r.client.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(data.Id.ValueString()))
			if err != nil {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service pg, got error: %s", err))
				return
			}
			if res.StatusCode() != http.StatusOK {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service pg, unexpected status: %s", res.Status()))
				return
			}
			if res.JSON200.State != nil {
				if *res.JSON200.State == oapi.EnumServiceStatePoweroff {
					diagnostics.AddError("Client Error", fmt.Sprintf("Unexpected service state: %s", *res.JSON200.State))
					return
				}
				if *res.JSON200.State == oapi.EnumServiceStateRunning {
					apiService = res.JSON200
					break pooling
				}
			}
			time.Sleep(time.Second * 2)
		}
	}

	if pgbouncerSettings != nil {
		clientV3, err := utils.SwitchClientZone(ctx, r.clientV3, v3.ZoneName(data.Zone.ValueString()))
		if err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to init client, got error: %s", err))
			return
		}

		op, err := clientV3.UpdateDBAASServicePG(
			ctx,
			data.Id.ValueString(),
			v3.UpdateDBAASServicePGRequest{
				PgbouncerSettings: pgbouncerSettings,
			},
		)
		if err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update database service pg, got error: %s", err))
			return
		}

		if _, err := clientV3.Wait(ctx, op, v3.OperationStateSuccess); err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update database service pg, got error: %s", err))
			return
		}

		tflog.Info(ctx, "DB Service updated with pgbouncer settings, waiting for the service to be in 'running' state")
	poolingAfterUpdate:
		for {
			select {
			case <-ctx.Done():
				diagnostics.AddError("Error", ctx.Err().Error())
				return
			default:
				res, err := r.client.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(data.Id.ValueString()))
				if err != nil {
					diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service pg, got error: %s", err))
					return
				}
				if res.StatusCode() != http.StatusOK {
					diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service pg, unexpected status: %s", res.Status()))
					return
				}
				if res.JSON200.State != nil {
					if *res.JSON200.State == oapi.EnumServiceStatePoweroff {
						diagnostics.AddError("Client Error", fmt.Sprintf("Unexpected service state: %s", *res.JSON200.State))
						return
					}
					if *res.JSON200.State == oapi.EnumServiceStateRunning {
						apiService = res.JSON200
						break poolingAfterUpdate
					}
				}
				time.Sleep(time.Second * 2)
			}
		}
	}

	// Fill in unknown values.
	caCert, err := r.client.GetDatabaseCACertificate(ctx, data.Zone.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get CA Certificate: %s", err))
		return
	}
	data.CA = types.StringValue(caCert)

	data.CreatedAt = types.StringValue(apiService.CreatedAt.String())
	data.DiskSize = types.Int64PointerValue(apiService.DiskSize)
	data.NodeCPUs = types.Int64PointerValue(apiService.NodeCpuCount)
	data.NodeMemory = types.Int64PointerValue(apiService.NodeMemory)
	data.Nodes = types.Int64PointerValue(apiService.NodeCount)
	data.State = types.StringPointerValue((*string)(apiService.State))
	data.UpdatedAt = types.StringValue(apiService.UpdatedAt.String())

	if data.TerminationProtection.IsUnknown() {
		data.TerminationProtection = types.BoolPointerValue(apiService.TerminationProtection)
	}

	if data.MaintenanceDOW.IsUnknown() && data.MaintenanceTime.IsUnknown() {
		data.MaintenanceDOW = types.StringNull()
		data.MaintenanceTime = types.StringNull()
		if apiService.Maintenance != nil {
			data.MaintenanceDOW = types.StringValue(string(apiService.Maintenance.Dow))
			data.MaintenanceTime = types.StringValue(apiService.Maintenance.Time)
		}
	}

	if data.Pg.BackupSchedule.IsUnknown() {
		data.Pg.BackupSchedule = types.StringNull()
		if apiService.BackupSchedule != nil {
			backupHour := types.Int64PointerValue(apiService.BackupSchedule.BackupHour)
			backupMinute := types.Int64PointerValue(apiService.BackupSchedule.BackupMinute)
			data.Pg.BackupSchedule = types.StringValue(fmt.Sprintf(
				"%02d:%02d",
				backupHour.ValueInt64(),
				backupMinute.ValueInt64(),
			))
		}
	}

	if data.Pg.IpFilter.IsUnknown() {
		data.Pg.IpFilter = types.SetNull(types.StringType)
		if apiService.IpFilter != nil {
			v, dg := types.SetValueFrom(ctx, types.StringType, *apiService.IpFilter)
			if dg.HasError() {
				diagnostics.Append(dg...)
				return
			}

			data.Pg.IpFilter = v
		}
	}

	if data.Pg.Version.IsUnknown() {
		data.Pg.Version = types.StringNull()
		if apiService.Version != nil {
			data.Pg.Version = types.StringValue(strings.SplitN(*apiService.Version, ".", 2)[0])
		}
	}

	if data.Pg.Settings.IsUnknown() {
		data.Pg.Settings = types.StringNull()
		if apiService.PgSettings != nil {
			settings, err := json.Marshal(*apiService.PgSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			data.Pg.Settings = types.StringValue(string(settings))
		}
	}

	if data.Pg.PgbouncerSettings.IsUnknown() {
		data.Pg.PgbouncerSettings = types.StringNull()
		if apiService.PgbouncerSettings != nil {
			settings, err := json.Marshal(*apiService.PgbouncerSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			data.Pg.PgbouncerSettings = types.StringValue(string(settings))
		}
	}

	if data.Pg.PglookoutSettings.IsUnknown() {
		data.Pg.PglookoutSettings = types.StringNull()
		if apiService.PglookoutSettings != nil {
			settings, err := json.Marshal(*apiService.PglookoutSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			data.Pg.PglookoutSettings = types.StringValue(string(settings))
		}
	}

	if data.Pg.TimescaledbSettings.IsUnknown() {
		data.Pg.TimescaledbSettings = types.StringNull()
		if apiService.TimescaledbSettings != nil {
			settings, err := json.Marshal(*apiService.TimescaledbSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			data.Pg.TimescaledbSettings = types.StringValue(string(settings))
		}
	}

	if data.Pg.SharedBuffersPercentage.IsUnknown() {
		data.Pg.SharedBuffersPercentage = types.Int64Null()
		if apiService.SharedBuffersPercentage != nil {
			data.Pg.SharedBuffersPercentage = types.Int64Value(*apiService.SharedBuffersPercentage)
		}
	}

	if data.Pg.WorkMem.IsUnknown() {
		data.Pg.WorkMem = types.Int64Null()
		if apiService.WorkMem != nil {
			data.Pg.WorkMem = types.Int64Value(*apiService.WorkMem)
		}
	}
}

// readPg function handles PostgreSQL specific part of database resource Read logic.
// It is used in the dedicated Read action but also as a finishing step of Create, Update and Import.
func (r *ServiceResource) readPg(ctx context.Context, data *ServiceResourceModel, diagnostics *diag.Diagnostics) (clearState bool) {

	res, err := r.client.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(data.Id.ValueString()))
	if err != nil {
		if errors.Is(err, apiv2.ErrNotFound) {
			return true
		}
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service pg, got error: %s", err))
		return
	}
	if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service pg, unexpected status: %s", res.Status()))
		return false
	}

	caCert, err := r.client.GetDatabaseCACertificate(ctx, data.Zone.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get CA Certificate: %s", err))
		return false
	}
	data.CA = types.StringValue(caCert)

	apiService := res.JSON200

	data.CreatedAt = types.StringValue(apiService.CreatedAt.String())
	data.DiskSize = types.Int64PointerValue(apiService.DiskSize)
	data.NodeCPUs = types.Int64PointerValue(apiService.NodeCpuCount)
	data.NodeMemory = types.Int64PointerValue(apiService.NodeMemory)
	data.Nodes = types.Int64PointerValue(apiService.NodeCount)
	data.Plan = types.StringValue(apiService.Plan)
	data.State = types.StringPointerValue((*string)(apiService.State))
	data.TerminationProtection = types.BoolPointerValue(apiService.TerminationProtection)
	data.UpdatedAt = types.StringValue(apiService.UpdatedAt.String())

	data.MaintenanceDOW = types.StringNull()
	data.MaintenanceTime = types.StringNull()
	if apiService.Maintenance != nil {
		data.MaintenanceDOW = types.StringValue(string(apiService.Maintenance.Dow))
		data.MaintenanceTime = types.StringValue(apiService.Maintenance.Time)
	}

	// Database block is required but it may be nil during import.
	if data.Pg == nil {
		data.Pg = &ResourcePgModel{}
	}

	data.Pg.BackupSchedule = types.StringNull()
	if apiService.BackupSchedule != nil {
		backupHour := types.Int64PointerValue(apiService.BackupSchedule.BackupHour)
		backupMinute := types.Int64PointerValue(apiService.BackupSchedule.BackupMinute)
		data.Pg.BackupSchedule = types.StringValue(fmt.Sprintf(
			"%02d:%02d",
			backupHour.ValueInt64(),
			backupMinute.ValueInt64(),
		))
	}

	data.Pg.IpFilter = types.SetNull(types.StringType)
	if apiService.IpFilter != nil {
		v, dg := types.SetValueFrom(ctx, types.StringType, *apiService.IpFilter)
		if dg.HasError() {
			diagnostics.Append(dg...)
			return false
		}

		data.Pg.IpFilter = v
	}

	data.Pg.Version = types.StringNull()
	if apiService.Version != nil {
		data.Pg.Version = types.StringValue(strings.SplitN(*apiService.Version, ".", 2)[0])
	}

	// For database settings, we have a special behaviour:
	// - If not set in plan we handle it as normal computed field;
	// - If set in plan we only read (manage) key(s) that were set!
	//   This prevents showing drift due to Aiven setting default values for keys not specified in plan.
	if data.Pg.Settings.IsUnknown() || apiService.PgSettings == nil {
		data.Pg.Settings = types.StringNull()
		if apiService.PgSettings != nil {
			settings, err := json.Marshal(*apiService.PgSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return false
			}

			data.Pg.Settings = types.StringValue(string(settings))
		}
	} else if data.Pg.Settings.ValueString() != "" {
		var userSettings map[string]any

		if err := json.Unmarshal([]byte(data.Pg.Settings.ValueString()), &userSettings); err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("unable to unmarshal JSON: %s", err))
			return false
		}

		PartialSettingsPatch(userSettings, *apiService.PgSettings)
		settings, err := json.Marshal(userSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return false
		}
		data.Pg.Settings = types.StringValue(string(settings))
	}

	if data.Pg.PgbouncerSettings.IsUnknown() || apiService.PgbouncerSettings == nil {
		data.Pg.PgbouncerSettings = types.StringNull()
		if apiService.PgbouncerSettings != nil {
			settings, err := json.Marshal(*apiService.PgbouncerSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return false
			}

			data.Pg.PgbouncerSettings = types.StringValue(string(settings))
		}
	} else if data.Pg.PgbouncerSettings.ValueString() != "" {
		var userSettings map[string]any

		if err := json.Unmarshal([]byte(data.Pg.PgbouncerSettings.ValueString()), &userSettings); err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("unable to unmarshal JSON: %s", err))
			return false
		}

		PartialSettingsPatch(userSettings, *apiService.PgbouncerSettings)
		settings, err := json.Marshal(userSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return false
		}
		data.Pg.PgbouncerSettings = types.StringValue(string(settings))
	}

	if data.Pg.PglookoutSettings.IsUnknown() || apiService.PglookoutSettings == nil {
		data.Pg.PglookoutSettings = types.StringNull()
		if apiService.PglookoutSettings != nil {
			settings, err := json.Marshal(*apiService.PglookoutSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return false
			}

			data.Pg.PglookoutSettings = types.StringValue(string(settings))
		}
	} else if data.Pg.PglookoutSettings.ValueString() != "" {
		var userSettings map[string]any

		if err := json.Unmarshal([]byte(data.Pg.PglookoutSettings.ValueString()), &userSettings); err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("unable to unmarshal JSON: %s", err))
			return false
		}

		PartialSettingsPatch(userSettings, *apiService.PglookoutSettings)
		settings, err := json.Marshal(userSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return false
		}
		data.Pg.PglookoutSettings = types.StringValue(string(settings))
	}

	// TimescaleDB settings follow the same partial-management pattern.
	if data.Pg.TimescaledbSettings.IsUnknown() || apiService.TimescaledbSettings == nil {
		data.Pg.TimescaledbSettings = types.StringNull()
		if apiService.TimescaledbSettings != nil {
			settings, err := json.Marshal(*apiService.TimescaledbSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return false
			}

			data.Pg.TimescaledbSettings = types.StringValue(string(settings))
		}
	} else if data.Pg.TimescaledbSettings.ValueString() != "" {
		var userSettings map[string]any

		if err := json.Unmarshal([]byte(data.Pg.TimescaledbSettings.ValueString()), &userSettings); err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("unable to unmarshal JSON: %s", err))
			return false
		}

		PartialSettingsPatch(userSettings, *apiService.TimescaledbSettings)
		settings, err := json.Marshal(userSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return false
		}
		data.Pg.TimescaledbSettings = types.StringValue(string(settings))
	}

	data.Pg.SharedBuffersPercentage = types.Int64Null()
	if apiService.SharedBuffersPercentage != nil {
		data.Pg.SharedBuffersPercentage = types.Int64Value(*apiService.SharedBuffersPercentage)
	}

	data.Pg.WorkMem = types.Int64Null()
	if apiService.WorkMem != nil {
		data.Pg.WorkMem = types.Int64Value(*apiService.WorkMem)
	}

	// Variant is not returned by the API read response, so preserve the plan value.
	// RecoveryBackupTime is write-only at creation time; not returned by the API.

	return false
}

// updatePg function handles PostgreSQL specific part of database resource Update logic.
func (r *ServiceResource) updatePg(ctx context.Context, stateData *ServiceResourceModel, planData *ServiceResourceModel, diagnostics *diag.Diagnostics) {
	var updated bool

	service := oapi.UpdateDbaasServicePgJSONRequestBody{}

	if (!planData.MaintenanceDOW.Equal(stateData.MaintenanceDOW) && !planData.MaintenanceDOW.IsUnknown()) ||
		(!planData.MaintenanceTime.Equal(stateData.MaintenanceTime) && !planData.MaintenanceTime.IsUnknown()) {
		service.Maintenance = &struct {
			Dow  oapi.UpdateDbaasServicePgJSONBodyMaintenanceDow `json:"dow"`
			Time string                                          `json:"time"`
		}{
			Dow:  oapi.UpdateDbaasServicePgJSONBodyMaintenanceDow(planData.MaintenanceDOW.ValueString()),
			Time: planData.MaintenanceTime.ValueString(),
		}
		stateData.MaintenanceDOW = planData.MaintenanceDOW
		stateData.MaintenanceTime = planData.MaintenanceTime
		updated = true
	}

	if !planData.Plan.Equal(stateData.Plan) {
		service.Plan = planData.Plan.ValueStringPointer()
		stateData.Plan = planData.Plan
		updated = true
	}

	if !planData.TerminationProtection.Equal(stateData.TerminationProtection) {
		service.TerminationProtection = planData.TerminationProtection.ValueBoolPointer()
		stateData.TerminationProtection = planData.TerminationProtection
		updated = true
	}

	if planData.Pg != nil {
		if stateData.Pg == nil {
			stateData.Pg = &ResourcePgModel{}
		}

		if !planData.Pg.BackupSchedule.IsUnknown() && !planData.Pg.BackupSchedule.Equal(stateData.Pg.BackupSchedule) {
			bh, bm, err := parseBackupSchedule(planData.Pg.BackupSchedule.ValueString())
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
			stateData.Pg.BackupSchedule = planData.Pg.BackupSchedule
			updated = true
		}

		if !planData.Pg.IpFilter.Equal(stateData.Pg.IpFilter) {
			obj := []string{}
			if len(planData.Pg.IpFilter.Elements()) > 0 {
				dg := planData.Pg.IpFilter.ElementsAs(ctx, &obj, false)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
			}
			service.IpFilter = &obj
			stateData.Pg.IpFilter = planData.Pg.IpFilter
			updated = true
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

		if !planData.Pg.Settings.Equal(stateData.Pg.Settings) {
			if planData.Pg.Settings.ValueString() != "" {
				obj, err := validateSettings(planData.Pg.Settings.ValueString(), settingsSchema.JSON200.Settings.Pg)
				if err != nil {
					diagnostics.AddError("Validation error", fmt.Sprintf("invalid Pg settings: %s", err))
					return
				}
				service.PgSettings = &obj
			}
			stateData.Pg.Settings = planData.Pg.Settings
			updated = true
		}

		if !planData.Pg.PgbouncerSettings.Equal(stateData.Pg.PgbouncerSettings) {
			if planData.Pg.PgbouncerSettings.ValueString() != "" {
				obj, err := validateSettings(planData.Pg.PgbouncerSettings.ValueString(), settingsSchema.JSON200.Settings.Pgbouncer)
				if err != nil {
					diagnostics.AddError("Validation error", fmt.Sprintf("invalid Pgbouncer settings: %s", err))
					return
				}
				service.PgbouncerSettings = &obj
			}
			stateData.Pg.PgbouncerSettings = planData.Pg.PgbouncerSettings
			updated = true
		}

		if !planData.Pg.PglookoutSettings.Equal(stateData.Pg.PglookoutSettings) {
			if planData.Pg.PglookoutSettings.ValueString() != "" {
				obj, err := validateSettings(planData.Pg.PglookoutSettings.ValueString(), settingsSchema.JSON200.Settings.Pglookout)
				if err != nil {
					diagnostics.AddError("Validation error", fmt.Sprintf("invalid Pglookout settings: %s", err))
					return
				}
				service.PglookoutSettings = &obj
			}
			stateData.Pg.PglookoutSettings = planData.Pg.PglookoutSettings
			updated = true
		}

		if !planData.Pg.TimescaledbSettings.Equal(stateData.Pg.TimescaledbSettings) {
			if planData.Pg.TimescaledbSettings.ValueString() != "" {
				obj, err := validateSettings(planData.Pg.TimescaledbSettings.ValueString(), settingsSchema.JSON200.Settings.Timescaledb)
				if err != nil {
					diagnostics.AddError("Validation error", fmt.Sprintf("invalid Timescaledb settings: %s", err))
					return
				}
				service.TimescaledbSettings = &obj
			}
			stateData.Pg.TimescaledbSettings = planData.Pg.TimescaledbSettings
			updated = true
		}

		if !planData.Pg.SharedBuffersPercentage.IsUnknown() && !planData.Pg.SharedBuffersPercentage.Equal(stateData.Pg.SharedBuffersPercentage) {
			if !planData.Pg.SharedBuffersPercentage.IsNull() {
				v := planData.Pg.SharedBuffersPercentage.ValueInt64()
				service.SharedBuffersPercentage = &v
			}
			stateData.Pg.SharedBuffersPercentage = planData.Pg.SharedBuffersPercentage
			updated = true
		}

		if !planData.Pg.Variant.IsUnknown() && !planData.Pg.Variant.Equal(stateData.Pg.Variant) {
			if !planData.Pg.Variant.IsNull() {
				v := oapi.EnumPgVariant(planData.Pg.Variant.ValueString())
				service.Variant = &v
			}
			stateData.Pg.Variant = planData.Pg.Variant
			updated = true
		}

		if !planData.Pg.WorkMem.IsUnknown() && !planData.Pg.WorkMem.Equal(stateData.Pg.WorkMem) {
			if !planData.Pg.WorkMem.IsNull() {
				v := planData.Pg.WorkMem.ValueInt64()
				service.WorkMem = &v
			}
			stateData.Pg.WorkMem = planData.Pg.WorkMem
			updated = true
		}
	}

	if !updated {
		tflog.Info(ctx, "no updates detected", map[string]any{})
		return
	}

	// Aiven would overwrite the backup schedule with random value if we don't specify it explicitly every time.
	if service.BackupSchedule == nil && !stateData.Pg.BackupSchedule.IsUnknown() {
		bh, bm, err := parseBackupSchedule(stateData.Pg.BackupSchedule.ValueString())
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

	if res, err := r.client.UpdateDbaasServicePgWithResponse(
		ctx,
		oapi.DbaasServiceName(planData.Id.ValueString()),
		service,
	); err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update database service pg, got error: %s", err))
		return
	} else if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update database service pg, unexpected status: %s", res.Status()))
		return
	}

	apiService := &oapi.DbaasServicePg{}
	if res, err := r.client.GetDbaasServicePgWithResponse(
		ctx,
		oapi.DbaasServiceName(stateData.Id.ValueString()),
	); err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service pg, got error: %s", err))
		return
	} else if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service pg, unexpected status: %s", res.Status()))
		return
	} else {
		apiService = res.JSON200
	}

	// Fill in unknown values.
	stateData.NodeCPUs = types.Int64PointerValue(apiService.NodeCpuCount)
	stateData.NodeMemory = types.Int64PointerValue(apiService.NodeMemory)
	stateData.Nodes = types.Int64PointerValue(apiService.NodeCount)
	stateData.State = types.StringPointerValue((*string)(apiService.State))
	if apiService.UpdatedAt != nil {
		stateData.UpdatedAt = types.StringValue(apiService.UpdatedAt.String())
	}
	if stateData.TerminationProtection.IsUnknown() {
		stateData.TerminationProtection = types.BoolPointerValue(apiService.TerminationProtection)
	}

	if stateData.Pg == nil {
		return
	}

	if stateData.Pg.IpFilter.IsUnknown() {
		stateData.Pg.IpFilter = types.SetNull(types.StringType)
		if apiService.IpFilter != nil {
			v, dg := types.SetValueFrom(ctx, types.StringType, *apiService.IpFilter)
			if dg.HasError() {
				diagnostics.Append(dg...)
				return
			}
			stateData.Pg.IpFilter = v
		}
	}

	if stateData.Pg.Version.IsUnknown() {
		stateData.Pg.Version = types.StringNull()
		if apiService.Version != nil {
			stateData.Pg.Version = types.StringValue(strings.SplitN(*apiService.Version, ".", 2)[0])
		}
	}

	if stateData.Pg.Settings.IsUnknown() {
		stateData.Pg.Settings = types.StringNull()
		if apiService.PgSettings != nil {
			settings, err := json.Marshal(*apiService.PgSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			stateData.Pg.Settings = types.StringValue(string(settings))
		}
	}

	if stateData.Pg.PgbouncerSettings.IsUnknown() {
		stateData.Pg.PgbouncerSettings = types.StringNull()
		if apiService.PgbouncerSettings != nil {
			settings, err := json.Marshal(*apiService.PgbouncerSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			stateData.Pg.PgbouncerSettings = types.StringValue(string(settings))
		}
	}

	if stateData.Pg.PglookoutSettings.IsUnknown() {
		stateData.Pg.PglookoutSettings = types.StringNull()
		if apiService.PglookoutSettings != nil {
			settings, err := json.Marshal(*apiService.PglookoutSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			stateData.Pg.PglookoutSettings = types.StringValue(string(settings))
		}
	}

	if stateData.Pg.TimescaledbSettings.IsUnknown() {
		stateData.Pg.TimescaledbSettings = types.StringNull()
		if apiService.TimescaledbSettings != nil {
			settings, err := json.Marshal(*apiService.TimescaledbSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			stateData.Pg.TimescaledbSettings = types.StringValue(string(settings))
		}
	}

	if stateData.Pg.SharedBuffersPercentage.IsUnknown() {
		stateData.Pg.SharedBuffersPercentage = types.Int64Null()
		if apiService.SharedBuffersPercentage != nil {
			stateData.Pg.SharedBuffersPercentage = types.Int64Value(*apiService.SharedBuffersPercentage)
		}
	}

	if stateData.Pg.WorkMem.IsUnknown() {
		stateData.Pg.WorkMem = types.Int64Null()
		if apiService.WorkMem != nil {
			stateData.Pg.WorkMem = types.Int64Value(*apiService.WorkMem)
		}
	}
}
