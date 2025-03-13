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
	"github.com/exoscale/terraform-provider-exoscale/pkg/validators"
)

type ResourceValkeyModel struct {
	IpFilter types.Set    `tfsdk:"ip_filter"`
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
func (r *Resource) createValkey(ctx context.Context, data *ResourceModel, diagnostics *diag.Diagnostics) {
	service := v3.CreateDBAASServiceValkeyRequest{
		Plan:                  data.Plan.ValueString(),
		TerminationProtection: data.TerminationProtection.ValueBoolPointer(),
	}

	if !data.MaintenanceDOW.IsUnknown() && !data.MaintenanceTime.IsUnknown() {
		service.Maintenance = &v3.CreateDBAASServiceValkeyRequestMaintenance{
			Dow:  v3.CreateDBAASServiceValkeyRequestMaintenanceDow(data.MaintenanceDOW.ValueString()),
			Time: data.MaintenanceTime.ValueString(),
		}
	}

	if data.Valkey != nil {
		if !data.Valkey.IpFilter.IsUnknown() {
			obj := []string{}
			if len(data.Valkey.IpFilter.Elements()) > 0 {
				dg := data.Valkey.IpFilter.ElementsAs(ctx, &obj, false)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
			}

			service.IPFilter = obj
		}

		if !data.Valkey.Settings.IsUnknown() {
			settingsSchema, err := r.clientV3.GetDBAASSettingsValkey(ctx)
			if err != nil {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, got error: %s", err))
				return
			}

			settings, err := validateSettings(data.Valkey.Settings.ValueString(), settingsSchema.Settings.Valkey)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}

			ssl := settings["ssl"].(bool)
			service.ValkeySettings = &v3.JSONSchemaValkey{
				AclChannelsDefault:            v3.JSONSchemaValkeyAclChannelsDefault(settings["acl_channels_default"].(string)),
				IoThreads:                     settings["io_threads"].(int),
				LfuDecayTime:                  settings["lfu_decay_time"].(int),
				LfuLogFactor:                  settings["lfu_log_factor"].(int),
				MaxmemoryPolicy:               v3.JSONSchemaValkeyMaxmemoryPolicy(settings["maxmemory_policy"].(string)),
				NotifyKeyspaceEvents:          settings["notify_keyspace_events"].(string),
				NumberOfDatabases:             settings["number_of_databases"].(int),
				Persistence:                   v3.JSONSchemaValkeyPersistence(settings["persistence"].(string)),
				PubsubClientOutputBufferLimit: settings["pubsub_client_output_buffer_limit"].(int),
				SSL:                           &ssl,
				Timeout:                       settings["timeout"].(int),
			}
		}
	}

	_, err := r.clientV3.CreateDBAASServiceValkey(
		ctx,
		data.Name.ValueString(),
		service,
	)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service valkey, got error: %s", err))
		return
	}

	r.readValkey(ctx, data, diagnostics)
}

// readValkey function handles Valkey specific part of database resource Read logic.
// It is used in the dedicated Read action but also as a finishing step of Create, Update and Import.
func (r *Resource) readValkey(ctx context.Context, data *ResourceModel, diagnostics *diag.Diagnostics) {
	caCert, err := r.clientV3.GetDBAASCACertificate(ctx)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get CA Certificate: %s", err))
		return
	}
	data.CA = types.StringValue(caCert.Certificate)

	res, err := r.clientV3.GetDBAASServiceValkey(ctx, data.Id.ValueString())
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

	data.Valkey.IpFilter = types.SetNull(types.StringType)
	if apiService.IPFilter != nil {
		v, dg := types.SetValueFrom(ctx, types.StringType, apiService.IPFilter)
		if dg.HasError() {
			diagnostics.Append(dg...)
			return
		}

		data.Valkey.IpFilter = v
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
func (r *Resource) updateValkey(ctx context.Context, stateData *ResourceModel, planData *ResourceModel, diagnostics *diag.Diagnostics) {
	var updated bool

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
		service.Plan = planData.Plan.String()
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

		if !planData.Valkey.IpFilter.Equal(stateData.Valkey.IpFilter) {
			obj := []string{}
			if len(planData.Valkey.IpFilter.Elements()) > 0 {
				dg := planData.Valkey.IpFilter.ElementsAs(ctx, &obj, false)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
			}
			service.IPFilter = obj
			updated = true
		}

		if !planData.Valkey.Settings.Equal(stateData.Valkey.Settings) {
			settingsSchema, err := r.clientV3.GetDBAASSettingsValkey(ctx)
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
				ssl := settings["ssl"].(bool)
				service.ValkeySettings = &v3.JSONSchemaValkey{
					AclChannelsDefault:            v3.JSONSchemaValkeyAclChannelsDefault(settings["acl_channels_default"].(string)),
					IoThreads:                     settings["io_threads"].(int),
					LfuDecayTime:                  settings["lfu_decay_time"].(int),
					LfuLogFactor:                  settings["lfu_log_factor"].(int),
					MaxmemoryPolicy:               v3.JSONSchemaValkeyMaxmemoryPolicy(settings["maxmemory_policy"].(string)),
					NotifyKeyspaceEvents:          settings["notify_keyspace_events"].(string),
					NumberOfDatabases:             settings["number_of_databases"].(int),
					Persistence:                   v3.JSONSchemaValkeyPersistence(settings["persistence"].(string)),
					PubsubClientOutputBufferLimit: settings["pubsub_client_output_buffer_limit"].(int),
					SSL:                           &ssl,
					Timeout:                       settings["timeout"].(int),
				}
			}
			updated = true
		}
	}

	if !updated {
		tflog.Info(ctx, "no updates detected", map[string]interface{}{})
	} else {
		_, err := r.clientV3.UpdateDBAASServiceValkey(
			ctx,
			planData.Id.ValueString(),
			service,
		)
		if err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service valkey, got error: %s", err))
			return
		}
	}

	r.readValkey(ctx, planData, diagnostics)
}
