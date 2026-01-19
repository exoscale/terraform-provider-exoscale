package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/validators"
)

type ResourceThanosModel struct {
	IPFilter types.Set    `tfsdk:"ip_filter"`
	Settings types.String `tfsdk:"thanos_settings"`

	// Computed URIs from ConnectionInfo
	QueryFrontendURI       types.String `tfsdk:"query_frontend_uri"`
	QueryURI               types.String `tfsdk:"query_uri"`
	ReceiverRemoteWriteURI types.String `tfsdk:"receiver_remote_write_uri"`
}

var ResourceThanosSchema = schema.SingleNestedBlock{
	MarkdownDescription: "*thanos* database service type specific arguments. Structure is documented below.",
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
		"thanos_settings": schema.StringAttribute{
			MarkdownDescription: "Thanos configuration settings in JSON format (`exo dbaas type show thanos --settings=thanos` for reference).",
			Optional:            true,
			Computed:            true,
		},
		"query_frontend_uri": schema.StringAttribute{
			MarkdownDescription: "Thanos Query Frontend URI.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"query_uri": schema.StringAttribute{
			MarkdownDescription: "Thanos Query URI.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"receiver_remote_write_uri": schema.StringAttribute{
			MarkdownDescription: "Thanos Receiver Remote Write URI.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
	},
}

// createThanos function handles Thanos specific part of database resource creation logic.
func (r *ServiceResource) createThanos(ctx context.Context, data *ServiceResourceModel, diagnostics *diag.Diagnostics) {
	service := v3.CreateDBAASServiceThanosRequest{
		Plan:                  data.Plan.ValueString(),
		TerminationProtection: data.TerminationProtection.ValueBoolPointer(),
	}

	client, err := utils.SwitchClientZone(ctx, r.clientV3, v3.ZoneName(data.Zone.ValueString()))
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to init client, got error: %s", err))
		return
	}

	if !data.MaintenanceDOW.IsUnknown() && !data.MaintenanceTime.IsUnknown() {
		service.Maintenance = &v3.CreateDBAASServiceThanosRequestMaintenance{
			Dow:  v3.CreateDBAASServiceThanosRequestMaintenanceDow(data.MaintenanceDOW.ValueString()),
			Time: data.MaintenanceTime.ValueString(),
		}
	}

	if data.Thanos != nil {
		if !data.Thanos.IPFilter.IsUnknown() {
			obj := []string{}
			if len(data.Thanos.IPFilter.Elements()) > 0 {
				dg := data.Thanos.IPFilter.ElementsAs(ctx, &obj, false)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
			}

			service.IPFilter = obj
		}

		if !data.Thanos.Settings.IsUnknown() {
			settings, err := parseThanosSettings(data.Thanos.Settings.ValueString())
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}

			service.ThanosSettings = settings
		}
	}

	_, err = client.CreateDBAASServiceThanos(
		ctx,
		data.Name.ValueString(),
		service,
	)

	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service thanos, got error: %s", err))
		return
	}

	res, err := client.GetDBAASServiceThanos(ctx, data.Name.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service thanos, got error: %s", err))
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

	if data.Thanos.IPFilter.IsUnknown() {
		data.Thanos.IPFilter = types.SetNull(types.StringType)
		if apiService.IPFilter != nil {
			v, dg := types.SetValueFrom(ctx, types.StringType, apiService.IPFilter)
			if dg.HasError() {
				diagnostics.Append(dg...)
				return
			}
			data.Thanos.IPFilter = v
		}
	}

	if data.Thanos.Settings.IsUnknown() {
		data.Thanos.Settings = types.StringNull()
		if apiService.ThanosSettings != nil {
			settings, err := json.Marshal(*apiService.ThanosSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			data.Thanos.Settings = types.StringValue(string(settings))
		}
	}

	// Set computed connection URIs
	data.Thanos.QueryFrontendURI = types.StringNull()
	data.Thanos.QueryURI = types.StringNull()
	data.Thanos.ReceiverRemoteWriteURI = types.StringNull()
	if apiService.ConnectionInfo != nil {
		if apiService.ConnectionInfo.QueryFrontendURI != "" {
			data.Thanos.QueryFrontendURI = types.StringValue(apiService.ConnectionInfo.QueryFrontendURI)
		}
		if apiService.ConnectionInfo.QueryURI != "" {
			data.Thanos.QueryURI = types.StringValue(apiService.ConnectionInfo.QueryURI)
		}
		if apiService.ConnectionInfo.ReceiverRemoteWriteURI != "" {
			data.Thanos.ReceiverRemoteWriteURI = types.StringValue(apiService.ConnectionInfo.ReceiverRemoteWriteURI)
		}
	}
}

// readThanos function handles Thanos specific part of database resource Read logic.
// It is used in the dedicated Read action but also as a finishing step of Create, Update and Import.
func (r *ServiceResource) readThanos(ctx context.Context, data *ServiceResourceModel, diagnostics *diag.Diagnostics) {
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

	res, err := client.GetDBAASServiceThanos(ctx, data.Id.ValueString())

	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service thanos, got error: %s", err))
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
	if data.Thanos == nil {
		data.Thanos = &ResourceThanosModel{}
	}

	data.Thanos.IPFilter = types.SetNull(types.StringType)
	if apiService.IPFilter != nil {
		v, dg := types.SetValueFrom(ctx, types.StringType, apiService.IPFilter)
		if dg.HasError() {
			diagnostics.Append(dg...)
			return
		}

		data.Thanos.IPFilter = v
	}

	data.Thanos.Settings = types.StringNull()
	if apiService.ThanosSettings != nil {
		settings, err := json.Marshal(*apiService.ThanosSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return
		}
		data.Thanos.Settings = types.StringValue(string(settings))
	}

	// Set computed connection URIs
	data.Thanos.QueryFrontendURI = types.StringNull()
	data.Thanos.QueryURI = types.StringNull()
	data.Thanos.ReceiverRemoteWriteURI = types.StringNull()
	if apiService.ConnectionInfo != nil {
		if apiService.ConnectionInfo.QueryFrontendURI != "" {
			data.Thanos.QueryFrontendURI = types.StringValue(apiService.ConnectionInfo.QueryFrontendURI)
		}
		if apiService.ConnectionInfo.QueryURI != "" {
			data.Thanos.QueryURI = types.StringValue(apiService.ConnectionInfo.QueryURI)
		}
		if apiService.ConnectionInfo.ReceiverRemoteWriteURI != "" {
			data.Thanos.ReceiverRemoteWriteURI = types.StringValue(apiService.ConnectionInfo.ReceiverRemoteWriteURI)
		}
	}
}

// updateThanos function handles Thanos specific part of database resource Update logic.
func (r *ServiceResource) updateThanos(ctx context.Context, stateData *ServiceResourceModel, planData *ServiceResourceModel, diagnostics *diag.Diagnostics) {
	var updated bool
	client, err := utils.SwitchClientZone(ctx, r.clientV3, v3.ZoneName(stateData.Zone.ValueString()))

	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("couldn't create client error: %s", err))
		return
	}

	service := v3.UpdateDBAASServiceThanosRequest{}

	if (!planData.MaintenanceDOW.Equal(stateData.MaintenanceDOW) && !planData.MaintenanceDOW.IsUnknown()) ||
		(!planData.MaintenanceTime.Equal(stateData.MaintenanceTime) && !planData.MaintenanceTime.IsUnknown()) {
		service.Maintenance = &v3.UpdateDBAASServiceThanosRequestMaintenance{
			Dow:  v3.UpdateDBAASServiceThanosRequestMaintenanceDow(planData.MaintenanceDOW.ValueString()),
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

	if planData.Thanos != nil {
		if stateData.Thanos == nil {
			stateData.Thanos = &ResourceThanosModel{}
		}

		if !planData.Thanos.IPFilter.Equal(stateData.Thanos.IPFilter) {
			ips := []string{}
			if len(planData.Thanos.IPFilter.Elements()) > 0 {
				dg := planData.Thanos.IPFilter.ElementsAs(ctx, &ips, false)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
			}
			service.IPFilter = ips
			updated = true
		}

		if !planData.Thanos.Settings.Equal(stateData.Thanos.Settings) {
			if planData.Thanos.Settings.ValueString() != "" {
				settings, err := parseThanosSettings(planData.Thanos.Settings.ValueString())
				if err != nil {
					diagnostics.AddError("Validation error", fmt.Sprintf("invalid Thanos settings: %s", err))
					return
				}
				service.ThanosSettings = settings
			}
			updated = true
		}
	}

	if !updated {
		tflog.Info(ctx, "no updates detected", map[string]interface{}{})
	} else {
		_, err := client.UpdateDBAASServiceThanos(
			ctx,
			planData.Id.ValueString(),
			service,
		)
		if err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update database service thanos, got error: %s", err))
			return
		}

		// Get the current state after update
		res, err := client.GetDBAASServiceThanos(ctx, planData.Id.ValueString())
		if err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service thanos, got error: %s", err))
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

		// Update Thanos specific settings
		if planData.Thanos != nil {
			// Update IP filter
			if res.IPFilter != nil && !planData.Thanos.IPFilter.IsUnknown() {
				v, dg := types.SetValueFrom(ctx, types.StringType, res.IPFilter)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
				planData.Thanos.IPFilter = v
			} else if !planData.Thanos.IPFilter.IsUnknown() {
				planData.Thanos.IPFilter = types.SetNull(types.StringType)
			}

			// Update Thanos settings
			if res.ThanosSettings != nil && !planData.Thanos.Settings.IsUnknown() {
				settings, err := json.Marshal(*res.ThanosSettings)
				if err != nil {
					diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
					return
				}
				planData.Thanos.Settings = types.StringValue(string(settings))
			} else if !planData.Thanos.Settings.IsUnknown() {
				planData.Thanos.Settings = types.StringNull()
			}

			// Update connection URIs
			planData.Thanos.QueryFrontendURI = types.StringNull()
			planData.Thanos.QueryURI = types.StringNull()
			planData.Thanos.ReceiverRemoteWriteURI = types.StringNull()
			if res.ConnectionInfo != nil {
				if res.ConnectionInfo.QueryFrontendURI != "" {
					planData.Thanos.QueryFrontendURI = types.StringValue(res.ConnectionInfo.QueryFrontendURI)
				}
				if res.ConnectionInfo.QueryURI != "" {
					planData.Thanos.QueryURI = types.StringValue(res.ConnectionInfo.QueryURI)
				}
				if res.ConnectionInfo.ReceiverRemoteWriteURI != "" {
					planData.Thanos.ReceiverRemoteWriteURI = types.StringValue(res.ConnectionInfo.ReceiverRemoteWriteURI)
				}
			}
		}
	}
}

// parseThanosSettings parses JSON-formatted Thanos settings and returns a JSONSchemaThanos struct.
func parseThanosSettings(settingsJSON string) (*v3.JSONSchemaThanos, error) {
	if settingsJSON == "" {
		return nil, nil
	}

	var settings map[string]interface{}
	if err := json.Unmarshal([]byte(settingsJSON), &settings); err != nil {
		return nil, fmt.Errorf("unable to unmarshal JSON: %w", err)
	}

	result := &v3.JSONSchemaThanos{}

	// Parse compactor settings
	if compactor, ok := settings["compactor"].(map[string]interface{}); ok {
		result.Compactor = &v3.JSONSchemaThanosCompactor{}
		if retentionDays, ok := compactor["retention.days"].(float64); ok {
			result.Compactor.RetentionDays = int(retentionDays)
		}
	}

	// Parse query settings
	if query, ok := settings["query"].(map[string]interface{}); ok {
		result.Query = &v3.JSONSchemaThanosQuery{}
		if v, ok := query["query.default-evaluation-interval"].(string); ok {
			result.Query.QueryDefaultEvaluationInterval = v
		}
		if v, ok := query["query.lookback-delta"].(string); ok {
			result.Query.QueryLookbackDelta = v
		}
		if v, ok := query["query.metadata.default-time-range"].(string); ok {
			result.Query.QueryMetadataDefaultTimeRange = v
		}
		if v, ok := query["query.timeout"].(string); ok {
			result.Query.QueryTimeout = v
		}
		if v, ok := query["store.limits.request-samples"].(float64); ok {
			result.Query.StoreLimitsRequestSamples = int(v)
		}
		if v, ok := query["store.limits.request-series"].(float64); ok {
			result.Query.StoreLimitsRequestSeries = int(v)
		}
	}

	// Parse query-frontend settings
	if queryFrontend, ok := settings["query-frontend"].(map[string]interface{}); ok {
		result.QueryFrontend = &v3.JSONSchemaThanosQueryFrontend{}
		if v, ok := queryFrontend["query-range.align-range-with-step"].(bool); ok {
			result.QueryFrontend.QueryRangeAlignRangeWithStep = &v
		}
	}

	return result, nil
}
