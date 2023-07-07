package database

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/exoscale/egoscale/v2/oapi"

	"github.com/exoscale/terraform-provider-exoscale/pkg/validators"
)

type ResourceKafkaModel struct {
	EnableCertAuth         types.Bool   `tfsdk:"enable_cert_auth"`
	EnableKafkaConnect     types.Bool   `tfsdk:"enable_kafka_connect"`
	EnableKafkaREST        types.Bool   `tfsdk:"enable_kafka_rest"`
	EnableSASLAuth         types.Bool   `tfsdk:"enable_sasl_auth"`
	EnableSchemaRegistry   types.Bool   `tfsdk:"enable_schema_registry"`
	IpFilter               types.Set    `tfsdk:"ip_filter"`
	Settings               types.String `tfsdk:"kafka_settings"`
	ConnectSettings        types.String `tfsdk:"kafka_connect_settings"`
	RestSettings           types.String `tfsdk:"kafka_rest_settings"`
	SchemaRegistrySettings types.String `tfsdk:"schema_registry_settings"`
	Version                types.String `tfsdk:"version"`
}

var ResourceKafkaSchema = schema.SingleNestedBlock{
	MarkdownDescription: "*kafka* database service type specific arguments. Structure is documented below.",
	Attributes: map[string]schema.Attribute{
		"enable_cert_auth": schema.BoolAttribute{
			MarkdownDescription: "Enable certificate-based authentication method.",
			Optional:            true,
			Computed:            true,
		},
		"enable_kafka_connect": schema.BoolAttribute{
			MarkdownDescription: "Enable Kafka Connect.",
			Optional:            true,
			Computed:            true,
		},
		"enable_kafka_rest": schema.BoolAttribute{
			MarkdownDescription: "Enable Kafka REST.",
			Optional:            true,
			Computed:            true,
		},
		"enable_sasl_auth": schema.BoolAttribute{
			MarkdownDescription: "Enable SASL-based authentication method.",
			Optional:            true,
			Computed:            true,
		},
		"enable_schema_registry": schema.BoolAttribute{
			MarkdownDescription: "Enable Schema Registry.",
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
		"kafka_settings": schema.StringAttribute{
			MarkdownDescription: "Kafka configuration settings in JSON format (`exo dbaas type show kafka --settings=kafka` for reference).",
			Optional:            true,
			Computed:            true,
		},
		"kafka_connect_settings": schema.StringAttribute{
			MarkdownDescription: "Kafka Connect configuration settings in JSON format (`exo dbaas type show kafka --settings=kafka-connect` for reference).",
			Optional:            true,
			Computed:            true,
		},
		"kafka_rest_settings": schema.StringAttribute{
			MarkdownDescription: "Kafka REST configuration settings in JSON format (`exo dbaas type show kafka --settings=kafka-rest` for reference).",
			Optional:            true,
			Computed:            true,
		},
		"schema_registry_settings": schema.StringAttribute{
			MarkdownDescription: "Schema Registry configuration settings in JSON format (`exo dbaas type show kafka --settings=schema-registry` for reference)",
			Optional:            true,
			Computed:            true,
		},
		"version": schema.StringAttribute{
			MarkdownDescription: "Kafka major version (`exo dbaas type show kafka` for reference; may only be set at creation time).",
			Optional:            true,
			Computed:            true,
		},
	},
}

// createKafka function handles Kafka specific part of database resource creation logic.
func (r *Resource) createKafka(ctx context.Context, data *ResourceModel, diagnostics *diag.Diagnostics) {
	kafkaData := &ResourceKafkaModel{}
	if data.Kafka != nil {
		kafkaData = data.Kafka
	}

	service := oapi.CreateDbaasServiceKafkaJSONRequestBody{
		Plan:                  data.Plan.ValueString(),
		TerminationProtection: data.TerminationProtection.ValueBoolPointer(),
		Version:               kafkaData.Version.ValueStringPointer(),
		KafkaConnectEnabled:   kafkaData.EnableKafkaConnect.ValueBoolPointer(),
		KafkaRestEnabled:      kafkaData.EnableKafkaREST.ValueBoolPointer(),
		SchemaRegistryEnabled: kafkaData.EnableSchemaRegistry.ValueBoolPointer(),
	}

	if kafkaData.EnableCertAuth.ValueBool() || kafkaData.EnableSASLAuth.ValueBool() {
		service.AuthenticationMethods = &struct {
			Certificate *bool `json:"certificate,omitempty"`
			Sasl        *bool `json:"sasl,omitempty"`
		}{
			Certificate: kafkaData.EnableCertAuth.ValueBoolPointer(),
			Sasl:        kafkaData.EnableSASLAuth.ValueBoolPointer(),
		}
	}

	if len(kafkaData.IpFilter.Elements()) > 0 {
		obj := []string{}
		dg := kafkaData.IpFilter.ElementsAs(ctx, &obj, false)
		if dg.HasError() {
			diagnostics.Append(dg...)
			return
		}

		service.IpFilter = &obj
	}

	if data.MaintenanceDOW.ValueString() != "" && data.MaintenanceTime.ValueString() != "" {
		service.Maintenance = &struct {
			Dow  oapi.CreateDbaasServiceKafkaJSONBodyMaintenanceDow `json:"dow"`
			Time string                                             `json:"time"`
		}{
			Dow:  oapi.CreateDbaasServiceKafkaJSONBodyMaintenanceDow(data.MaintenanceDOW.ValueString()),
			Time: data.MaintenanceTime.ValueString(),
		}
	}

	settingsSchema, err := r.client.GetDbaasSettingsKafkaWithResponse(ctx)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, got error: %s", err))
		return
	}
	if settingsSchema.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
		return
	}

	settings := kafkaData.Settings.ValueString()
	if settings != "" {
		obj, err := validateSettings(settings, settingsSchema.JSON200.Settings.Kafka)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return
		}
		service.KafkaSettings = &obj
	}

	connectSettings := kafkaData.ConnectSettings.ValueString()
	if connectSettings != "" {
		obj, err := validateSettings(connectSettings, settingsSchema.JSON200.Settings.KafkaConnect)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid Kafka Connect settings: %s", err))
			return
		}
		service.KafkaConnectSettings = &obj
	}

	restSettings := kafkaData.RestSettings.ValueString()
	if restSettings != "" {
		obj, err := validateSettings(restSettings, settingsSchema.JSON200.Settings.KafkaRest)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid Kafka REST settings: %s", err))
			return
		}
		service.KafkaRestSettings = &obj
	}

	schemaRegistrySettings := kafkaData.SchemaRegistrySettings.ValueString()
	if schemaRegistrySettings != "" {
		obj, err := validateSettings(schemaRegistrySettings, settingsSchema.JSON200.Settings.SchemaRegistry)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid Schema Registry settings: %s", err))
			return
		}
		service.SchemaRegistrySettings = &obj
	}

	res, err := r.client.CreateDbaasServiceKafkaWithResponse(
		ctx,
		oapi.DbaasServiceName(data.Name.ValueString()),
		service,
	)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service kafka, got error: %s", err))
		return
	}
	if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
		return
	}

	r.readKafka(ctx, data, diagnostics)
}

// readKafka function handles Kafka specific part of database resource Read logic.
// It is used in the dedicated Read action but also as a finishing step of Create, Update and Import.
func (r *Resource) readKafka(ctx context.Context, data *ResourceModel, diagnostics *diag.Diagnostics) {
	caCert, err := r.client.GetDatabaseCACertificate(context.Background(), data.Zone.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get CA Certificate: %s", err))
		return
	}
	data.CA = types.StringValue(caCert)

	res, err := r.client.GetDbaasServiceKafkaWithResponse(ctx, oapi.DbaasServiceName(data.Id.ValueString()))
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service kafka, got error: %s", err))
		return
	}
	if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database service kafka, unexpected status: %s", res.Status()))
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

	if data.Kafka == nil {
		data.Kafka = &ResourceKafkaModel{}
	}

	data.Kafka.EnableKafkaConnect = types.BoolPointerValue(apiService.KafkaConnectEnabled)
	data.Kafka.EnableKafkaREST = types.BoolPointerValue(apiService.KafkaRestEnabled)
	data.Kafka.EnableSchemaRegistry = types.BoolPointerValue(apiService.SchemaRegistryEnabled)

	if apiService.AuthenticationMethods != nil {
		data.Kafka.EnableSASLAuth = types.BoolPointerValue(apiService.AuthenticationMethods.Sasl)
		data.Kafka.EnableCertAuth = types.BoolPointerValue(apiService.AuthenticationMethods.Certificate)
	}

	if apiService.IpFilter == nil || len(*apiService.IpFilter) == 0 {
		data.Kafka.IpFilter = types.SetNull(types.StringType)
	} else {
		v, dg := types.SetValueFrom(ctx, types.StringType, *apiService.IpFilter)
		if dg.HasError() {
			diagnostics.Append(dg...)
			return
		}

		data.Kafka.IpFilter = v
	}

	if apiService.Version == nil {
		data.Kafka.Version = types.StringNull()
	} else {
		version := strings.SplitN(*apiService.Version, ".", 3)
		data.Kafka.Version = types.StringValue(version[0] + "." + version[1])
	}

	if apiService.KafkaSettings == nil {
		data.Kafka.Settings = types.StringNull()
	} else {
		settings, err := json.Marshal(*apiService.KafkaSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid settings: %s", err))
			return
		}
		data.Kafka.Settings = types.StringValue(string(settings))
	}

	if apiService.KafkaConnectSettings == nil {
		data.Kafka.ConnectSettings = types.StringNull()
	} else {
		settings, err := json.Marshal(*apiService.KafkaConnectSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid Kafka Connect settings: %s", err))
			return
		}
		data.Kafka.ConnectSettings = types.StringValue(string(settings))
	}

	if apiService.KafkaRestSettings == nil {
		data.Kafka.RestSettings = types.StringNull()
	} else {
		settings, err := json.Marshal(*apiService.KafkaRestSettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid Kafka REST settings: %s", err))
			return
		}
		data.Kafka.RestSettings = types.StringValue(string(settings))
	}

	if apiService.SchemaRegistrySettings == nil {
		data.Kafka.SchemaRegistrySettings = types.StringNull()
	} else {
		settings, err := json.Marshal(*apiService.SchemaRegistrySettings)
		if err != nil {
			diagnostics.AddError("Validation error", fmt.Sprintf("invalid Schema Registry settings: %s", err))
			return
		}
		data.Kafka.SchemaRegistrySettings = types.StringValue(string(settings))
	}
}

// updateKafka function handles Kafka specific part of database resource Update logic.
func (r *Resource) updateKafka(ctx context.Context, stateData *ResourceModel, planData *ResourceModel, diagnostics *diag.Diagnostics) {
	var updated bool

	service := oapi.UpdateDbaasServiceKafkaJSONRequestBody{}

	settingsSchema, err := r.client.GetDbaasSettingsKafkaWithResponse(ctx)
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
			Dow  oapi.UpdateDbaasServiceKafkaJSONBodyMaintenanceDow `json:"dow"`
			Time string                                             `json:"time"`
		}{
			Dow:  oapi.UpdateDbaasServiceKafkaJSONBodyMaintenanceDow(planData.MaintenanceDOW.ValueString()),
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

	stateKafkaData := &ResourceKafkaModel{}
	if stateData.Kafka != nil {
		stateKafkaData = stateData.Kafka
	}
	planKafkaData := &ResourceKafkaModel{}
	if planData.Kafka != nil {
		planKafkaData = planData.Kafka
	}

	if !planKafkaData.IpFilter.Equal(stateKafkaData.IpFilter) {
		obj := []string{}
		if len(planKafkaData.IpFilter.Elements()) > 0 {
			dg := planKafkaData.IpFilter.ElementsAs(ctx, &obj, false)
			if dg.HasError() {
				diagnostics.Append(dg...)
				return
			}
		}
		service.IpFilter = &obj
		updated = true
	}

	if !planKafkaData.EnableKafkaConnect.Equal(stateKafkaData.EnableKafkaConnect) {
		service.KafkaConnectEnabled = planKafkaData.EnableKafkaConnect.ValueBoolPointer()
		updated = true
	}

	if !planKafkaData.EnableKafkaREST.Equal(stateKafkaData.EnableKafkaREST) {
		service.KafkaRestEnabled = planKafkaData.EnableKafkaREST.ValueBoolPointer()
		updated = true
	}

	if !planKafkaData.EnableSchemaRegistry.Equal(stateKafkaData.EnableSchemaRegistry) {
		service.SchemaRegistryEnabled = planKafkaData.EnableSchemaRegistry.ValueBoolPointer()
		updated = true
	}

	if !planKafkaData.EnableCertAuth.Equal(stateKafkaData.EnableCertAuth) || !planKafkaData.EnableSASLAuth.Equal(stateKafkaData.EnableSASLAuth) {
		service.AuthenticationMethods = &struct {
			Certificate *bool `json:"certificate,omitempty"`
			Sasl        *bool `json:"sasl,omitempty"`
		}{
			Certificate: planKafkaData.EnableCertAuth.ValueBoolPointer(),
			Sasl:        planKafkaData.EnableSASLAuth.ValueBoolPointer(),
		}
		updated = true
	}

	if !planKafkaData.Settings.Equal(stateKafkaData.Settings) {
		if planKafkaData.Settings.ValueString() != "" {
			obj, err := validateSettings(planKafkaData.Settings.ValueString(), settingsSchema.JSON200.Settings.Kafka)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid Kafka settings: %s", err))
				return
			}
			service.KafkaSettings = &obj
		}
		updated = true
	}

	if !planKafkaData.ConnectSettings.Equal(stateKafkaData.ConnectSettings) {
		if planKafkaData.ConnectSettings.ValueString() != "" {
			obj, err := validateSettings(planKafkaData.ConnectSettings.ValueString(), settingsSchema.JSON200.Settings.KafkaConnect)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid Kafka Connect settings: %s", err))
				return
			}
			service.KafkaConnectSettings = &obj
		}
		updated = true
	}

	if !planKafkaData.RestSettings.Equal(stateKafkaData.RestSettings) {
		if planKafkaData.RestSettings.ValueString() != "" {
			obj, err := validateSettings(planKafkaData.RestSettings.ValueString(), settingsSchema.JSON200.Settings.KafkaRest)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid Kafka settings: %s", err))
				return
			}
			service.KafkaRestSettings = &obj
		}
		updated = true
	}

	if !planKafkaData.SchemaRegistrySettings.Equal(stateKafkaData.SchemaRegistrySettings) {
		if planKafkaData.SchemaRegistrySettings.ValueString() != "" {
			obj, err := validateSettings(planKafkaData.SchemaRegistrySettings.ValueString(), settingsSchema.JSON200.Settings.SchemaRegistry)
			if err != nil {
				diagnostics.AddError("Validation error", fmt.Sprintf("invalid Kafka settings: %s", err))
				return
			}
			service.SchemaRegistrySettings = &obj
		}
		updated = true
	}

	if !updated {
		tflog.Info(ctx, "no updates detected", map[string]interface{}{})
		return
	}

	res, err := r.client.UpdateDbaasServiceKafkaWithResponse(
		ctx,
		oapi.DbaasServiceName(planData.Id.ValueString()),
		service,
	)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create database service kafka, got error: %s", err))
		return
	}
	if res.StatusCode() != http.StatusOK {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read database settings schema, unexpected status: %s", settingsSchema.Status()))
		return
	}

	r.readKafka(ctx, planData, diagnostics)
}
