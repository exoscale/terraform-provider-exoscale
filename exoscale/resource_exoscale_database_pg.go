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
	resDatabaseAttrPgAdminPassword   = "admin_password"
	resDatabaseAttrPgAdminUsername   = "admin_username"
	resDatabaseAttrPgBackupSchedule  = "backup_schedule"
	resDatabaseAttrPgIPFilter        = "ip_filter"
	resDatabaseAttrPgSettings        = "pg_settings"
	resDatabaseAttrPgVersion         = "version"
	resDatabaseAttrPgActualVersion   = "actual_version"
	resDatabaseAttrPgbouncerSettings = "pgbouncer_settings"
	resDatabaseAttrPglookoutSettings = "pglookout_settings"
)

var resDatabasePgSchema = &schema.Schema{
	Type:     schema.TypeList,
	MaxItems: 1,
	Optional: true,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			resDatabaseAttrPgAdminPassword: {
				Type:      schema.TypeString,
				Optional:  true,
				Computed:  true,
				Sensitive: true,
			},
			resDatabaseAttrPgAdminUsername: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			resDatabaseAttrPgBackupSchedule: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			resDatabaseAttrPgIPFilter: {
				Type: schema.TypeSet,
				Set:  schema.HashString,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsCIDRNetwork(0, 128),
				},
				Optional: true,
				Computed: true,
			},
			resDatabaseAttrPgSettings: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			resDatabaseAttrPgVersion: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			resDatabaseAttrPgActualVersion: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			resDatabaseAttrPgbouncerSettings: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			resDatabaseAttrPglookoutSettings: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	},
}

func resourceDatabaseCreatePg(
	ctx context.Context,
	d *schema.ResourceData,
	client *egoscale.Client,
) diag.Diagnostics {
	databaseService := oapi.CreateDbaasServicePgJSONRequestBody{
		Plan: d.Get(resDatabaseAttrPlan).(string),
	}

	settingsSchema, err := client.GetDbaasSettingsPgWithResponse(ctx)
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
			Dow  oapi.CreateDbaasServicePgJSONBodyMaintenanceDow `json:"dow"`
			Time string                                          `json:"time"`
		}{
			Dow:  oapi.CreateDbaasServicePgJSONBodyMaintenanceDow(maintenanceDOW),
			Time: maintenanceTime,
		}
	}

	if v, ok := d.GetOk(resDatabaseAttrTerminationProtection); ok {
		b := v.(bool)
		databaseService.TerminationProtection = &b
	}

	if v, ok := d.GetOk(resDatabaseAttrPg(resDatabaseAttrPgAdminPassword)); ok {
		password := v.(string)
		databaseService.AdminPassword = &password
	}

	if v, ok := d.GetOk(resDatabaseAttrPg(resDatabaseAttrPgAdminUsername)); ok {
		username := v.(string)
		databaseService.AdminUsername = &username
	}

	if v, ok := d.GetOk(resDatabaseAttrPg(resDatabaseAttrPgBackupSchedule)); ok {
		bh, bm, err := parseDatabaseServiceBackupSchedule(v.(string))
		if err != nil {
			return diag.FromErr(err)
		}

		databaseService.BackupSchedule = &struct {
			BackupHour   *int64 `json:"backup-hour,omitempty"`
			BackupMinute *int64 `json:"backup-minute,omitempty"`
		}{
			BackupHour:   &bh,
			BackupMinute: &bm,
		}
	}

	if s, ok := d.GetOk(resDatabaseAttrPg(resDatabaseAttrPgIPFilter)); ok {
		databaseService.IpFilter = func() (v *[]string) {
			if l := s.(*schema.Set).Len(); l > 0 {
				list := make([]string, l)
				for i, v := range s.(*schema.Set).List() {
					list[i] = v.(string)
				}
				v = &list
			}
			return
		}()
	}

	if v, ok := d.GetOk(resDatabaseAttrPg(resDatabaseAttrPgSettings)); ok {
		settings, err := validateDatabaseServiceSettings(v.(string), settingsSchema.JSON200.Settings.Pg)
		if err != nil {
			return diag.Errorf("invalid settings: %v", err)
		}
		databaseService.PgSettings = &settings
	}

	if v, ok := d.GetOk(resDatabaseAttrPg(resDatabaseAttrPgVersion)); ok {
		version := v.(string)
		databaseService.Version = &version
	}

	if v, ok := d.GetOk(resDatabaseAttrPg(resDatabaseAttrPgbouncerSettings)); ok {
		settings, err := validateDatabaseServiceSettings(v.(string), settingsSchema.JSON200.Settings.Pgbouncer)
		if err != nil {
			return diag.Errorf("invalid settings: %v", err)
		}
		databaseService.PgbouncerSettings = &settings
	}

	if v, ok := d.GetOk(resDatabaseAttrPg(resDatabaseAttrPglookoutSettings)); ok {
		settings, err := validateDatabaseServiceSettings(v.(string), settingsSchema.JSON200.Settings.Pglookout)
		if err != nil {
			return diag.Errorf("invalid settings: %v", err)
		}
		databaseService.PglookoutSettings = &settings
	}

	databaseServiceName := d.Get(resDatabaseAttrName).(string)

	res, err := client.CreateDbaasServicePgWithResponse(
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

func resourceDatabaseUpdatePg(
	ctx context.Context,
	d *schema.ResourceData,
	client *egoscale.Client,
) diag.Diagnostics {
	var updated bool

	databaseService := oapi.UpdateDbaasServicePgJSONRequestBody{}

	settingsSchema, err := client.GetDbaasSettingsPgWithResponse(ctx)
	if err != nil {
		return diag.Errorf("unable to retrieve Database Service settings: %v", err)
	}
	if settingsSchema.StatusCode() != http.StatusOK {
		return diag.Errorf("API request error: unexpected status %s", settingsSchema.Status())
	}

	if d.HasChange(resDatabaseAttrMaintenanceDOW) || d.HasChange(resDatabaseAttrMaintenanceTime) {
		databaseService.Maintenance = &struct {
			Dow  oapi.UpdateDbaasServicePgJSONBodyMaintenanceDow `json:"dow"`
			Time string                                          `json:"time"`
		}{
			Dow:  oapi.UpdateDbaasServicePgJSONBodyMaintenanceDow(d.Get(resDatabaseAttrMaintenanceDOW).(string)),
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

	if d.HasChange("pg") {
		if d.HasChange(resDatabaseAttrPg(resDatabaseAttrPgBackupSchedule)) {
			bh, bm, err := parseDatabaseServiceBackupSchedule(
				d.Get(resDatabaseAttrPg(resDatabaseAttrPgBackupSchedule)).(string),
			)
			if err != nil {
				return diag.FromErr(err)
			}

			databaseService.BackupSchedule = &struct {
				BackupHour   *int64 `json:"backup-hour,omitempty"`
				BackupMinute *int64 `json:"backup-minute,omitempty"`
			}{
				BackupHour:   &bh,
				BackupMinute: &bm,
			}
		}

		if d.HasChange(resDatabaseAttrPg(resDatabaseAttrPgIPFilter)) {
			databaseService.IpFilter = func() *[]string {
				list := make([]string, 0)
				for _, v := range d.Get(resDatabaseAttrPg(resDatabaseAttrPgIPFilter)).(*schema.Set).List() {
					list = append(list, v.(string))
				}
				return &list
			}()
			updated = true
		}

		if d.HasChange(resDatabaseAttrPg(resDatabaseAttrPgSettings)) {
			settings, err := validateDatabaseServiceSettings(
				d.Get(resDatabaseAttrPg(resDatabaseAttrPgSettings)).(string),
				settingsSchema.JSON200.Settings.Pg,
			)
			if err != nil {
				return diag.Errorf("invalid settings: %v", err)
			}
			databaseService.PgSettings = &settings
			updated = true
		}

		if d.HasChange(resDatabaseAttrPg(resDatabaseAttrPgbouncerSettings)) {
			settings, err := validateDatabaseServiceSettings(
				d.Get(resDatabaseAttrPg(resDatabaseAttrPgbouncerSettings)).(string),
				settingsSchema.JSON200.Settings.Pgbouncer,
			)
			if err != nil {
				return diag.Errorf("invalid settings: %v", err)
			}
			databaseService.PgbouncerSettings = &settings
			updated = true
		}

		if d.HasChange(resDatabaseAttrPg(resDatabaseAttrPglookoutSettings)) {
			settings, err := validateDatabaseServiceSettings(
				d.Get(resDatabaseAttrPg(resDatabaseAttrPglookoutSettings)).(string),
				settingsSchema.JSON200.Settings.Pglookout,
			)
			if err != nil {
				return diag.Errorf("invalid settings: %v", err)
			}
			databaseService.PglookoutSettings = &settings
			updated = true
		}
	}

	if updated {
		res, err := client.UpdateDbaasServicePgWithResponse(ctx,
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

func resourceDatabaseApplyPg(ctx context.Context, d *schema.ResourceData, client *egoscale.Client) error {
	res, err := client.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(d.Id()))
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

	pg := make(map[string]interface{})

	if databaseService.BackupSchedule != nil {
		pg[resDatabaseAttrPgBackupSchedule] = fmt.Sprintf(
			"%02d:%02d",
			defaultInt64(databaseService.BackupSchedule.BackupHour, 0),
			defaultInt64(databaseService.BackupSchedule.BackupMinute, 0),
		)
	}

	if v := databaseService.IpFilter; v != nil {
		pg[resDatabaseAttrPgIPFilter] = *v
	}

	if v := databaseService.PgSettings; v != nil {
		settings, err := json.Marshal(*databaseService.PgSettings)
		if err != nil {
			return err
		}
		pg[resDatabaseAttrPgSettings] = string(settings)
	}

	if v := databaseService.Version; v != nil {
		pg[resDatabaseAttrPgActualVersion] = *v
		pg[resDatabaseAttrPgVersion] = strings.SplitN(*v, ".", 2)[0]
	}

	if v := databaseService.PgbouncerSettings; v != nil {
		settings, err := json.Marshal(*databaseService.PgbouncerSettings)
		if err != nil {
			return err
		}
		pg[resDatabaseAttrPgbouncerSettings] = string(settings)
	}

	if v := databaseService.PglookoutSettings; v != nil {
		settings, err := json.Marshal(*databaseService.PglookoutSettings)
		if err != nil {
			return err
		}
		pg[resDatabaseAttrPglookoutSettings] = string(settings)
	}

	if len(pg) > 0 {
		if err := d.Set("pg", []interface{}{pg}); err != nil {
			return err
		}
	}

	return nil
}

// resDatabaseAttrPg returns a database resource attribute key formatted for a "pg {}" block.
func resDatabaseAttrPg(a string) string { return fmt.Sprintf("pg.0.%s", a) }
