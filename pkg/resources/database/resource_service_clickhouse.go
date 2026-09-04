package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	v3 "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/validators"
)

type ResourceClickhouseModel struct {
	IPFilter           types.Set    `tfsdk:"ip_filter"`
	ClickhouseSettings types.String `tfsdk:"clickhouse_settings"`
	Version            types.String `tfsdk:"version"`
	ForkFromService    types.String `tfsdk:"fork_from_service"`
	RecoveryBackupName types.String `tfsdk:"recovery_backup_name"`
}

var ResourceClickhouseSchema = schema.SingleNestedAttribute{
	Optional:            true,
	MarkdownDescription: "*clickhouse* database service type specific arguments. Structure is documented below.",
	Attributes: map[string]schema.Attribute{
		"ip_filter": schema.SetAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "A list of CIDR blocks to allow incoming connections from.",
			Optional:            true,
			Computed:            true,
			Validators: []validator.Set{
				setvalidator.ValueStringsAre(validators.IsCIDRNetworkValidator{Min: 0, Max: 128}),
			},
		},
		"clickhouse_settings": schema.StringAttribute{
			MarkdownDescription: "ClickHouse configuration settings in JSON format (`exo dbaas type show clickhouse --settings=clickhouse` for reference).",
			Optional:            true,
			Computed:            true,
		},
		"version": schema.StringAttribute{
			MarkdownDescription: "ClickHouse major version (`exo dbaas type show clickhouse` for reference).",
			Optional:            true,
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				versionUseStateUnlessChanged(),
			},
		},
		"fork_from_service": schema.StringAttribute{
			MarkdownDescription: "Name of an existing ClickHouse service to fork from. Cannot be changed after creation.",
			Optional:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
		"recovery_backup_name": schema.StringAttribute{
			MarkdownDescription: "Name of the backup to restore when forking from `fork_from_service`.",
			Optional:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.RequiresReplace(),
			},
		},
	},
}

func parseClickhouseSettings(raw string) (*v3.JSONSchemaClickhouse, error) {
	settings := &v3.JSONSchemaClickhouse{}
	if err := json.Unmarshal([]byte(raw), settings); err != nil {
		return nil, err
	}
	return settings, nil
}

// createClickhouse function handles ClickHouse specific part of database resource creation logic.
func (r *ServiceResource) createClickhouse(ctx context.Context, data *ServiceResourceModel, diagnostics *diag.Diagnostics) {
	service := v3.CreateDBAASServiceClickhouseRequest{
		Plan:                  data.Plan.ValueString(),
		TerminationProtection: data.TerminationProtection.ValueBoolPointer(),
	}

	client, err := utils.SwitchClientZone(ctx, r.clientV3, v3.ZoneName(data.Zone.ValueString()))
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to init client, got error: %s", err))
		return
	}

	if data.Clickhouse != nil && !data.Clickhouse.Version.IsUnknown() && !data.Clickhouse.Version.IsNull() {
		service.Version = data.Clickhouse.Version.ValueString()
	}

	if data.Clickhouse != nil && !data.Clickhouse.ForkFromService.IsUnknown() && !data.Clickhouse.ForkFromService.IsNull() {
		service.ForkFromService = v3.DBAASServiceName(data.Clickhouse.ForkFromService.ValueString())
	}

	if data.Clickhouse != nil && !data.Clickhouse.RecoveryBackupName.IsUnknown() && !data.Clickhouse.RecoveryBackupName.IsNull() {
		service.RecoveryBackupName = data.Clickhouse.RecoveryBackupName.ValueString()
	}

	if !data.MaintenanceDOW.IsUnknown() && !data.MaintenanceTime.IsUnknown() {
		service.Maintenance = &v3.CreateDBAASServiceClickhouseRequestMaintenance{
			Dow:  v3.CreateDBAASServiceClickhouseRequestMaintenanceDow(data.MaintenanceDOW.ValueString()),
			Time: data.MaintenanceTime.ValueString(),
		}
	}

	if data.Clickhouse != nil {
		if !data.Clickhouse.IPFilter.IsUnknown() {
			obj := []string{}
			if len(data.Clickhouse.IPFilter.Elements()) > 0 {
				dg := data.Clickhouse.IPFilter.ElementsAs(ctx, &obj, false)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
			}
			service.IPFilter = obj
		}

		if !data.Clickhouse.ClickhouseSettings.IsUnknown() && !data.Clickhouse.ClickhouseSettings.IsNull() {
			settingsSchema, err := client.GetDBAASSettingsClickhouse(ctx)
			if err != nil {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, got error: %s", err))
				return
			}

			_, err = validateSettings(data.Clickhouse.ClickhouseSettings.ValueString(), settingsSchema.Settings.Clickhouse.Properties)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}

			settings, err := parseClickhouseSettings(data.Clickhouse.ClickhouseSettings.ValueString())
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings JSON: %s", err))
				return
			}
			service.ClickhouseSettings = settings
		}
	}

	op, err := client.CreateDBAASServiceClickhouse(
		ctx,
		data.Name.ValueString(),
		service,
	)

	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service clickhouse, got error: %s", err))
		return
	}

	_, err = client.Wait(ctx, op, v3.OperationStateSuccess)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to wait for database service clickhouse, got error: %s", err))
		return
	}

	res, err := client.GetDBAASServiceClickhouse(ctx, data.Name.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service clickhouse, got error: %s", err))
		return
	}

	// Fill in unknown values.
	apiService := res
	caCert, err := client.GetDBAASCACertificate(ctx)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get CA Certificate: %s", err))
		return
	}

	data.CA = types.StringValue(caCert.Certificate)

	serviceState := string(apiService.State)

	data.CreatedAt = types.StringValue(apiService.CreatedAT.String())
	data.DiskSize = types.Int64PointerValue(&apiService.DiskSize)
	data.NodeCPUs = types.Int64PointerValue(&apiService.NodeCPUCount)
	data.NodeMemory = types.Int64PointerValue(&apiService.NodeMemory)
	data.Nodes = types.Int64PointerValue(&apiService.NodeCount)
	data.State = types.StringPointerValue(&serviceState)
	data.UpdatedAt = types.StringValue(apiService.UpdatedAT.String())

	uri, err := uriWitoutCreds(&apiService.URI)
	if err != nil {
		diagnostics.AddError(err.Error(), "")
		return
	}
	data.URI = types.StringPointerValue(uri)

	if data.TerminationProtection.IsUnknown() {
		data.TerminationProtection = types.BoolPointerValue(apiService.TerminationProtection)
	}

	if data.MaintenanceDOW.IsUnknown() || data.MaintenanceTime.IsUnknown() {
		data.MaintenanceDOW = types.StringNull()
		data.MaintenanceTime = types.StringNull()

		if apiService.Maintenance != nil {
			data.MaintenanceDOW = types.StringValue(string(apiService.Maintenance.Dow))
			data.MaintenanceTime = types.StringValue(apiService.Maintenance.Time)
		}
	}

	if data.Clickhouse != nil {
		if data.Clickhouse.IPFilter.IsUnknown() {
			data.Clickhouse.IPFilter = types.SetNull(types.StringType)
			if apiService.IPFilter != nil {
				v, dg := types.SetValueFrom(ctx, types.StringType, apiService.IPFilter)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
				data.Clickhouse.IPFilter = v
			}
		}

		if data.Clickhouse.ClickhouseSettings.IsUnknown() {
			data.Clickhouse.ClickhouseSettings = types.StringNull()
			if apiService.ClickhouseSettings != nil {
				settings, err := json.Marshal(apiService.ClickhouseSettings)
				if err != nil {
					diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
					return
				}
				data.Clickhouse.ClickhouseSettings = types.StringValue(string(settings))
			}
		}

		if data.Clickhouse.Version.IsUnknown() {
			data.Clickhouse.Version = types.StringValue(apiService.Version)
		}
	}
}

// readClickhouse function handles ClickHouse specific part of database resource Read logic.
func (r *ServiceResource) readClickhouse(ctx context.Context, data *ServiceResourceModel, diagnostics *diag.Diagnostics) (clearState bool) {
	client, err := utils.SwitchClientZone(ctx, r.clientV3, v3.ZoneName(data.Zone.ValueString()))
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to init client, got error: %s", err))
		return false
	}

	res, err := client.GetDBAASServiceClickhouse(ctx, data.Id.ValueString())
	if err != nil {
		if errors.Is(err, v3.ErrNotFound) {
			return true
		}
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service clickhouse, got error: %s", err))
		return false
	}
	apiService := res

	caCert, err := client.GetDBAASCACertificate(ctx)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get CA Certificate: %s", err))
		return false
	}
	data.CA = types.StringValue(caCert.Certificate)

	serviceState := string(apiService.State)

	data.CreatedAt = types.StringValue(apiService.CreatedAT.String())
	data.DiskSize = types.Int64PointerValue(&apiService.DiskSize)
	data.NodeCPUs = types.Int64PointerValue(&apiService.NodeCPUCount)
	data.NodeMemory = types.Int64PointerValue(&apiService.NodeMemory)
	data.Nodes = types.Int64PointerValue(&apiService.NodeCount)
	data.State = types.StringPointerValue(&serviceState)
	data.TerminationProtection = types.BoolPointerValue(apiService.TerminationProtection)
	data.UpdatedAt = types.StringValue(apiService.UpdatedAT.String())

	uri, err := uriWitoutCreds(&apiService.URI)
	if err != nil {
		diagnostics.AddError(err.Error(), "")
		return
	}
	data.URI = types.StringPointerValue(uri)

	data.MaintenanceDOW = types.StringNull()
	data.MaintenanceTime = types.StringNull()
	if apiService.Maintenance != nil {
		data.MaintenanceDOW = types.StringValue(string(apiService.Maintenance.Dow))
		data.MaintenanceTime = types.StringValue(apiService.Maintenance.Time)
	}

	// Database block is required but it may be nil during import.
	if data.Clickhouse == nil {
		data.Clickhouse = &ResourceClickhouseModel{}
	}

	data.Clickhouse.IPFilter = types.SetNull(types.StringType)
	if apiService.IPFilter != nil {
		v, dg := types.SetValueFrom(ctx, types.StringType, apiService.IPFilter)
		if dg.HasError() {
			diagnostics.Append(dg...)
			return false
		}

		data.Clickhouse.IPFilter = v
	}

	// Preserve a user-configured settings value: the ClickHouse API returns
	// the full settings object including server-injected defaults (e.g.
	// tiered_storage_move_factor), which would otherwise drift against a
	// minimal user config on every plan. Only fill from the API when unset.
	if data.Clickhouse.ClickhouseSettings.IsNull() && apiService.ClickhouseSettings != nil {
		settings, err := json.Marshal(apiService.ClickhouseSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return false
		}
		data.Clickhouse.ClickhouseSettings = types.StringValue(string(settings))
	}

	data.Clickhouse.Version = types.StringValue(apiService.Version)

	return false
}

// updateClickhouse function handles ClickHouse specific part of database resource Update logic.
func (r *ServiceResource) updateClickhouse(ctx context.Context, stateData *ServiceResourceModel, planData *ServiceResourceModel, diagnostics *diag.Diagnostics) {
	var updated bool

	client, err := utils.SwitchClientZone(ctx, r.clientV3, v3.ZoneName(stateData.Zone.ValueString()))
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("couldn't create client error: %s", err))
		return
	}

	service := v3.UpdateDBAASServiceClickhouseRequest{}

	maintenanceDowChanged := !planData.MaintenanceDOW.Equal(stateData.MaintenanceDOW) && !planData.MaintenanceDOW.IsUnknown()
	maintenanceTimeChanged := !planData.MaintenanceTime.Equal(stateData.MaintenanceTime) && !planData.MaintenanceTime.IsUnknown()
	if maintenanceDowChanged || maintenanceTimeChanged {
		service.Maintenance = &v3.UpdateDBAASServiceClickhouseRequestMaintenance{
			Dow:  v3.UpdateDBAASServiceClickhouseRequestMaintenanceDow(planData.MaintenanceDOW.ValueString()),
			Time: planData.MaintenanceTime.ValueString(),
		}
		stateData.MaintenanceDOW = planData.MaintenanceDOW
		stateData.MaintenanceTime = planData.MaintenanceTime
		updated = true
	}

	if !planData.Plan.Equal(stateData.Plan) {
		service.Plan = planData.Plan.ValueString()
		stateData.Plan = planData.Plan
		updated = true
	}

	if !planData.TerminationProtection.Equal(stateData.TerminationProtection) {
		service.TerminationProtection = planData.TerminationProtection.ValueBoolPointer()
		stateData.TerminationProtection = planData.TerminationProtection
		updated = true
	}

	if planData.Clickhouse != nil {
		if stateData.Clickhouse == nil {
			stateData.Clickhouse = &ResourceClickhouseModel{}
		}

		if !planData.Clickhouse.IPFilter.Equal(stateData.Clickhouse.IPFilter) {
			ips := []string{}
			if len(planData.Clickhouse.IPFilter.Elements()) > 0 {
				dg := planData.Clickhouse.IPFilter.ElementsAs(ctx, &ips, false)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
			}
			service.IPFilter = ips
			stateData.Clickhouse.IPFilter = planData.Clickhouse.IPFilter
			updated = true
		}

		if !planData.Clickhouse.Version.Equal(stateData.Clickhouse.Version) {
			service.Version = planData.Clickhouse.Version.ValueString()
			stateData.Clickhouse.Version = planData.Clickhouse.Version
			updated = true
		}

		if !planData.Clickhouse.ClickhouseSettings.Equal(stateData.Clickhouse.ClickhouseSettings) {
			settingsSchema, err := client.GetDBAASSettingsClickhouse(ctx)
			if err != nil {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, got error: %s", err))
				return
			}

			if planData.Clickhouse.ClickhouseSettings.ValueString() != "" {
				_, err := validateSettings(planData.Clickhouse.ClickhouseSettings.ValueString(), settingsSchema.Settings.Clickhouse.Properties)
				if err != nil {
					diagnostics.AddError("Validation error", fmt.Sprintf("invalid ClickHouse settings: %s", err))
					return
				}

				settings, err := parseClickhouseSettings(planData.Clickhouse.ClickhouseSettings.ValueString())
				if err != nil {
					diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings JSON: %s", err))
					return
				}
				service.ClickhouseSettings = settings
			}
			stateData.Clickhouse.ClickhouseSettings = planData.Clickhouse.ClickhouseSettings
			updated = true
		}
	}

	// The ClickHouse update API requires a full settings object, not a partial
	// one: a request that omits a key the server holds (e.g. the default
	// tiered_storage_move_factor) is rejected. When a settings object is being
	// sent, fill any keys absent from the config with the server's current
	// values so the request is complete. Config values win.
	if updated && service.ClickhouseSettings != nil {
		if svc, err := client.GetDBAASServiceClickhouse(ctx, planData.Id.ValueString()); err == nil && svc.ClickhouseSettings != nil {
			merged := *svc.ClickhouseSettings
			merged.ServerSettings = service.ClickhouseSettings.ServerSettings
			if service.ClickhouseSettings.TieredStorageMoveFactor != 0 {
				merged.TieredStorageMoveFactor = service.ClickhouseSettings.TieredStorageMoveFactor
			}
			service.ClickhouseSettings = &merged
		}
	}

	if !updated {
		return
	}

	if _, err := client.UpdateDBAASServiceClickhouse(
		ctx,
		planData.Id.ValueString(),
		service,
	); err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update database service clickhouse, got error: %s", err))
		return
	}

	// Get the current state after update
	apiService, err := client.GetDBAASServiceClickhouse(ctx, planData.Id.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service clickhouse, got error: %s", err))
		return
	}

	// Fill in unknown values.
	stateData.State = types.StringValue(string(apiService.State))
	stateData.NodeCPUs = types.Int64PointerValue(&apiService.NodeCPUCount)
	stateData.Nodes = types.Int64PointerValue(&apiService.NodeCount)
	stateData.NodeMemory = types.Int64PointerValue(&apiService.NodeMemory)
	stateData.UpdatedAt = types.StringValue(apiService.UpdatedAT.String())
	stateData.TerminationProtection = types.BoolPointerValue(apiService.TerminationProtection)
	uri, err := uriWitoutCreds(&apiService.URI)
	if err != nil {
		diagnostics.AddError(err.Error(), "")
		return
	}
	stateData.URI = types.StringPointerValue(uri)
	if apiService.Maintenance != nil {
		if !stateData.MaintenanceDOW.IsUnknown() {
			stateData.MaintenanceDOW = types.StringValue(string(apiService.Maintenance.Dow))
		}
		if !stateData.MaintenanceTime.IsUnknown() {
			stateData.MaintenanceTime = types.StringValue(apiService.Maintenance.Time)
		}
	} else {
		if !stateData.MaintenanceDOW.IsUnknown() {
			stateData.MaintenanceDOW = types.StringNull()
		}
		if !stateData.MaintenanceTime.IsUnknown() {
			stateData.MaintenanceTime = types.StringNull()
		}
	}

	if stateData.Clickhouse != nil {
		if stateData.Clickhouse.IPFilter.IsUnknown() {
			stateData.Clickhouse.IPFilter = types.SetNull(types.StringType)
			if apiService.IPFilter != nil {
				v, dg := types.SetValueFrom(ctx, types.StringType, apiService.IPFilter)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
				stateData.Clickhouse.IPFilter = v
			}
		}
		if stateData.Clickhouse.ClickhouseSettings.IsUnknown() {
			stateData.Clickhouse.ClickhouseSettings = types.StringNull()
			if apiService.ClickhouseSettings != nil {
				settings, err := json.Marshal(apiService.ClickhouseSettings)
				if err != nil {
					diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
					return
				}
				stateData.Clickhouse.ClickhouseSettings = types.StringValue(string(settings))
			}
		}
	}
}
