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
	resDatabaseAttrMysqlAdminPassword  = "admin_password"
	resDatabaseAttrMysqlAdminUsername  = "admin_username"
	resDatabaseAttrMysqlBackupSchedule = "backup_schedule"
	resDatabaseAttrMysqlIPFilter       = "ip_filter"
	resDatabaseAttrMysqlSettings       = "mysql_settings"
	resDatabaseAttrMysqlVersion        = "version"
)

var resDatabaseMysqlSchema = &schema.Schema{
	Type:     schema.TypeList,
	MaxItems: 1,
	Optional: true,
	Elem: &schema.Resource{
		Schema: map[string]*schema.Schema{
			resDatabaseAttrMysqlAdminPassword: {
				Type:      schema.TypeString,
				Optional:  true,
				Computed:  true,
				Sensitive: true,
			},
			resDatabaseAttrMysqlAdminUsername: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			resDatabaseAttrMysqlBackupSchedule: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			resDatabaseAttrMysqlIPFilter: {
				Type: schema.TypeSet,
				Set:  schema.HashString,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsCIDRNetwork(0, 128),
				},
				Optional: true,
				Computed: true,
			},
			resDatabaseAttrMysqlSettings: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			resDatabaseAttrMysqlVersion: {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	},
}

func resourceDatabaseCreateMysql(
	ctx context.Context,
	d *schema.ResourceData,
	client *egoscale.Client,
) diag.Diagnostics {
	databaseService := oapi.CreateDbaasServiceMysqlJSONRequestBody{
		Plan: d.Get(resDatabaseAttrPlan).(string),
	}

	settingsSchema, err := client.GetDbaasSettingsMysqlWithResponse(ctx)
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
			Dow  oapi.CreateDbaasServiceMysqlJSONBodyMaintenanceDow `json:"dow"`
			Time string                                             `json:"time"`
		}{
			Dow:  oapi.CreateDbaasServiceMysqlJSONBodyMaintenanceDow(maintenanceDOW),
			Time: maintenanceTime,
		}
	}

	if v, ok := d.GetOk(resDatabaseAttrTerminationProtection); ok {
		b := v.(bool)
		databaseService.TerminationProtection = &b
	}

	if v, ok := d.GetOk(resDatabaseAttrMysql(resDatabaseAttrMysqlAdminPassword)); ok {
		password := v.(string)
		databaseService.AdminPassword = &password
	}

	if v, ok := d.GetOk(resDatabaseAttrMysql(resDatabaseAttrMysqlAdminUsername)); ok {
		username := v.(string)
		databaseService.AdminUsername = &username
	}

	if v, ok := d.GetOk(resDatabaseAttrMysql(resDatabaseAttrMysqlBackupSchedule)); ok {
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

	dg := newResourceDataGetter(d)
	dgos := dg.Under("mysql").Under("0")

	databaseService.IpFilter = dgos.GetSet(resDatabaseAttrMysqlIPFilter)

	if v, ok := d.GetOk(resDatabaseAttrMysql(resDatabaseAttrMysqlSettings)); ok {
		settings, err := validateDatabaseServiceSettings(v.(string), settingsSchema.JSON200.Settings.Mysql)
		if err != nil {
			return diag.Errorf("invalid settings: %v", err)
		}
		databaseService.MysqlSettings = &settings
	}

	if v, ok := d.GetOk(resDatabaseAttrMysql(resDatabaseAttrMysqlVersion)); ok {
		version := v.(string)
		databaseService.Version = &version
	}

	databaseServiceName := d.Get(resDatabaseAttrName).(string)

	res, err := client.CreateDbaasServiceMysqlWithResponse(
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

func resourceDatabaseUpdateMysql(
	ctx context.Context,
	d *schema.ResourceData,
	client *egoscale.Client,
) diag.Diagnostics {
	var updated bool

	databaseService := oapi.UpdateDbaasServiceMysqlJSONRequestBody{}

	settingsSchema, err := client.GetDbaasSettingsMysqlWithResponse(ctx)
	if err != nil {
		return diag.Errorf("unable to retrieve Database Service settings: %v", err)
	}
	if settingsSchema.StatusCode() != http.StatusOK {
		return diag.Errorf("API request error: unexpected status %s", settingsSchema.Status())
	}

	if d.HasChange(resDatabaseAttrMaintenanceDOW) || d.HasChange(resDatabaseAttrMaintenanceTime) {
		databaseService.Maintenance = &struct {
			Dow  oapi.UpdateDbaasServiceMysqlJSONBodyMaintenanceDow `json:"dow"`
			Time string                                             `json:"time"`
		}{
			Dow:  oapi.UpdateDbaasServiceMysqlJSONBodyMaintenanceDow(d.Get(resDatabaseAttrMaintenanceDOW).(string)),
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

	if d.HasChange("mysql") {
		if d.HasChange(resDatabaseAttrMysql(resDatabaseAttrMysqlBackupSchedule)) {
			bh, bm, err := parseDatabaseServiceBackupSchedule(
				d.Get(resDatabaseAttrMysql(resDatabaseAttrMysqlBackupSchedule)).(string),
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
			updated = true
		}

		if d.HasChange(resDatabaseAttrMysql(resDatabaseAttrMysqlIPFilter)) {
			dg := newResourceDataGetter(d)
			dgos := dg.Under("mysql").Under("0")

			databaseService.IpFilter = dgos.GetSet(resDatabaseAttrMysqlIPFilter)
			updated = true
		}

		if d.HasChange(resDatabaseAttrMysql(resDatabaseAttrMysqlSettings)) {
			settings, err := validateDatabaseServiceSettings(
				d.Get(resDatabaseAttrMysql(resDatabaseAttrMysqlSettings)).(string),
				settingsSchema.JSON200.Settings.Mysql,
			)
			if err != nil {
				return diag.Errorf("invalid settings: %v", err)
			}
			databaseService.MysqlSettings = &settings
			updated = true
		}
	}

	if updated {
		// Aiven would overwrite the backup schedule with random value if we don't specify it explicitly.
		if databaseService.BackupSchedule == nil {
			bh, bm, err := parseDatabaseServiceBackupSchedule(
				d.Get(resDatabaseAttrMysql(resDatabaseAttrMysqlBackupSchedule)).(string),
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

		res, err := client.UpdateDbaasServiceMysqlWithResponse(ctx,
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

func resourceDatabaseApplyMysql(ctx context.Context, d *schema.ResourceData, client *egoscale.Client) error {
	res, err := client.GetDbaasServiceMysqlWithResponse(ctx, oapi.DbaasServiceName(d.Id()))
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

	mysql := make(map[string]interface{})

	if databaseService.BackupSchedule != nil {
		mysql[resDatabaseAttrMysqlBackupSchedule] = fmt.Sprintf(
			"%02d:%02d",
			defaultInt64(databaseService.BackupSchedule.BackupHour, 0),
			defaultInt64(databaseService.BackupSchedule.BackupMinute, 0),
		)
	}

	if v := databaseService.IpFilter; v != nil {
		mysql[resDatabaseAttrMysqlIPFilter] = *v
	}

	if v := databaseService.MysqlSettings; v != nil {
		settings, err := json.Marshal(*databaseService.MysqlSettings)
		if err != nil {
			return err
		}
		mysql[resDatabaseAttrMysqlSettings] = string(settings)
	}

	if v := databaseService.Version; v != nil {
		mysql[resDatabaseAttrMysqlVersion] = strings.SplitN(*v, ".", 2)[0]
	}

	if len(mysql) > 0 {
		if err := d.Set("mysql", []interface{}{mysql}); err != nil {
			return err
		}
	}

	return nil
}

// resDatabaseAttrMysql returns a database resource attribute key formatted for a "mysql {}" block.
func resDatabaseAttrMysql(a string) string { return fmt.Sprintf("mysql.0.%s", a) }
