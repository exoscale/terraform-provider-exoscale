package exoscale

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	"github.com/exoscale/egoscale/v2/oapi"
)

var (
	testAccResourceDatabasePgBackupSchedule        = "01:23"
	testAccResourceDatabasePgBackupScheduleUpdated = "23:45"
	testAccResourceDatabasePgIPFilter              = []string{"1.2.3.4/32"}
	testAccResourceDatabasePgVersion               = "13"
	testAccResourceDatabasePlanPg                  = "hobbyist-2"

	testAccResourceDatabaseConfigCreatePg = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_database" "test" {
  zone                   = local.zone
  type                   = "pg"
  name                   = "%s"
  plan                   = "%s"
  maintenance_dow        = "%s"
  maintenance_time       = "%s"
  termination_protection = true

  pg {
    version         = "%s"
    ip_filter       = ["%s"]
    backup_schedule = "%s"

    pg_settings = jsonencode({
      timezone = "Europe/Zurich"
    })

    pgbouncer_settings = jsonencode({
      min_pool_size = 10
    })
  }
}
`,
		testZoneName,
		testAccResourceDatabaseName,
		testAccResourceDatabasePlanPg,
		testAccResourceDatabaseMaintenanceDOW,
		testAccResourceDatabaseMaintenanceTime,
		testAccResourceDatabasePgVersion,
		testAccResourceDatabasePgIPFilter[0],
		testAccResourceDatabasePgBackupSchedule,
	)

	testAccResourceDatabaseConfigUpdatePg = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_database" "test" {
  zone                   = local.zone
  type                   = "pg"
  name                   = "%s"
  plan                   = "%s"
  maintenance_dow        = "%s"
  maintenance_time       = "%s"
  termination_protection = false

  pg {
    version         = "%s"
    ip_filter       = []
    backup_schedule = "%s"

    pg_settings = jsonencode({
      timezone               = "Europe/Zurich"
      autovacuum_max_workers = 5
    })

    pgbouncer_settings = jsonencode({
      min_pool_size    = 10
      autodb_pool_size = 5
    })

    pglookout_settings = jsonencode({
      max_failover_replication_time_lag = 30
    })
  }
}
`,
		testZoneName,
		testAccResourceDatabaseName,
		testAccResourceDatabasePlanPg,
		testAccResourceDatabaseMaintenanceDOWUpdated,
		testAccResourceDatabaseMaintenanceTimeUpdated,
		testAccResourceDatabasePgVersion,
		testAccResourceDatabasePgBackupScheduleUpdated,
	)
)

func TestAccResourceDatabase_Pg(t *testing.T) {
	var (
		r               = "exoscale_database.test"
		databaseService interface{}
	)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceDatabaseDestroy("pg", testAccResourceDatabaseName),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceDatabaseConfigCreatePg,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceDatabaseExists(r, &databaseService),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(
							testAccResourceDatabasePgBackupSchedule,
							fmt.Sprintf(
								"%02d:%02d",
								*databaseService.(*oapi.DbaasServicePg).BackupSchedule.BackupHour,
								*databaseService.(*oapi.DbaasServicePg).BackupSchedule.BackupMinute,
							),
						)
						a.Equal(
							testAccResourceDatabasePgIPFilter,
							*databaseService.(*oapi.DbaasServicePg).IpFilter,
						)
						a.Equal(oapi.DbaasServiceMaintenance{
							Dow:     oapi.DbaasServiceMaintenanceDow(testAccResourceDatabaseMaintenanceDOW),
							Time:    testAccResourceDatabaseMaintenanceTime,
							Updates: []oapi.DbaasServiceUpdate{},
						}, *databaseService.(*oapi.DbaasServicePg).Maintenance)
						a.Equal(
							map[string]interface{}{"timezone": "Europe/Zurich"},
							*databaseService.(*oapi.DbaasServicePg).PgSettings,
						)
						a.Equal(
							map[string]interface{}{"min_pool_size": float64(10)},
							*databaseService.(*oapi.DbaasServicePg).PgbouncerSettings,
						)
						a.Equal(testAccResourceDatabasePlanPg, databaseService.(*oapi.DbaasServicePg).Plan)
						a.True(*databaseService.(*oapi.DbaasServicePg).TerminationProtection)
						a.Equal(testAccResourceDatabasePgVersion, *databaseService.(*oapi.DbaasServicePg).Version)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resDatabaseAttrCreatedAt:                           validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrDiskSize:                            validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrMaintenanceDOW:                      validateString(testAccResourceDatabaseMaintenanceDOW),
						resDatabaseAttrMaintenanceTime:                     validateString(testAccResourceDatabaseMaintenanceTime),
						resDatabaseAttrNodeCPUs:                            validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrNodeMemory:                          validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrNodes:                               validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrState:                               validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrCA:                                  validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrUpdatedAt:                           validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrPg(resDatabaseAttrPgBackupSchedule): validateString(testAccResourceDatabasePgBackupSchedule),
						resDatabaseAttrPg(resDatabaseAttrPgIPFilter) + ".0": validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile(testAccResourceDatabasePgIPFilter[0]), ""),
						),
						resDatabaseAttrPg(resDatabaseAttrPgSettings): validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile("timezone"), ""),
						),
						resDatabaseAttrPg(resDatabaseAttrPgbouncerSettings): validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile("min_pool_size"), ""),
						),
						resDatabaseAttrPg(resDatabaseAttrPgVersion): validateString(testAccResourceDatabasePgVersion),
					})),
				),
			},
			{
				// Update
				Config: testAccResourceDatabaseConfigUpdatePg,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceDatabaseExists(r, &databaseService),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(
							testAccResourceDatabasePgBackupScheduleUpdated,
							fmt.Sprintf(
								"%02d:%02d",
								*databaseService.(*oapi.DbaasServicePg).BackupSchedule.BackupHour,
								*databaseService.(*oapi.DbaasServicePg).BackupSchedule.BackupMinute,
							),
						)
						a.Empty(*databaseService.(*oapi.DbaasServicePg).IpFilter)
						a.Equal(oapi.DbaasServiceMaintenance{
							Dow:     oapi.DbaasServiceMaintenanceDow(testAccResourceDatabaseMaintenanceDOWUpdated),
							Time:    testAccResourceDatabaseMaintenanceTimeUpdated,
							Updates: []oapi.DbaasServiceUpdate{},
						}, *databaseService.(*oapi.DbaasServicePg).Maintenance)
						a.Equal(
							map[string]interface{}{
								"timezone":               "Europe/Zurich",
								"autovacuum_max_workers": float64(5),
							},
							*databaseService.(*oapi.DbaasServicePg).PgSettings,
						)
						a.Equal(
							map[string]interface{}{
								"min_pool_size":    float64(10),
								"autodb_pool_size": float64(5),
							},
							*databaseService.(*oapi.DbaasServicePg).PgbouncerSettings,
						)
						a.Equal(
							map[string]interface{}{"max_failover_replication_time_lag": float64(30)},
							*databaseService.(*oapi.DbaasServicePg).PglookoutSettings,
						)
						a.False(*databaseService.(*oapi.DbaasServicePg).TerminationProtection)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resDatabaseAttrMaintenanceDOW:                       validateString(testAccResourceDatabaseMaintenanceDOWUpdated),
						resDatabaseAttrMaintenanceTime:                      validateString(testAccResourceDatabaseMaintenanceTimeUpdated),
						resDatabaseAttrPg(resDatabaseAttrPgBackupSchedule):  validateString(testAccResourceDatabasePgBackupScheduleUpdated),
						resDatabaseAttrPg(resDatabaseAttrPgIPFilter) + ".#": validateString("0"),
						resDatabaseAttrPg(resDatabaseAttrPgSettings): validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile("autovacuum_max_workers"), ""),
						),
						resDatabaseAttrPg(resDatabaseAttrPgbouncerSettings): validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile("autodb_pool_size"), ""),
						),
						resDatabaseAttrPg(resDatabaseAttrPglookoutSettings): validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile("max_failover_replication_time_lag"), ""),
						),
					})),
				),
			},
			{
				// Import
				ResourceName: r,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", testAccResourceDatabaseName, testZoneName), nil
					}
				}(),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resDatabaseAttrCreatedAt:                            validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrDiskSize:                             validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrMaintenanceDOW:                       validateString(testAccResourceDatabaseMaintenanceDOWUpdated),
							resDatabaseAttrMaintenanceTime:                      validateString(testAccResourceDatabaseMaintenanceTimeUpdated),
							resDatabaseAttrName:                                 validateString(testAccResourceDatabaseName),
							resDatabaseAttrNodeCPUs:                             validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrNodeMemory:                           validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrNodes:                                validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrPlan:                                 validateString(testAccResourceDatabasePlanPg),
							resDatabaseAttrState:                                validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrCA:                                   validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrType:                                 validateString("pg"),
							resDatabaseAttrUpdatedAt:                            validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrPg(resDatabaseAttrPgBackupSchedule):  validateString(testAccResourceDatabasePgBackupScheduleUpdated),
							resDatabaseAttrPg(resDatabaseAttrPgIPFilter) + ".#": validateString("0"),
							resDatabaseAttrPg(resDatabaseAttrPgSettings):        validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrPg(resDatabaseAttrPgbouncerSettings): validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrPg(resDatabaseAttrPglookoutSettings): validation.ToDiagFunc(validation.NoZeroValues),
						},
						s[0].Attributes)
				},
			},
		},
	})
}
