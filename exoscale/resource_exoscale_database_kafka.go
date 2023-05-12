package exoscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	egoscale "github.com/exoscale/egoscale/v2"
	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	resDatabaseAttrKafkaConnectSettings        = "kafka_connect_settings"
	resDatabaseAttrKafkaEnableCertAuth         = "enable_cert_auth"
	resDatabaseAttrKafkaEnableKafkaConnect     = "enable_kafka_connect"
	resDatabaseAttrKafkaEnableKafkaREST        = "enable_kafka_rest"
	resDatabaseAttrKafkaEnableSASLAuth         = "enable_sasl_auth"
	resDatabaseAttrKafkaEnableSchemaRegistry   = "enable_schema_registry"
	resDatabaseAttrKafkaIPFilter               = "ip_filter"
	resDatabaseAttrKafkaRESTSettings           = "kafka_rest_settings"
	resDatabaseAttrKafkaSchemaRegistrySettings = "schema_registry_settings"
	resDatabaseAttrKafkaSettings               = "kafka_settings"
	resDatabaseAttrKafkaVersion                = "version"
)

var resDatabaseKafkaSchema = &schema.Schema{
	Type:        schema.TypeList,
	MaxItems:    1,
	Optional:    true,
	Description: "*kafka* database service type specific arguments.",
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			resDatabaseAttrKafkaConnectSettings: {
				Description: "Kafka Connect configuration settings in JSON format (`exo dbaas type show kafka --settings=kafka-connect` for reference).",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			resDatabaseAttrKafkaEnableCertAuth: {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Enable certificate-based authentication method.",
			},
			resDatabaseAttrKafkaEnableKafkaConnect: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Enable Kafka Connect.",
			},
			resDatabaseAttrKafkaEnableKafkaREST: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Enable Kafka REST.",
			},
			resDatabaseAttrKafkaEnableSASLAuth: {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "Enable SASL-based authentication method.",
			},
			resDatabaseAttrKafkaEnableSchemaRegistry: {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Enable Schema Registry.",
			},
			resDatabaseAttrKafkaIPFilter: {
				Description: "A list of CIDR blocks to allow incoming connections from.",
				Type:        schema.TypeSet,
				Set:         schema.HashString,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsCIDRNetwork(0, 128),
				},
				Optional: true,
				Computed: true,
			},
			resDatabaseAttrKafkaRESTSettings: {
				Description: "Kafka REST configuration settings in JSON format (`exo dbaas type show kafka --settings=kafka-rest` for reference).",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			resDatabaseAttrKafkaSchemaRegistrySettings: {
				Description: "Schema Registry configuration settings in JSON format (`exo dbaas type show kafka --settings=schema-registry` for reference)",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			resDatabaseAttrKafkaSettings: {
				Description: "Kafka configuration settings in JSON format (`exo dbaas type show kafka --settings=kafka` for reference).",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
			resDatabaseAttrKafkaVersion: {
				Description: "Kafka major version (`exo dbaas type show kafka` for reference; may only be set at creation time).",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
		},
	},
}

func resourceDatabaseCreateKafka(
	ctx context.Context,
	d *schema.ResourceData,
	client *egoscale.Client,
) diag.Diagnostics {
	databaseService := oapi.CreateDbaasServiceKafkaJSONRequestBody{
		Plan: d.Get(resDatabaseAttrPlan).(string),
	}

	settingsSchema, err := client.GetDbaasSettingsKafkaWithResponse(ctx)
	if err != nil {
		return diag.Errorf("unable to retrieve Database Service settings: %v", err)
	}
	if settingsSchema.StatusCode() != http.StatusOK {
		return diag.Errorf("API request error: unexpected status %s", settingsSchema.Status())
	}

	maintenanceDOW := d.Get(resDatabaseAttrMaintenanceDOW).(string)
	maintenanceTime := d.Get(resDatabaseAttrMaintenanceTime).(string)
	if maintenanceDOW != "" && maintenanceTime != "" {
		databaseService.Maintenance = &struct {
			Dow  oapi.CreateDbaasServiceKafkaJSONBodyMaintenanceDow `json:"dow"`
			Time string                                             `json:"time"`
		}{
			Dow:  oapi.CreateDbaasServiceKafkaJSONBodyMaintenanceDow(maintenanceDOW),
			Time: maintenanceTime,
		}
	}

	if v, ok := d.GetOk(resDatabaseAttrTerminationProtection); ok {
		b := v.(bool)
		databaseService.TerminationProtection = &b
	}

	if v, ok := d.GetOk(resDatabaseAttrKafka(resDatabaseAttrKafkaConnectSettings)); ok {
		settings, err := validateDatabaseServiceSettings(v.(string), settingsSchema.JSON200.Settings.KafkaConnect)
		if err != nil {
			return diag.Errorf("invalid settings: %v", err)
		}
		databaseService.KafkaConnectSettings = &settings
	}

	_, enableCertAuth := d.GetOk(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableCertAuth))
	_, enableSASLAuth := d.GetOk(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableSASLAuth))
	if enableCertAuth || enableSASLAuth {
		databaseService.AuthenticationMethods = &struct {
			Certificate *bool `json:"certificate,omitempty"`
			Sasl        *bool `json:"sasl,omitempty"`
		}{}
		if enableCertAuth {
			enabled := d.Get(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableCertAuth)).(bool)
			databaseService.AuthenticationMethods.Certificate = &enabled
		}
		if enableSASLAuth {
			enabled := d.Get(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableSASLAuth)).(bool)
			databaseService.AuthenticationMethods.Sasl = &enabled
		}
	}

	if v, ok := d.GetOk(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableKafkaConnect)); ok {
		enabled := v.(bool)
		databaseService.KafkaConnectEnabled = &enabled
	}

	if v, ok := d.GetOk(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableKafkaREST)); ok {
		enabled := v.(bool)
		databaseService.KafkaRestEnabled = &enabled
	}

	if v, ok := d.GetOk(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableSchemaRegistry)); ok {
		enabled := v.(bool)
		databaseService.SchemaRegistryEnabled = &enabled
	}

	dg := newResourceDataGetter(d)
	dgos := dg.Under("kafka").Under("0")

	databaseService.IpFilter = dgos.GetSet(resDatabaseAttrKafkaIPFilter)

	if v, ok := d.GetOk(resDatabaseAttrKafka(resDatabaseAttrKafkaRESTSettings)); ok {
		settings, err := validateDatabaseServiceSettings(v.(string), settingsSchema.JSON200.Settings.KafkaRest)
		if err != nil {
			return diag.Errorf("invalid settings: %v", err)
		}
		databaseService.KafkaRestSettings = &settings
	}

	if v, ok := d.GetOk(resDatabaseAttrKafka(resDatabaseAttrKafkaSchemaRegistrySettings)); ok {
		settings, err := validateDatabaseServiceSettings(v.(string), settingsSchema.JSON200.Settings.SchemaRegistry)
		if err != nil {
			return diag.Errorf("invalid settings: %v", err)
		}
		databaseService.SchemaRegistrySettings = &settings
	}

	if v, ok := d.GetOk(resDatabaseAttrKafka(resDatabaseAttrKafkaSettings)); ok {
		settings, err := validateDatabaseServiceSettings(v.(string), settingsSchema.JSON200.Settings.Kafka)
		if err != nil {
			return diag.Errorf("invalid settings: %v", err)
		}
		databaseService.KafkaSettings = &settings
	}

	if v, ok := d.GetOk(resDatabaseAttrKafka(resDatabaseAttrKafkaVersion)); ok {
		version := v.(string)
		databaseService.Version = &version
	}

	databaseServiceName := d.Get(resDatabaseAttrName).(string)

	res, err := client.CreateDbaasServiceKafkaWithResponse(
		ctx,
		oapi.DbaasServiceName(databaseServiceName),
		databaseService,
	)
	if err != nil {
		return diag.FromErr(err)
	}
	if res.StatusCode() != http.StatusOK {
		return diag.Errorf("API request error: unexpected status %s", res.Status())
	}

	d.SetId(databaseServiceName)

	return nil
}

func resourceDatabaseUpdateKafka(
	ctx context.Context,
	d *schema.ResourceData,
	client *egoscale.Client,
) diag.Diagnostics {
	var updated bool

	databaseService := oapi.UpdateDbaasServiceKafkaJSONRequestBody{}

	settingsSchema, err := client.GetDbaasSettingsKafkaWithResponse(ctx)
	if err != nil {
		return diag.Errorf("unable to retrieve Database Service settings: %v", err)
	}
	if settingsSchema.StatusCode() != http.StatusOK {
		return diag.Errorf("API request error: unexpected status %s", settingsSchema.Status())
	}

	if d.HasChange(resDatabaseAttrMaintenanceDOW) || d.HasChange(resDatabaseAttrMaintenanceTime) {
		databaseService.Maintenance = &struct {
			Dow  oapi.UpdateDbaasServiceKafkaJSONBodyMaintenanceDow `json:"dow"`
			Time string                                             `json:"time"`
		}{
			Dow:  oapi.UpdateDbaasServiceKafkaJSONBodyMaintenanceDow(d.Get(resDatabaseAttrMaintenanceDOW).(string)),
			Time: d.Get(resDatabaseAttrMaintenanceTime).(string),
		}
		updated = true
	}

	if d.HasChange(resDatabaseAttrPlan) {
		v := d.Get(resDatabaseAttrPlan).(string)
		databaseService.Plan = &v
		updated = true
	}

	if d.HasChange(resDatabaseAttrTerminationProtection) {
		v := d.Get(resDatabaseAttrTerminationProtection).(bool)
		databaseService.TerminationProtection = &v
		updated = true
	}

	if d.HasChange("kafka") {
		if d.HasChange(resDatabaseAttrKafka(resDatabaseAttrKafkaConnectSettings)) {
			settings, err := validateDatabaseServiceSettings(
				d.Get(resDatabaseAttrKafka(resDatabaseAttrKafkaConnectSettings)).(string),
				settingsSchema.JSON200.Settings.KafkaConnect,
			)
			if err != nil {
				return diag.Errorf("invalid settings: %v", err)
			}
			databaseService.KafkaConnectSettings = &settings
			updated = true
		}

		if d.HasChange(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableCertAuth)) ||
			d.HasChange(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableSASLAuth)) {
			databaseService.AuthenticationMethods = &struct {
				Certificate *bool `json:"certificate,omitempty"`
				Sasl        *bool `json:"sasl,omitempty"`
			}{}
			if d.HasChange(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableCertAuth)) {
				v := d.Get(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableCertAuth)).(bool)
				databaseService.AuthenticationMethods.Certificate = &v
			}
			if d.HasChange(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableSASLAuth)) {
				v := d.Get(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableSASLAuth)).(bool)
				databaseService.AuthenticationMethods.Sasl = &v
			}
			updated = true
		}

		if d.HasChange(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableKafkaConnect)) {
			v := d.Get(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableKafkaConnect)).(bool)
			databaseService.KafkaConnectEnabled = &v
			updated = true
		}

		if d.HasChange(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableKafkaREST)) {
			v := d.Get(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableKafkaREST)).(bool)
			databaseService.KafkaRestEnabled = &v
			updated = true
		}

		if d.HasChange(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableSchemaRegistry)) {
			v := d.Get(resDatabaseAttrKafka(resDatabaseAttrKafkaEnableSchemaRegistry)).(bool)
			databaseService.SchemaRegistryEnabled = &v
			updated = true
		}

		if d.HasChange(resDatabaseAttrKafka(resDatabaseAttrKafkaIPFilter)) {
			dg := newResourceDataGetter(d)
			dgos := dg.Under("kafka").Under("0")

			databaseService.IpFilter = dgos.GetSet(resDatabaseAttrKafkaIPFilter)
			updated = true
		}

		if d.HasChange(resDatabaseAttrKafka(resDatabaseAttrKafkaRESTSettings)) {
			settings, err := validateDatabaseServiceSettings(
				d.Get(resDatabaseAttrKafka(resDatabaseAttrKafkaRESTSettings)).(string),
				settingsSchema.JSON200.Settings.KafkaRest,
			)
			if err != nil {
				return diag.Errorf("invalid settings: %v", err)
			}
			databaseService.KafkaRestSettings = &settings
			updated = true
		}

		if d.HasChange(resDatabaseAttrKafka(resDatabaseAttrKafkaSchemaRegistrySettings)) {
			settings, err := validateDatabaseServiceSettings(
				d.Get(resDatabaseAttrKafka(resDatabaseAttrKafkaSchemaRegistrySettings)).(string),
				settingsSchema.JSON200.Settings.SchemaRegistry,
			)
			if err != nil {
				return diag.Errorf("invalid settings: %v", err)
			}
			databaseService.SchemaRegistrySettings = &settings
			updated = true
		}

		if d.HasChange(resDatabaseAttrKafka(resDatabaseAttrKafkaSettings)) {
			settings, err := validateDatabaseServiceSettings(
				d.Get(resDatabaseAttrKafka(resDatabaseAttrKafkaSettings)).(string),
				settingsSchema.JSON200.Settings.Kafka,
			)
			if err != nil {
				return diag.Errorf("invalid settings: %v", err)
			}
			databaseService.KafkaSettings = &settings
			updated = true
		}
	}

	if updated {
		res, err := client.UpdateDbaasServiceKafkaWithResponse(ctx,
			oapi.DbaasServiceName(d.Get(resDatabaseAttrName).(string)),
			databaseService)
		if err != nil {
			return diag.FromErr(err)
		}
		if res.StatusCode() != http.StatusOK {
			return diag.Errorf("API request error: unexpected status %s", res.Status())
		}
	}

	return nil
}

func resourceDatabaseApplyKafka(ctx context.Context, d *schema.ResourceData, client *egoscale.Client) error { //nolint:gocyclo
	res, err := client.GetDbaasServiceKafkaWithResponse(ctx, oapi.DbaasServiceName(d.Id()))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusOK {
		return fmt.Errorf("API request error: unexpected status %s", res.Status())
	}
	databaseService := res.JSON200

	if err := d.Set(resDatabaseAttrCreatedAt, databaseService.CreatedAt.String()); err != nil {
		return err
	}

	if err := d.Set(resDatabaseAttrDiskSize, *databaseService.DiskSize); err != nil {
		return err
	}

	if databaseService.Maintenance != nil {
		if err := d.Set(resDatabaseAttrMaintenanceDOW, databaseService.Maintenance.Dow); err != nil {
			return err
		}
		if err := d.Set(resDatabaseAttrMaintenanceTime, databaseService.Maintenance.Time); err != nil {
			return err
		}
	}

	if err := d.Set(resDatabaseAttrName, databaseService.Name); err != nil {
		return err
	}

	if err := d.Set(resDatabaseAttrNodeCPUs, *databaseService.NodeCpuCount); err != nil {
		return err
	}

	if err := d.Set(resDatabaseAttrNodeMemory, *databaseService.NodeMemory); err != nil {
		return err
	}

	if err := d.Set(resDatabaseAttrNodes, *databaseService.NodeCount); err != nil {
		return err
	}

	if err := d.Set(resDatabaseAttrPlan, databaseService.Plan); err != nil {
		return err
	}

	if err := d.Set(resDatabaseAttrState, *databaseService.State); err != nil {
		return err
	}

	if err := d.Set(
		resDatabaseAttrTerminationProtection,
		defaultBool(databaseService.TerminationProtection, false),
	); err != nil {
		return err
	}

	if err := d.Set(resDatabaseAttrType, databaseService.Type); err != nil {
		return err
	}

	if err := d.Set(resDatabaseAttrUpdatedAt, databaseService.UpdatedAt.String()); err != nil {
		return err
	}

	if err := d.Set(resDatabaseAttrURI, defaultString(databaseService.Uri, "")); err != nil {
		return err
	}

	kafka := make(map[string]interface{})

	if v := databaseService.KafkaConnectSettings; v != nil {
		settings, err := json.Marshal(*databaseService.KafkaConnectSettings)
		if err != nil {
			return err
		}
		kafka[resDatabaseAttrKafkaConnectSettings] = string(settings)
	}

	if v := databaseService.AuthenticationMethods; v != nil {
		kafka[resDatabaseAttrKafkaEnableCertAuth] = defaultBool(v.Certificate, false)
		kafka[resDatabaseAttrKafkaEnableSASLAuth] = defaultBool(v.Sasl, false)
	}

	kafka[resDatabaseAttrKafkaEnableKafkaConnect] = defaultBool(databaseService.KafkaConnectEnabled, false)
	kafka[resDatabaseAttrKafkaEnableKafkaREST] = defaultBool(databaseService.KafkaRestEnabled, false)
	kafka[resDatabaseAttrKafkaEnableSchemaRegistry] = defaultBool(databaseService.SchemaRegistryEnabled, false)

	if v := databaseService.IpFilter; v != nil {
		kafka[resDatabaseAttrKafkaIPFilter] = *v
	}

	if v := databaseService.KafkaRestSettings; v != nil {
		settings, err := json.Marshal(*databaseService.KafkaRestSettings)
		if err != nil {
			return err
		}
		kafka[resDatabaseAttrKafkaRESTSettings] = string(settings)
	}

	if v := databaseService.SchemaRegistrySettings; v != nil {
		settings, err := json.Marshal(*databaseService.SchemaRegistrySettings)
		if err != nil {
			return err
		}
		kafka[resDatabaseAttrKafkaSchemaRegistrySettings] = string(settings)
	}

	if v := databaseService.KafkaSettings; v != nil {
		settings, err := json.Marshal(*databaseService.KafkaSettings)
		if err != nil {
			return err
		}
		kafka[resDatabaseAttrKafkaSettings] = string(settings)
	}

	if v := databaseService.Version; v != nil {
		// for kafka, the user specifies major.minor e.g. 3.0
		// and the api returns major.minor.patch e.g. 3.0.0
		version := strings.SplitN(*v, ".", 3)
		kafka[resDatabaseAttrKafkaVersion] = version[0] + "." + version[1]
	}

	if len(kafka) > 0 {
		if err := d.Set("kafka", []interface{}{kafka}); err != nil {
			return err
		}
	}

	return nil
}

// resDatabaseAttrKafka returns a database resource attribute key formatted for a "kafka {}" block.
func resDatabaseAttrKafka(a string) string { return fmt.Sprintf("kafka.0.%s", a) }
