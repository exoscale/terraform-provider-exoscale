package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/validators"
)

type ResourceValkeyModel struct {
	IPFilter types.Set    `tfsdk:"ip_filter"`
	Settings types.String `tfsdk:"valkey_settings"`
}

var ResourceValkeySchema = schema.SingleNestedBlock{
	MarkdownDescription: "*valkey* database service type specific arguments. Structure is documented below.",
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
		"valkey_settings": schema.StringAttribute{
			MarkdownDescription: "Valkey configuration settings in JSON format (`exo dbaas type show valkey --settings=valkey` for reference).",
			Optional:            true,
			Computed:            true,
		},
	},
}

// createValkey function handles Valkey specific part of database resource creation logic.
func (r *ServiceResource) createValkey(ctx context.Context, data *ServiceResourceModel, diagnostics *diag.Diagnostics) {
	service := v3.CreateDBAASServiceValkeyRequest{
		Plan:                  data.Plan.ValueString(),
		TerminationProtection: data.TerminationProtection.ValueBoolPointer(),
	}

	client, err := utils.SwitchClientZone(ctx, r.clientV3, v3.ZoneName(data.Zone.ValueString()))
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to init client, got error: %s", err))
		return
	}

	if !data.MaintenanceDOW.IsUnknown() && !data.MaintenanceTime.IsUnknown() {
		service.Maintenance = &v3.CreateDBAASServiceValkeyRequestMaintenance{
			Dow:  v3.CreateDBAASServiceValkeyRequestMaintenanceDow(data.MaintenanceDOW.ValueString()),
			Time: data.MaintenanceTime.ValueString(),
		}
	}

	if data.Valkey != nil {
		if !data.Valkey.IPFilter.IsUnknown() {
			obj := []string{}
			if len(data.Valkey.IPFilter.Elements()) > 0 {
				dg := data.Valkey.IPFilter.ElementsAs(ctx, &obj, false)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
			}

			service.IPFilter = obj
		}

		if !data.Valkey.Settings.IsUnknown() {
			settingsSchema, err := client.GetDBAASSettingsValkey(ctx)
			if err != nil {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, got error: %s", err))
				return
			}

			settings, err := validateSettings(data.Valkey.Settings.ValueString(), settingsSchema.Settings.Valkey)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}

			ssl := getSettingBool(settings, "ssl")
			service.ValkeySettings = &v3.JSONSchemaValkey{
				AclChannelsDefault:            v3.JSONSchemaValkeyAclChannelsDefault(getSettingString(settings, "acl_channels_default")),
				IoThreads:                     getSettingFloat64(settings, "io_threads"),
				LfuDecayTime:                  getSettingFloat64(settings, "lfu_decay_time"),
				LfuLogFactor:                  getSettingFloat64(settings, "lfu_log_factor"),
				MaxmemoryPolicy:               v3.JSONSchemaValkeyMaxmemoryPolicy(getSettingString(settings, "maxmemory_policy")),
				NotifyKeyspaceEvents:          getSettingString(settings, "notify_keyspace_events"),
				NumberOfDatabases:             getSettingFloat64(settings, "number_of_databases"),
				Persistence:                   v3.JSONSchemaValkeyPersistence(getSettingString(settings, "persistence")),
				PubsubClientOutputBufferLimit: getSettingFloat64(settings, "pubsub_client_output_buffer_limit"),
				SSL:                           &ssl,
				Timeout:                       getSettingFloat64(settings, "timeout"),
			}
		}
	}

	_, err = client.CreateDBAASServiceValkey(
		ctx,
		data.Name.ValueString(),
		service,
	)

	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service valkey, got error: %s", err))
		return
	}

	res, err := client.GetDBAASServiceValkey(ctx, data.Name.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service valkey, got error: %s", err))
		return
	}

	// Set computed attributes
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

	if data.Valkey.IPFilter.IsUnknown() {
		data.Valkey.IPFilter = types.SetNull(types.StringType)
		if apiService.IPFilter != nil {
			v, dg := types.SetValueFrom(ctx, types.StringType, apiService.IPFilter)
			if dg.HasError() {
				diagnostics.Append(dg...)
				return
			}
			data.Valkey.IPFilter = v
		}
	}

	if data.Valkey.Settings.IsUnknown() {
		data.Valkey.Settings = types.StringNull()
		if apiService.ValkeySettings != nil {
			settings, err := json.Marshal(*apiService.ValkeySettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			data.Valkey.Settings = types.StringValue(string(settings))
		}
	}
}

// readValkey function handles Valkey specific part of database resource Read logic.
// It is used in the dedicated Read action but also as a finishing step of Create, Update and Import.
func (r *ServiceResource) readValkey(ctx context.Context, data *ServiceResourceModel, diagnostics *diag.Diagnostics) {
	client, err := utils.SwitchClientZone(ctx, r.clientV3, v3.ZoneName(data.Zone.ValueString()))
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to init client, got error: %s", err))
		return
	}

	caCert, err := client.GetDBAASCACertificate(ctx)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get CA Certificate: %s", err))
		return
	}
	data.CA = types.StringValue(caCert.Certificate)

	res, err := client.GetDBAASServiceValkey(ctx, data.Id.ValueString())

	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service valkey, got error: %s", err))
		return
	}

	apiService := res
	serviceState := string(apiService.State)

	data.CreatedAt = types.StringValue(apiService.CreatedAT.String())
	data.DiskSize = types.Int64PointerValue(&apiService.DiskSize)
	data.NodeCPUs = types.Int64PointerValue(&apiService.NodeCPUCount)
	data.NodeMemory = types.Int64PointerValue(&apiService.NodeMemory)
	data.Nodes = types.Int64PointerValue(&apiService.NodeCount)
	data.State = types.StringPointerValue(&serviceState)
	data.TerminationProtection = types.BoolPointerValue(apiService.TerminationProtection)
	data.UpdatedAt = types.StringValue(apiService.UpdatedAT.String())

	data.MaintenanceDOW = types.StringNull()
	data.MaintenanceTime = types.StringNull()
	if apiService.Maintenance != nil {
		data.MaintenanceDOW = types.StringValue(string(apiService.Maintenance.Dow))
		data.MaintenanceTime = types.StringValue(apiService.Maintenance.Time)
	}

	// Database block is required but it may be nil during import.
	if data.Valkey == nil {
		data.Valkey = &ResourceValkeyModel{}
	}

	data.Valkey.IPFilter = types.SetNull(types.StringType)
	if apiService.IPFilter != nil {
		v, dg := types.SetValueFrom(ctx, types.StringType, apiService.IPFilter)
		if dg.HasError() {
			diagnostics.Append(dg...)
			return
		}

		data.Valkey.IPFilter = v
	}

	data.Valkey.Settings = types.StringNull()
	if apiService.ValkeySettings != nil {
		settings, err := json.Marshal(*apiService.ValkeySettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return
		}
		data.Valkey.Settings = types.StringValue(string(settings))
	}
}

// updateValkey function handles Valkey specific part of database resource Update logic.
func (r *ServiceResource) updateValkey(ctx context.Context, stateData *ServiceResourceModel, planData *ServiceResourceModel, diagnostics *diag.Diagnostics) {
	var updated bool

	client, err := utils.SwitchClientZone(ctx, r.clientV3, v3.ZoneName(stateData.Zone.ValueString()))

	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("couldn't create client error: %s", err))
		return
	}

	service := v3.UpdateDBAASServiceValkeyRequest{}

	if (!planData.MaintenanceDOW.Equal(stateData.MaintenanceDOW) && !planData.MaintenanceDOW.IsUnknown()) ||
		(!planData.MaintenanceTime.Equal(stateData.MaintenanceTime) && !planData.MaintenanceTime.IsUnknown()) {
		service.Maintenance = &v3.UpdateDBAASServiceValkeyRequestMaintenance{
			Dow:  v3.UpdateDBAASServiceValkeyRequestMaintenanceDow(planData.MaintenanceDOW.ValueString()),
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

	if planData.Valkey != nil {
		if stateData.Valkey == nil {
			stateData.Valkey = &ResourceValkeyModel{}
		}

		if !planData.Valkey.IPFilter.Equal(stateData.Valkey.IPFilter) {
			ips := []string{}
			if len(planData.Valkey.IPFilter.Elements()) > 0 {
				dg := planData.Valkey.IPFilter.ElementsAs(ctx, &ips, false)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
			}
			service.IPFilter = ips
			stateData.Valkey.IPFilter = planData.Valkey.IPFilter
			updated = true
		}

		if !planData.Valkey.Settings.Equal(stateData.Valkey.Settings) {
			settingsSchema, err := client.GetDBAASSettingsValkey(ctx)
			if err != nil {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, got error: %s", err))
				return
			}

			if planData.Valkey.Settings.ValueString() != "" {
				settings, err := validateSettings(planData.Valkey.Settings.ValueString(), settingsSchema.Settings.Valkey)
				if err != nil {
					diagnostics.AddError("Validation error", fmt.Sprintf("invalid Valkey settings: %s", err))
					return
				}

				ssl := getSettingBool(settings, "ssl")
				service.ValkeySettings = &v3.JSONSchemaValkey{
					AclChannelsDefault:            v3.JSONSchemaValkeyAclChannelsDefault(getSettingString(settings, "acl_channels_default")),
					IoThreads:                     getSettingFloat64(settings, "io_threads"),
					LfuDecayTime:                  getSettingFloat64(settings, "lfu_decay_time"),
					LfuLogFactor:                  getSettingFloat64(settings, "lfu_log_factor"),
					MaxmemoryPolicy:               v3.JSONSchemaValkeyMaxmemoryPolicy(getSettingString(settings, "maxmemory_policy")),
					NotifyKeyspaceEvents:          getSettingString(settings, "notify_keyspace_events"),
					NumberOfDatabases:             getSettingFloat64(settings, "number_of_databases"),
					Persistence:                   v3.JSONSchemaValkeyPersistence(getSettingString(settings, "persistence")),
					PubsubClientOutputBufferLimit: getSettingFloat64(settings, "pubsub_client_output_buffer_limit"),
					SSL:                           &ssl,
					Timeout:                       getSettingFloat64(settings, "timeout"),
				}
			}
			stateData.Valkey.Settings = planData.Valkey.Settings
			updated = true
		}
	}

	if !updated {
		tflog.Info(ctx, "no updates detected", map[string]interface{}{})
		return
	}

	if _, err := client.UpdateDBAASServiceValkey(
		ctx,
		planData.Id.ValueString(),
		service,
	); err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service valkey, got error: %s", err))
		return
	}

	// Get the current state after update
	res, err := client.GetDBAASServiceValkey(ctx, planData.Id.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service valkey, got error: %s", err))
		return
	}

	// Update all computed attributes
	stateData.State = types.StringValue(string(res.State))
	stateData.DiskSize = types.Int64PointerValue(&res.DiskSize)
	stateData.NodeCPUs = types.Int64PointerValue(&res.NodeCPUCount)
	stateData.Nodes = types.Int64PointerValue(&res.NodeCount)
	stateData.NodeMemory = types.Int64PointerValue(&res.NodeMemory)
	stateData.UpdatedAt = types.StringValue(res.UpdatedAT.String())
	stateData.TerminationProtection = types.BoolPointerValue(res.TerminationProtection)

	// Update maintenance settings
	if res.Maintenance != nil {
		if !stateData.MaintenanceDOW.IsUnknown() {
			stateData.MaintenanceDOW = types.StringValue(string(res.Maintenance.Dow))
		}
		if !stateData.MaintenanceTime.IsUnknown() {
			stateData.MaintenanceTime = types.StringValue(res.Maintenance.Time)
		}
	} else {
		if !stateData.MaintenanceDOW.IsUnknown() {
			stateData.MaintenanceDOW = types.StringNull()
		}
		if !stateData.MaintenanceTime.IsUnknown() {
			stateData.MaintenanceTime = types.StringNull()
		}
	}
}
