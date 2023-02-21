package exoscale

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	egoscale "github.com/exoscale/egoscale/v2"
	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	resDatabaseAttrRedisIPFilter = "ip_filter"
	resDatabaseAttrRedisSettings = "redis_settings"
)

var resDatabaseRedisSchema = &schema.Schema{
	Description: "*redis* database service type specific arguments. Structure is documented below.",
	Type:        schema.TypeList,
	MaxItems:    1,
	Optional:    true,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			resDatabaseAttrRedisIPFilter: {
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
			resDatabaseAttrRedisSettings: {
				Description: "Redis configuration settings in JSON format (`exo dbaas type show redis --settings=redis` for reference).",
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
			},
		},
	},
}

func resourceDatabaseCreateRedis(
	ctx context.Context,
	d *schema.ResourceData,
	client *egoscale.Client,
) diag.Diagnostics {
	databaseService := oapi.CreateDbaasServiceRedisJSONRequestBody{
		Plan: d.Get(resDatabaseAttrPlan).(string),
	}

	settingsSchema, err := client.GetDbaasSettingsRedisWithResponse(ctx)
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
			Dow  oapi.CreateDbaasServiceRedisJSONBodyMaintenanceDow `json:"dow"`
			Time string                                             `json:"time"`
		}{
			Dow:  oapi.CreateDbaasServiceRedisJSONBodyMaintenanceDow(maintenanceDOW),
			Time: maintenanceTime,
		}
	}

	if v, ok := d.GetOk(resDatabaseAttrTerminationProtection); ok {
		b := v.(bool)
		databaseService.TerminationProtection = &b
	}

	dg := newResourceDataGetter(d)
	dgos := dg.Under("redis").Under("0")

	databaseService.IpFilter = dgos.GetSet(resDatabaseAttrRedisIPFilter)

	if v, ok := d.GetOk(resDatabaseAttrRedis(resDatabaseAttrRedisSettings)); ok {
		settings, err := validateDatabaseServiceSettings(v.(string), settingsSchema.JSON200.Settings.Redis)
		if err != nil {
			return diag.Errorf("invalid settings: %v", err)
		}
		databaseService.RedisSettings = &settings
	}

	databaseServiceName := d.Get(resDatabaseAttrName).(string)

	res, err := client.CreateDbaasServiceRedisWithResponse(
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

func resourceDatabaseUpdateRedis(
	ctx context.Context,
	d *schema.ResourceData,
	client *egoscale.Client,
) diag.Diagnostics {
	var updated bool
	databaseService := oapi.UpdateDbaasServiceRedisJSONRequestBody{}

	settingsSchema, err := client.GetDbaasSettingsRedisWithResponse(ctx)
	if err != nil {
		return diag.Errorf("unable to retrieve Database Service settings: %v", err)
	}
	if settingsSchema.StatusCode() != http.StatusOK {
		return diag.Errorf("API request error: unexpected status %s", settingsSchema.Status())
	}

	if d.HasChange(resDatabaseAttrMaintenanceDOW) || d.HasChange(resDatabaseAttrMaintenanceTime) {
		databaseService.Maintenance = &struct {
			Dow  oapi.UpdateDbaasServiceRedisJSONBodyMaintenanceDow `json:"dow"`
			Time string                                             `json:"time"`
		}{
			Dow:  oapi.UpdateDbaasServiceRedisJSONBodyMaintenanceDow(d.Get(resDatabaseAttrMaintenanceDOW).(string)),
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

	if d.HasChange("redis") {
		if d.HasChange(resDatabaseAttrRedis(resDatabaseAttrRedisIPFilter)) {
			dg := newResourceDataGetter(d)
			dgos := dg.Under("redis").Under("0")

			databaseService.IpFilter = dgos.GetSet(resDatabaseAttrRedisIPFilter)
			updated = true
		}

		if d.HasChange(resDatabaseAttrRedis(resDatabaseAttrRedisSettings)) {
			settings, err := validateDatabaseServiceSettings(
				d.Get(resDatabaseAttrRedis(resDatabaseAttrRedisSettings)).(string),
				settingsSchema.JSON200.Settings.Redis,
			)
			if err != nil {
				return diag.Errorf("invalid settings: %v", err)
			}
			databaseService.RedisSettings = &settings
			updated = true
		}
	}

	if updated {
		res, err := client.UpdateDbaasServiceRedisWithResponse(ctx,
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

func resourceDatabaseApplyRedis(ctx context.Context, d *schema.ResourceData, client *egoscale.Client) error {
	res, err := client.GetDbaasServiceRedisWithResponse(ctx, oapi.DbaasServiceName(d.Id()))
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

	redis := make(map[string]interface{})

	if v := databaseService.IpFilter; v != nil {
		redis[resDatabaseAttrRedisIPFilter] = *v
	}

	if v := databaseService.RedisSettings; v != nil {
		settings, err := json.Marshal(*databaseService.RedisSettings)
		if err != nil {
			return err
		}
		redis[resDatabaseAttrRedisSettings] = string(settings)
	}

	if len(redis) > 0 {
		if err := d.Set("redis", []interface{}{redis}); err != nil {
			return err
		}
	}

	return nil
}

// resDatabaseAttrRedis returns a database resource attribute key formatted for a "redis {}" block.
func resDatabaseAttrRedis(a string) string { return fmt.Sprintf("redis.0.%s", a) }
