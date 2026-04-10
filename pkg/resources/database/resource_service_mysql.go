package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	exoscale "github.com/exoscale/egoscale/v2"
	apiv2 "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/egoscale/v2/oapi"

	"github.com/exoscale/terraform-provider-exoscale/pkg/validators"
)

type ResourceMysqlModel struct {
	AdminPassword  types.String `tfsdk:"admin_password"`
	AdminUsername  types.String `tfsdk:"admin_username"`
	BackupSchedule types.String `tfsdk:"backup_schedule"`
	IpFilter       types.Set    `tfsdk:"ip_filter"`
	Settings       types.String `tfsdk:"mysql_settings"`
	Version        types.String `tfsdk:"version"`
	Integrations   types.Set    `tfsdk:"integrations"`
}

var ResourceMysqlSchema = schema.SingleNestedAttribute{
	Optional:            true,
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
			Computed:            true,
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
		"integrations": ResourceDbaasIntegrationsSchema,
	},
}

// createMysql function handles MySQL specific part of database resource creation logic.
// configData carries the raw configuration so we can distinguish "operator
// omitted integrations" (config null) from "operator set integrations to an
// unknown expression" (config non-null but plan unknown).
func (r *ServiceResource) createMysql(ctx context.Context, data *ServiceResourceModel, configData *ServiceResourceModel, diagnostics *diag.Diagnostics) {
	service := oapi.CreateDbaasServiceMysqlJSONRequestBody{
		Plan:                  data.Plan.ValueString(),
		TerminationProtection: data.TerminationProtection.ValueBoolPointer(),
	}

	if !data.MaintenanceDOW.IsUnknown() && !data.MaintenanceTime.IsUnknown() {
		service.Maintenance = &struct {
			Dow  oapi.CreateDbaasServiceMysqlJSONBodyMaintenanceDow `json:"dow"`
			Time string                                             `json:"time"`
		}{
			Dow:  oapi.CreateDbaasServiceMysqlJSONBodyMaintenanceDow(data.MaintenanceDOW.ValueString()),
			Time: data.MaintenanceTime.ValueString(),
		}
	}

	if data.Mysql != nil {
		if !data.Mysql.Version.IsUnknown() {
			service.Version = data.Mysql.Version.ValueStringPointer()
		}

		if !data.Mysql.AdminPassword.IsNull() {
			service.AdminPassword = data.Mysql.AdminPassword.ValueStringPointer()
		}

		if !data.Mysql.AdminUsername.IsNull() {
			service.AdminUsername = data.Mysql.AdminUsername.ValueStringPointer()
		}

		if !data.Mysql.IpFilter.IsUnknown() {
			obj := []string{}
			if len(data.Mysql.IpFilter.Elements()) > 0 {
				dg := data.Mysql.IpFilter.ElementsAs(ctx, &obj, false)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
			}

			service.IpFilter = &obj
		}

		if !data.Mysql.BackupSchedule.IsUnknown() {
			bh, bm, err := parseBackupSchedule(data.Mysql.BackupSchedule.ValueString())
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

		if !data.Mysql.Settings.IsUnknown() {
			obj, err := validateSettings(data.Mysql.Settings.ValueString(), settingsSchema.JSON200.Settings.Mysql)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			service.MysqlSettings = &obj
		}

		// P2: See createPg for the full rationale. Likely unreachable
		// for the current schema shape but guards against future changes.
		if data.Mysql.Integrations.IsUnknown() && configData.Mysql != nil && !configData.Mysql.Integrations.IsNull() {
			diagnostics.AddError(
				"mysql.integrations: entire integrations set is unknown at create time",
				"The `integrations` attribute was set in configuration but its value "+
					"is still unknown at apply time. This usually means the entire set is "+
					"derived from a resource output that Terraform could not resolve before "+
					"creating this service. Set `integrations` to a list of objects with "+
					"known `type` and `source_service` values, for example:\n\n"+
					"  integrations = [{ type = \"read_replica\", source_service = exoscale_dbaas.<primary>.name }]",
			)
			return
		}

		if !data.Mysql.Integrations.IsNull() && !data.Mysql.Integrations.IsUnknown() {
			var integrationModels []ResourceDbaasIntegrationModel
			// allowUnhandled=true so unknown per-field values in the
			// set (e.g. source_service pointing at a computed
			// attribute of another resource that is unknown at
			// plan time) do not produce a hard decoding error.
			// See resource_service_pg.go for the full rationale.
			if dg := data.Mysql.Integrations.ElementsAs(ctx, &integrationModels, true); dg.HasError() {
				diagnostics.Append(dg...)
				return
			}
			// Single-pass validation: null/unknown rejection + semantic
			// re-checks. See createPg for the full rationale.
			selfName := data.Name.ValueString()
			for i, integration := range integrationModels {
				if integration.Type.IsNull() || integration.Type.IsUnknown() {
					diagnostics.AddError(
						"mysql.integrations: element type is unknown or null at create time",
						fmt.Sprintf(
							"Element %d of the `mysql.integrations` set has no concrete "+
								"`type` value at the time the service is being created. "+
								"This usually means `type` depends on a computed value "+
								"that Terraform could not resolve before the service create "+
								"call. Set `type` to a literal value (currently only "+
								"\"read_replica\" is supported).",
							i,
						),
					)
					return
				}
				if integration.SourceService.IsNull() || integration.SourceService.IsUnknown() {
					diagnostics.AddError(
						"mysql.integrations: source_service is unknown or null at create time",
						fmt.Sprintf(
							"Element %d of the `mysql.integrations` set has no concrete "+
								"`source_service` value at the time the service is being "+
								"created. This usually means `source_service` references a "+
								"value that Terraform could not resolve before the create "+
								"call — for example, a primary service whose name is itself "+
								"computed from another resource that has not been applied "+
								"yet. Ensure `source_service` references a plan-time-known "+
								"value, typically `exoscale_dbaas.<primary>.name` where the "+
								"primary's name is a literal or a fully-resolved expression.",
							i,
						),
					)
					return
				}
				typeVal := integration.Type.ValueString()
				supported := false
				for _, t := range supportedIntegrationTypes {
					if typeVal == t {
						supported = true
						break
					}
				}
				if !supported {
					diagnostics.AddError(
						"mysql.integrations: unsupported integration type",
						fmt.Sprintf(
							"Element %d of the `mysql.integrations` set has type %q, "+
								"which is not a supported integration type. "+
								"Supported types: %s.",
							i, typeVal, strings.Join(supportedIntegrationTypes, ", "),
						),
					)
					return
				}
				if integration.SourceService.ValueString() == selfName {
					diagnostics.AddError(
						"mysql.integrations: self-referencing source_service",
						fmt.Sprintf(
							"Element %d of the `mysql.integrations` set has "+
								"source_service=%q, which is the same service this "+
								"resource declares. Integrations must be declared on "+
								"the destination side, with source_service pointing "+
								"at a different service.",
							i, selfName,
						),
					)
					return
				}
			}
			if len(integrationModels) > 0 {
				integrations := make([]struct {
					DestService   *oapi.DbaasServiceName                               `json:"dest-service,omitempty"`
					Settings      *map[string]interface{}                              `json:"settings,omitempty"`
					SourceService *oapi.DbaasServiceName                               `json:"source-service,omitempty"`
					Type          oapi.CreateDbaasServiceMysqlJSONBodyIntegrationsType `json:"type"`
				}, len(integrationModels))
				for i, integration := range integrationModels {
					source := oapi.DbaasServiceName(integration.SourceService.ValueString())
					integrations[i].SourceService = &source
					integrations[i].Type = oapi.CreateDbaasServiceMysqlJSONBodyIntegrationsType(integration.Type.ValueString())
				}
				service.Integrations = &integrations
			}
		}
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
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database settings schema, unexpected status: %s", res.Status()))
		return
	}

	// The service now exists on Exoscale. Any error return after
	// this point must delete it first — otherwise we orphan a
	// billable database service that Terraform state never knew
	// about. See createPg for the full rationale.
	createSucceeded := false
	defer func() {
		if createSucceeded {
			return
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		cleanupCtx = apiv2.WithEndpoint(cleanupCtx, apiv2.NewReqEndpoint(r.env, data.Zone.ValueString()))
		name := data.Id.ValueString()
		if delErr := r.client.DeleteDatabaseService(
			cleanupCtx,
			data.Zone.ValueString(),
			&exoscale.DatabaseService{Name: &name},
		); delErr != nil {
			tflog.Warn(ctx, fmt.Sprintf(
				"orphan cleanup failed after createMysql error for service %q: %v",
				name, delErr,
			))
		} else {
			tflog.Info(ctx, fmt.Sprintf(
				"orphan cleanup: deleted partially-created mysql service %q",
				name,
			))
		}
	}()

	tflog.Info(ctx, "DB Service created, waiting for the service to be in 'running' state")

	apiService := &oapi.DbaasServiceMysql{}
pooling:
	for {
		select {
		case <-ctx.Done():
			diagnostics.AddError("Error", ctx.Err().Error())
			return
		default:
			res, err := r.client.GetDbaasServiceMysqlWithResponse(ctx, oapi.DbaasServiceName(data.Id.ValueString()))
			if err != nil {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service mysql, got error: %s", err))
				return
			}
			if res.StatusCode() != http.StatusOK {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service mysql, unexpected status: %s", res.Status()))
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

	if data.Mysql.BackupSchedule.IsUnknown() {
		data.Mysql.BackupSchedule = types.StringNull()
		if apiService.BackupSchedule != nil {
			backupHour := types.Int64PointerValue(apiService.BackupSchedule.BackupHour)
			backupMinute := types.Int64PointerValue(apiService.BackupSchedule.BackupMinute)
			data.Mysql.BackupSchedule = types.StringValue(fmt.Sprintf(
				"%02d:%02d",
				backupHour.ValueInt64(),
				backupMinute.ValueInt64(),
			))
		}
	}

	if data.Mysql.IpFilter.IsUnknown() {
		data.Mysql.IpFilter = types.SetNull(types.StringType)
		if apiService.IpFilter != nil {
			v, dg := types.SetValueFrom(ctx, types.StringType, *apiService.IpFilter)
			if dg.HasError() {
				diagnostics.Append(dg...)
				return
			}

			data.Mysql.IpFilter = v
		}
	}

	if data.Mysql.Version.IsUnknown() {
		data.Mysql.Version = types.StringNull()
		if apiService.Version != nil {
			data.Mysql.Version = types.StringValue(strings.SplitN(*apiService.Version, ".", 2)[0])
		}
	}

	if data.Mysql.Settings.IsUnknown() {
		data.Mysql.Settings = types.StringNull()
		if apiService.MysqlSettings != nil {
			settings, err := json.Marshal(*apiService.MysqlSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			data.Mysql.Settings = types.StringValue(string(settings))
		}
	}

	// Integrations is Optional+Computed: if the operator did not set
	// the attribute in config, the framework left the plan value
	// unknown at the Create phase. Populate it from the API response
	// so the post-create state has a concrete value. Only surface
	// integrations where the current service is the destination,
	// matching readMysql's filter.
	if data.Mysql.Integrations.IsUnknown() {
		data.Mysql.Integrations = types.SetNull(resourceDbaasIntegrationObjectType)
		if apiService.Integrations != nil {
			var models []ResourceDbaasIntegrationModel
			for _, integration := range *apiService.Integrations {
				if integration.Dest == nil || *integration.Dest != data.Id.ValueString() {
					continue
				}
				models = append(models, ResourceDbaasIntegrationModel{
					Type:          types.StringPointerValue(integration.Type),
					SourceService: types.StringPointerValue(integration.Source),
				})
			}
			if len(models) > 0 {
				v, dg := types.SetValueFrom(ctx, resourceDbaasIntegrationObjectType, models)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
				data.Mysql.Integrations = v
			}
		}
	}

	// All post-create population succeeded — cancel the deferred
	// orphan cleanup.
	createSucceeded = true
}

// readMysql function handles MySQL specific part of database resource Read logic.
func (r *ServiceResource) readMysql(ctx context.Context, data *ServiceResourceModel, diagnostics *diag.Diagnostics) (clearState bool) {

	res, err := r.client.GetDbaasServiceMysqlWithResponse(ctx, oapi.DbaasServiceName(data.Id.ValueString()))
	if err != nil {
		if errors.Is(err, apiv2.ErrNotFound) {
			return true
		}
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service mysql, got error: %s", err))
		return false
	}
	if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service mysql, unexpected status: %s", res.Status()))
		return false
	}
	apiService := res.JSON200

	caCert, err := r.client.GetDatabaseCACertificate(ctx, data.Zone.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get CA Certificate: %s", err))
		return false
	}
	data.CA = types.StringValue(caCert)

	data.CreatedAt = types.StringValue(apiService.CreatedAt.String())
	data.DiskSize = types.Int64PointerValue(apiService.DiskSize)
	data.NodeCPUs = types.Int64PointerValue(apiService.NodeCpuCount)
	data.NodeMemory = types.Int64PointerValue(apiService.NodeMemory)
	data.Nodes = types.Int64PointerValue(apiService.NodeCount)
	data.State = types.StringPointerValue((*string)(apiService.State))
	data.TerminationProtection = types.BoolPointerValue(apiService.TerminationProtection)
	data.UpdatedAt = types.StringValue(apiService.UpdatedAt.String())

	data.MaintenanceDOW = types.StringNull()
	data.MaintenanceTime = types.StringNull()
	if apiService.Maintenance != nil {
		data.MaintenanceDOW = types.StringValue(string(apiService.Maintenance.Dow))
		data.MaintenanceTime = types.StringValue(apiService.Maintenance.Time)
	}

	if data.Mysql == nil {
		data.Mysql = &ResourceMysqlModel{}
	}

	data.Mysql.BackupSchedule = types.StringNull()
	if apiService.BackupSchedule != nil {
		backupHour := types.Int64PointerValue(apiService.BackupSchedule.BackupHour)
		backupMinute := types.Int64PointerValue(apiService.BackupSchedule.BackupMinute)
		data.Mysql.BackupSchedule = types.StringValue(fmt.Sprintf(
			"%02d:%02d",
			backupHour.ValueInt64(),
			backupMinute.ValueInt64(),
		))
	}

	data.Mysql.IpFilter = types.SetNull(types.StringType)
	if apiService.IpFilter != nil {
		v, dg := types.SetValueFrom(ctx, types.StringType, *apiService.IpFilter)
		if dg.HasError() {
			diagnostics.Append(dg...)
			return false
		}

		data.Mysql.IpFilter = v
	}

	data.Mysql.Version = types.StringNull()
	if apiService.Version != nil {
		data.Mysql.Version = types.StringValue(strings.SplitN(*apiService.Version, ".", 2)[0])
	}

	data.Mysql.Settings = types.StringNull()
	if apiService.MysqlSettings != nil {
		settings, err := json.Marshal(*apiService.MysqlSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return false
		}
		data.Mysql.Settings = types.StringValue(string(settings))
	}

	// Only surface integrations where the current service is the destination,
	// so that Terraform state reflects exactly what the resource's config
	// declares (i.e. integrations the user asked to create for this service).
	data.Mysql.Integrations = types.SetNull(resourceDbaasIntegrationObjectType)
	if apiService.Integrations != nil {
		var integrationModels []ResourceDbaasIntegrationModel
		for _, integration := range *apiService.Integrations {
			if integration.Dest == nil || *integration.Dest != data.Id.ValueString() {
				continue
			}
			integrationModels = append(integrationModels, ResourceDbaasIntegrationModel{
				Type:          types.StringPointerValue(integration.Type),
				SourceService: types.StringPointerValue(integration.Source),
			})
		}
		if len(integrationModels) > 0 {
			v, dg := types.SetValueFrom(ctx, resourceDbaasIntegrationObjectType, integrationModels)
			if dg.HasError() {
				diagnostics.Append(dg...)
				return false
			}
			data.Mysql.Integrations = v
		}
	}

	return false
}

// updateMysql function handles MySQL specific part of database resource Update logic.
func (r *ServiceResource) updateMysql(ctx context.Context, stateData *ServiceResourceModel, planData *ServiceResourceModel, diagnostics *diag.Diagnostics) {
	var updated bool

	service := oapi.UpdateDbaasServiceMysqlJSONRequestBody{}

	if (!planData.MaintenanceDOW.Equal(stateData.MaintenanceDOW) && !planData.MaintenanceDOW.IsUnknown()) ||
		(!planData.MaintenanceTime.Equal(stateData.MaintenanceTime) && !planData.MaintenanceTime.IsUnknown()) {
		service.Maintenance = &struct {
			Dow  oapi.UpdateDbaasServiceMysqlJSONBodyMaintenanceDow `json:"dow"`
			Time string                                             `json:"time"`
		}{
			Dow:  oapi.UpdateDbaasServiceMysqlJSONBodyMaintenanceDow(planData.MaintenanceDOW.ValueString()),
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

	if planData.Mysql != nil {
		if stateData.Mysql == nil {
			stateData.Mysql = &ResourceMysqlModel{}
		}

		if !planData.Mysql.BackupSchedule.IsUnknown() && !planData.Mysql.BackupSchedule.Equal(stateData.Mysql.BackupSchedule) {
			bh, bm, err := parseBackupSchedule(planData.Mysql.BackupSchedule.ValueString())
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
			stateData.Mysql.BackupSchedule = planData.Mysql.BackupSchedule
			updated = true
		}

		if !planData.Mysql.IpFilter.Equal(stateData.Mysql.IpFilter) {
			obj := []string{}
			if len(planData.Mysql.IpFilter.Elements()) > 0 {
				dg := planData.Mysql.IpFilter.ElementsAs(ctx, &obj, false)
				if dg.HasError() {
					diagnostics.Append(dg...)
					return
				}
			}
			service.IpFilter = &obj
			stateData.Mysql.IpFilter = planData.Mysql.IpFilter
			updated = true
		}

		if !planData.Mysql.Settings.Equal(stateData.Mysql.Settings) {
			settingsSchema, err := r.client.GetDbaasSettingsMysqlWithResponse(ctx)
			if err != nil {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, got error: %s", err))
				return
			}
			if settingsSchema.StatusCode() != http.StatusOK {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
				return
			}

			if planData.Mysql.Settings.ValueString() != "" {
				obj, err := validateSettings(planData.Mysql.Settings.ValueString(), settingsSchema.JSON200.Settings.Mysql)
				if err != nil {
					diagnostics.AddError("Validation error", fmt.Sprintf("invalid Mysql settings: %s", err))
					return
				}
				service.MysqlSettings = &obj
			}
			stateData.Mysql.Settings = planData.Mysql.Settings
			updated = true
		}

		// Defensive no-op: integrations is Optional with a RequiresReplace
		// plan modifier, so Terraform Core never calls Update when the
		// set differs from state. Keep stateData in sync with plan here
		// as a safety net in case that contract is ever weakened.
		stateData.Mysql.Integrations = planData.Mysql.Integrations
	}

	if !updated {
		tflog.Info(ctx, "no updates detected", map[string]interface{}{})
		return
	}

	// Aiven would overwrite the backup schedule with random value if we don't specify it explicitly every time.
	if service.BackupSchedule == nil && !stateData.Mysql.BackupSchedule.IsUnknown() {
		bh, bm, err := parseBackupSchedule(stateData.Mysql.BackupSchedule.ValueString())
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
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service mysql, unexpected status: %s", res.Status()))
		return
	}

	apiService := &oapi.DbaasServiceMysql{}
	if res, err := r.client.GetDbaasServiceMysqlWithResponse(
		ctx,
		oapi.DbaasServiceName(stateData.Id.ValueString()),
	); err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service mysql, got error: %s", err))
		return
	} else if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service mysql, unexpected status: %s", res.Status()))
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

	if stateData.Mysql == nil {
		return
	}

	if stateData.Mysql.IpFilter.IsUnknown() {
		stateData.Mysql.IpFilter = types.SetNull(types.StringType)
		if apiService.IpFilter != nil {
			v, dg := types.SetValueFrom(ctx, types.StringType, *apiService.IpFilter)
			if dg.HasError() {
				diagnostics.Append(dg...)
				return
			}
			stateData.Mysql.IpFilter = v
		}
	}
	if stateData.Mysql.Version.IsUnknown() {
		stateData.Mysql.Version = types.StringNull()
		if apiService.Version != nil {
			stateData.Mysql.Version = types.StringValue(strings.SplitN(*apiService.Version, ".", 2)[0])
		}
	}

	if stateData.Mysql.Settings.IsUnknown() {
		stateData.Mysql.Settings = types.StringNull()
		if apiService.MysqlSettings != nil {
			settings, err := json.Marshal(*apiService.MysqlSettings)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
				return
			}
			stateData.Mysql.Settings = types.StringValue(string(settings))
		}
	}
}
