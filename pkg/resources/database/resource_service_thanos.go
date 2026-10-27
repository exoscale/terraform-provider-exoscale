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
	RulerURI               types.String `tfsdk:"ruler_uri"`
}

var ResourceThanosSchema = schema.SingleNestedAttribute{
	MarkdownDescription: "*thanos* database service type specific arguments. Structure is documented below.",
	Optional:            true,
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
			MarkdownDescription: "Thanos configuration settings in JSON format (`exo dbaas type show thanos` for reference).",
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
			MarkdownDescription: "Thanos Receiver remote write URI.",
			Computed:            true,
			PlanModifiers: []planmodifier.String{
				stringplanmodifier.UseStateForUnknown(),
			},
		},
		"ruler_uri": schema.StringAttribute{
			MarkdownDescription: "Thanos Ruler URI.",
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
			settings, err := validatedThanosSettings(ctx, client, data.Thanos.Settings.ValueString())
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

	// Wait for the service to be ready: the connection URIs are only
	// populated once the service is up.
	res, err := waitForDBAASServiceReadyForFn(
		ctx,
		client.GetDBAASServiceThanos,
		data.Name.ValueString(),
		func(s *v3.DBAASServiceThanos) bool {
			return s.ConnectionInfo != nil && s.ConnectionInfo.QueryFrontendURI != ""
		},
	)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service thanos, got error: %s", err))
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

	setThanosConnectionInfo(data.Thanos, apiService.ConnectionInfo)
}

// readThanos function handles Thanos specific part of database resource Read logic.
// It is used in the dedicated Read action but also as a finishing step of Create, Update and Import.
func (r *ServiceResource) readThanos(ctx context.Context, data *ServiceResourceModel, diagnostics *diag.Diagnostics) (clearState bool) {
	client, err := utils.SwitchClientZone(ctx, r.clientV3, v3.ZoneName(data.Zone.ValueString()))
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to init client, got error: %s", err))
		return false
	}

	caCert, err := client.GetDBAASCACertificate(ctx)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get CA Certificate: %s", err))
		return false
	}
	data.CA = types.StringValue(caCert.Certificate)

	res, err := client.GetDBAASServiceThanos(ctx, data.Id.ValueString())
	if err != nil {
		if errors.Is(err, v3.ErrNotFound) {
			return true
		}

		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service thanos, got error: %s", err))
		return false
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

	uri, err := uriWitoutCreds(&apiService.URI)
	if err != nil {
		diagnostics.AddError(err.Error(), "")
		return false
	}
	data.URI = types.StringPointerValue(uri)

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
			return false
		}

		data.Thanos.IPFilter = v
	}

	data.Thanos.Settings = types.StringNull()
	if apiService.ThanosSettings != nil {
		settings, err := json.Marshal(*apiService.ThanosSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return false
		}
		data.Thanos.Settings = types.StringValue(string(settings))
	}

	setThanosConnectionInfo(data.Thanos, apiService.ConnectionInfo)

	return false
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
			stateData.Thanos.IPFilter = planData.Thanos.IPFilter
			updated = true
		}

		if !planData.Thanos.Settings.Equal(stateData.Thanos.Settings) {
			if planData.Thanos.Settings.ValueString() != "" {
				settings, err := validatedThanosSettings(ctx, client, planData.Thanos.Settings.ValueString())
				if err != nil {
					diagnostics.AddError("Validation error", fmt.Sprintf("invalid Thanos settings: %s", err))
					return
				}
				service.ThanosSettings = settings
			}
			stateData.Thanos.Settings = planData.Thanos.Settings
			updated = true
		}
	}

	if !updated {
		tflog.Info(ctx, "no updates detected", map[string]interface{}{})
		return
	}

	if _, err := client.UpdateDBAASServiceThanos(
		ctx,
		planData.Id.ValueString(),
		service,
	); err != nil {
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

	if stateData.Thanos == nil {
		return
	}

	// Only fill in values the plan left unknown: the API applies updates
	// asynchronously, so an immediate read may still return stale values.
	if stateData.Thanos.IPFilter.IsUnknown() {
		stateData.Thanos.IPFilter = types.SetNull(types.StringType)
		if res.IPFilter != nil {
			v, dg := types.SetValueFrom(ctx, types.StringType, res.IPFilter)
			if dg.HasError() {
				diagnostics.Append(dg...)
				return
			}
			stateData.Thanos.IPFilter = v
		}
	}

	if stateData.Thanos.Settings.IsUnknown() {
		stateData.Thanos.Settings = types.StringNull()
		if res.ThanosSettings != nil {
			settings, err := json.Marshal(*res.ThanosSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			stateData.Thanos.Settings = types.StringValue(string(settings))
		}
	}

	setThanosConnectionInfo(stateData.Thanos, res.ConnectionInfo)
}

// validatedThanosSettings validates user-provided JSON-formatted Thanos
// settings against the settings schema returned by the API and unmarshals
// them into a JSONSchemaThanos struct.
func validatedThanosSettings(ctx context.Context, client *v3.Client, settingsJSON string) (*v3.JSONSchemaThanos, error) {
	settingsSchema, err := client.GetDBAASSettingsThanos(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to read database settings schema: %w", err)
	}

	settings, err := validateSettings(settingsJSON, settingsSchema.Settings.Thanos)
	if err != nil {
		return nil, err
	}

	// The validated settings keys match the JSONSchemaThanos JSON tags,
	// so round-tripping through JSON fills the typed struct.
	validated, err := json.Marshal(settings)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal validated settings: %w", err)
	}

	result := &v3.JSONSchemaThanos{}
	if err := json.Unmarshal(validated, result); err != nil {
		return nil, fmt.Errorf("unable to unmarshal settings: %w", err)
	}

	return result, nil
}

// setThanosConnectionInfo copies the service connection URIs into the
// Thanos block computed attributes.
func setThanosConnectionInfo(model *ResourceThanosModel, info *v3.DBAASServiceThanosConnectionInfo) {
	model.QueryFrontendURI = types.StringNull()
	model.QueryURI = types.StringNull()
	model.ReceiverRemoteWriteURI = types.StringNull()
	model.RulerURI = types.StringNull()

	if info == nil {
		return
	}

	if info.QueryFrontendURI != "" {
		model.QueryFrontendURI = types.StringValue(info.QueryFrontendURI)
	}
	if info.QueryURI != "" {
		model.QueryURI = types.StringValue(info.QueryURI)
	}
	if info.ReceiverRemoteWriteURI != "" {
		model.ReceiverRemoteWriteURI = types.StringValue(info.ReceiverRemoteWriteURI)
	}
	if info.RulerURI != "" {
		model.RulerURI = types.StringValue(info.RulerURI)
	}
}
