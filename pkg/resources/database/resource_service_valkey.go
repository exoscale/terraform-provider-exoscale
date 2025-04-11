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
		updated = true
	}

	if !planData.Plan.Equal(stateData.Plan) {
		service.Plan = planData.Plan.ValueString()
		updated = true
	}

	if !planData.TerminationProtection.Equal(stateData.TerminationProtection) {
		service.TerminationProtection = planData.TerminationProtection.ValueBoolPointer()
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
			updated = true
		}
	}

	if !updated {
		tflog.Info(ctx, "no updates detected", map[string]interface{}{})
	} else {
		_, err := client.UpdateDBAASServiceValkey(
			ctx,
			planData.Id.ValueString(),
			service,
		)
		if err != nil {
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
		planData.State = types.StringValue(string(res.State))
		planData.DiskSize = types.Int64PointerValue(&res.DiskSize)
		planData.NodeCPUs = types.Int64PointerValue(&res.NodeCPUCount)
		planData.Nodes = types.Int64PointerValue(&res.NodeCount)
		planData.NodeMemory = types.Int64PointerValue(&res.NodeMemory)
		planData.UpdatedAt = types.StringValue(res.UpdatedAT.String())
		planData.TerminationProtection = types.BoolPointerValue(res.TerminationProtection)

		// Update maintenance settings
		if res.Maintenance != nil {
			if !planData.MaintenanceDOW.IsUnknown() {
				planData.MaintenanceDOW = types.StringValue(string(res.Maintenance.Dow))
			}
			if !planData.MaintenanceTime.IsUnknown() {
				planData.MaintenanceTime = types.StringValue(res.Maintenance.Time)
			}
		} else {
			if !planData.MaintenanceDOW.IsUnknown() {
				planData.MaintenanceDOW = types.StringNull()
			}
			if !planData.MaintenanceTime.IsUnknown() {
				planData.MaintenanceTime = types.StringNull()
			}
		}

		// Update Valkey specific settings
		if planData.Valkey != nil {
			// Update IP filter
			if res.IPFilter != nil && !planData.Valkey.IPFilter.IsUnknown() {
				v, dg := types.SetValueFrom(ctx, types.StringType, res.IPFilter)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
				planData.Valkey.IPFilter = v
			} else if !planData.Valkey.IPFilter.IsUnknown() {
				planData.Valkey.IPFilter = types.SetNull(types.StringType)
			}

			// Update Valkey settings
			if res.ValkeySettings != nil && !planData.Valkey.Settings.IsUnknown() {
				settings, err := json.Marshal(*res.ValkeySettings)
				if err != nil {
					diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
					return
				}
				planData.Valkey.Settings = types.StringValue(string(settings))
			} else if !planData.Valkey.Settings.IsUnknown() {
				planData.Valkey.Settings = types.StringNull()
			}
		}
	}
}
