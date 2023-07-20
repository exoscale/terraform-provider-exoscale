package database

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/egoscale/v2/oapi"

	"github.com/exoscale/terraform-provider-exoscale/pkg/validators"
)

type ResourceGrafanaModel struct {
	IpFilter types.Set    `tfsdk:"ip_filter"`
	Settings types.String `tfsdk:"grafana_settings"`
}

var ResourceGrafanaSchema = schema.SingleNestedBlock{
	MarkdownDescription: "*grafana* database service type specific arguments. Structure is documented below.",
	Attributes: map[string]schema.Attribute{
		"ip_filter": schema.SetAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "A list of CIDR blocks to allow incoming connections from.",
			Optional:            true,
			Validators: []validator.Set{
				setvalidator.ValueStringsAre(validators.IsCIDRNetworkValidator{Min: 0, Max: 128}),
			},
		},
		"grafana_settings": schema.StringAttribute{
			MarkdownDescription: "Grafana configuration settings in JSON format (`exo dbaas type show grafana --settings=grafana` for reference).",
			Optional:            true,
			Computed:            true,
		},
	},
}

// createGrafana function handles Grafana specific part of database resource creation logic.
func (r *Resource) createGrafana(ctx context.Context, data *ResourceModel, diagnostics *diag.Diagnostics) {
	grafanaData := &ResourceGrafanaModel{}
	if data.Grafana != nil {
		grafanaData = data.Grafana
	}

	service := oapi.CreateDbaasServiceGrafanaJSONRequestBody{
		Plan:                  data.Plan.ValueString(),
		TerminationProtection: data.TerminationProtection.ValueBoolPointer(),
	}

	if !grafanaData.IpFilter.IsUnknown() {
		obj := []string{}
		if len(grafanaData.IpFilter.Elements()) > 0 {
			dg := grafanaData.IpFilter.ElementsAs(ctx, &obj, false)
			if dg.HasError() {
				diagnostics.Append(dg...)
				return
			}
		}

		service.IpFilter = &obj
	}

	if !data.MaintenanceDOW.IsUnknown() && !data.MaintenanceTime.IsUnknown() {
		service.Maintenance = &struct {
			Dow  oapi.CreateDbaasServiceGrafanaJSONBodyMaintenanceDow `json:"dow"`
			Time string                                               `json:"time"`
		}{
			Dow:  oapi.CreateDbaasServiceGrafanaJSONBodyMaintenanceDow(data.MaintenanceDOW.ValueString()),
			Time: data.MaintenanceTime.ValueString(),
		}
	}

	settingsSchema, err := r.client.GetDbaasSettingsGrafanaWithResponse(ctx)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, got error: %s", err))
		return
	}
	if settingsSchema.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
		return
	}

	if !grafanaData.Settings.IsUnknown() {
		obj, err := validateSettings(grafanaData.Settings.ValueString(), settingsSchema.JSON200.Settings.Grafana)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return
		}
		service.GrafanaSettings = &obj
	}

	res, err := r.client.CreateDbaasServiceGrafanaWithResponse(
		ctx,
		oapi.DbaasServiceName(data.Name.ValueString()),
		service,
	)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service grafana, got error: %s", err))
		return
	}
	if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
		return
	}

	r.readGrafana(ctx, data, diagnostics)
}

// readGrafana function handles Grafana specific part of database resource Read logic.
// It is used in the dedicated Read action but also as a finishing step of Create, Update and Import.
func (r *Resource) readGrafana(ctx context.Context, data *ResourceModel, diagnostics *diag.Diagnostics) {
	caCert, err := r.client.GetDatabaseCACertificate(ctx, data.Zone.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get CA Certificate: %s", err))
		return
	}
	data.CA = types.StringValue(caCert)

	res, err := r.client.GetDbaasServiceGrafanaWithResponse(ctx, oapi.DbaasServiceName(data.Id.ValueString()))
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service grafana, got error: %s", err))
		return
	}
	if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service grafana, unexpected status: %s", res.Status()))
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

	if data.Grafana == nil {
		data.Grafana = &ResourceGrafanaModel{}
	}

	if apiService.IpFilter == nil || len(*apiService.IpFilter) == 0 {
		data.Grafana.IpFilter = types.SetNull(types.StringType)
	} else {
		v, dg := types.SetValueFrom(ctx, types.StringType, *apiService.IpFilter)
		if dg.HasError() {
			diagnostics.Append(dg...)
			return
		}

		data.Grafana.IpFilter = v
	}

	if apiService.GrafanaSettings == nil {
		data.Grafana.Settings = types.StringNull()
	} else {
		settings, err := json.Marshal(*apiService.GrafanaSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return
		}
		data.Grafana.Settings = types.StringValue(string(settings))
	}
}

// updateGrafana function handles Grafana specific part of database resource Update logic.
func (r *Resource) updateGrafana(ctx context.Context, stateData *ResourceModel, planData *ResourceModel, diagnostics *diag.Diagnostics) {
	var updated bool

	service := oapi.UpdateDbaasServiceGrafanaJSONRequestBody{}

	settingsSchema, err := r.client.GetDbaasSettingsGrafanaWithResponse(ctx)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, got error: %s", err))
		return
	}
	if settingsSchema.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
		return
	}

	if (!planData.MaintenanceDOW.Equal(stateData.MaintenanceDOW) && !planData.MaintenanceDOW.IsUnknown()) ||
		(!planData.MaintenanceTime.Equal(stateData.MaintenanceTime) && !planData.MaintenanceTime.IsUnknown()) {
		service.Maintenance = &struct {
			Dow  oapi.UpdateDbaasServiceGrafanaJSONBodyMaintenanceDow `json:"dow"`
			Time string                                               `json:"time"`
		}{
			Dow:  oapi.UpdateDbaasServiceGrafanaJSONBodyMaintenanceDow(planData.MaintenanceDOW.ValueString()),
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

	stateGrafanaData := &ResourceGrafanaModel{}
	if stateData.Grafana != nil {
		stateGrafanaData = stateData.Grafana
	}
	planGrafanaData := &ResourceGrafanaModel{}
	if planData.Grafana != nil {
		planGrafanaData = planData.Grafana
	}

	if !planGrafanaData.IpFilter.Equal(stateGrafanaData.IpFilter) {
		obj := []string{}
		if len(planGrafanaData.IpFilter.Elements()) > 0 {
			dg := planGrafanaData.IpFilter.ElementsAs(ctx, &obj, false)
			if dg.HasError() {
				diagnostics.Append(dg...)
				return
			}
		}
		service.IpFilter = &obj
		updated = true
	}

	if !planGrafanaData.Settings.Equal(stateGrafanaData.Settings) {
		if planGrafanaData.Settings.ValueString() != "" {
			obj, err := validateSettings(planGrafanaData.Settings.ValueString(), settingsSchema.JSON200.Settings.Grafana)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid Grafana settings: %s", err))
				return
			}
			service.GrafanaSettings = &obj
		}
		updated = true
	}

	if !updated {
		tflog.Info(ctx, "no updates detected", map[string]interface{}{})
		return
	}

	res, err := r.client.UpdateDbaasServiceGrafanaWithResponse(
		ctx,
		oapi.DbaasServiceName(planData.Id.ValueString()),
		service,
	)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service grafana, got error: %s", err))
		return
	}
	if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
		return
	}

	r.readGrafana(ctx, planData, diagnostics)
}
