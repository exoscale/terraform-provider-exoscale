package exoscale

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"

	"github.com/exoscale/egoscale/v2/oapi"
)

var (
	testAccResourceDatabaseMysqlBackupSchedule        = "01:23"
	testAccResourceDatabaseMysqlBackupScheduleUpdated = "23:45"
	testAccResourceDatabaseMysqlIPFilter              = []string{"1.2.3.4/32"}
	testAccResourceDatabaseMysqlVersion               = "8"
	testAccResourceDatabasePlanMysql                  = "hobbyist-2"

	testAccResourceDatabaseConfigCreateMysql = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_database" "test" {
  zone                   = local.zone
  type                   = "mysql"
  name                   = "%s"
  plan                   = "%s"
  maintenance_dow        = "%s"
  maintenance_time       = "%s"
  termination_protection = true

  mysql {
    version         = "%s"
    ip_filter       = ["%s"]
    backup_schedule = "%s"

    mysql_settings = jsonencode({
      sql_mode                = "ANSI,TRADITIONAL"
      sql_require_primary_key = true
    })
  }
}
`,
		testZoneName,
		testAccResourceDatabaseName,
		testAccResourceDatabasePlanMysql,
		testAccResourceDatabaseMaintenanceDOW,
		testAccResourceDatabaseMaintenanceTime,
		testAccResourceDatabaseMysqlVersion,
		testAccResourceDatabaseMysqlIPFilter[0],
		testAccResourceDatabaseMysqlBackupSchedule,
	)

	testAccResourceDatabaseConfigUpdateMysql = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_database" "test" {
  zone                   = local.zone
  type                   = "mysql"
  name                   = "%s"
  plan                   = "%s"
  maintenance_dow        = "%s"
  maintenance_time       = "%s"
  termination_protection = false

  mysql {
    version         = "%s"
    ip_filter       = []
    backup_schedule = "%s"

    mysql_settings = jsonencode({
      long_query_time         = 5
      sql_mode                = "ANSI,TRADITIONAL"
      sql_require_primary_key = true
    })
  }
}
`,
		testZoneName,
		testAccResourceDatabaseName,
		testAccResourceDatabasePlanMysql,
		testAccResourceDatabaseMaintenanceDOWUpdated,
		testAccResourceDatabaseMaintenanceTimeUpdated,
		testAccResourceDatabaseMysqlVersion,
		testAccResourceDatabaseMysqlBackupScheduleUpdated,
	)
)

func TestAccResourceDatabase_Mysql(t *testing.T) {
	var (
		r               = "exoscale_database.test"
		databaseService interface{}
	)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceDatabaseDestroy("mysql", testAccResourceDatabaseName),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceDatabaseConfigCreateMysql,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceDatabaseExists(r, &databaseService),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(
							testAccResourceDatabaseMysqlBackupSchedule,
							fmt.Sprintf(
								"%02d:%02d",
								*databaseService.(*oapi.DbaasServiceMysql).BackupSchedule.BackupHour,
								*databaseService.(*oapi.DbaasServiceMysql).BackupSchedule.BackupMinute,
							),
						)
						a.Equal(
							testAccResourceDatabaseMysqlIPFilter,
							*databaseService.(*oapi.DbaasServiceMysql).IpFilter,
						)
						a.Equal(oapi.DbaasServiceMaintenance{
							Dow:     oapi.DbaasServiceMaintenanceDow(testAccResourceDatabaseMaintenanceDOW),
							Time:    testAccResourceDatabaseMaintenanceTime,
							Updates: []oapi.DbaasServiceUpdate{},
						}, *databaseService.(*oapi.DbaasServiceMysql).Maintenance)
						a.Equal(
							map[string]interface{}{
								"sql_mode":                "ANSI,TRADITIONAL",
								"sql_require_primary_key": true,
							},
							*databaseService.(*oapi.DbaasServiceMysql).MysqlSettings,
						)
						a.Equal(testAccResourceDatabasePlanMysql, databaseService.(*oapi.DbaasServiceMysql).Plan)
						a.True(*databaseService.(*oapi.DbaasServiceMysql).TerminationProtection)
						a.Equal(testAccResourceDatabaseMysqlVersion, *databaseService.(*oapi.DbaasServiceMysql).Version)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resDatabaseAttrCreatedAt:                                 validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrDiskSize:                                  validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrMaintenanceDOW:                            validateString(testAccResourceDatabaseMaintenanceDOW),
						resDatabaseAttrMaintenanceTime:                           validateString(testAccResourceDatabaseMaintenanceTime),
						resDatabaseAttrNodeCPUs:                                  validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrNodeMemory:                                validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrNodes:                                     validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrState:                                     validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrCA:                                        validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrUpdatedAt:                                 validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrMysql(resDatabaseAttrMysqlBackupSchedule): validateString(testAccResourceDatabaseMysqlBackupSchedule),
						resDatabaseAttrMysql(resDatabaseAttrMysqlIPFilter) + ".0": validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile(testAccResourceDatabaseMysqlIPFilter[0]), ""),
						),
						resDatabaseAttrMysql(resDatabaseAttrMysqlSettings): validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile("sql_mode"), ""),
						),
						resDatabaseAttrMysql(resDatabaseAttrMysqlVersion): validateString(testAccResourceDatabaseMysqlVersion),
					})),
				),
			},
			{
				// Update
				Config: testAccResourceDatabaseConfigUpdateMysql,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceDatabaseExists(r, &databaseService),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(
							testAccResourceDatabaseMysqlBackupScheduleUpdated,
							fmt.Sprintf(
								"%02d:%02d",
								*databaseService.(*oapi.DbaasServiceMysql).BackupSchedule.BackupHour,
								*databaseService.(*oapi.DbaasServiceMysql).BackupSchedule.BackupMinute,
							),
						)
						a.Empty(*databaseService.(*oapi.DbaasServiceMysql).IpFilter)
						a.Equal(oapi.DbaasServiceMaintenance{
							Dow:     oapi.DbaasServiceMaintenanceDow(testAccResourceDatabaseMaintenanceDOWUpdated),
							Time:    testAccResourceDatabaseMaintenanceTimeUpdated,
							Updates: []oapi.DbaasServiceUpdate{},
						}, *databaseService.(*oapi.DbaasServiceMysql).Maintenance)
						a.Equal(
							map[string]interface{}{
								"long_query_time":         float64(5),
								"sql_mode":                "ANSI,TRADITIONAL",
								"sql_require_primary_key": true,
							},
							*databaseService.(*oapi.DbaasServiceMysql).MysqlSettings,
						)
						a.False(*databaseService.(*oapi.DbaasServiceMysql).TerminationProtection)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resDatabaseAttrMaintenanceDOW:                             validateString(testAccResourceDatabaseMaintenanceDOWUpdated),
						resDatabaseAttrMaintenanceTime:                            validateString(testAccResourceDatabaseMaintenanceTimeUpdated),
						resDatabaseAttrMysql(resDatabaseAttrMysqlBackupSchedule):  validateString(testAccResourceDatabaseMysqlBackupScheduleUpdated),
						resDatabaseAttrMysql(resDatabaseAttrMysqlIPFilter) + ".#": validateString("0"),
						resDatabaseAttrMysql(resDatabaseAttrMysqlSettings): validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile("long_query_time"), ""),
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
							resDatabaseAttrCreatedAt:       validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrDiskSize:        validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrMaintenanceDOW:  validateString(testAccResourceDatabaseMaintenanceDOWUpdated),
							resDatabaseAttrMaintenanceTime: validateString(testAccResourceDatabaseMaintenanceTimeUpdated),
							resDatabaseAttrName:            validateString(testAccResourceDatabaseName),
							resDatabaseAttrNodeCPUs:        validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrNodeMemory:      validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrNodes:           validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrPlan:            validateString(testAccResourceDatabasePlanMysql),
							resDatabaseAttrState:           validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrCA:              validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrType:            validateString("mysql"),
							resDatabaseAttrUpdatedAt:       validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrMysql(resDatabaseAttrMysqlBackupSchedule):  validateString(testAccResourceDatabaseMysqlBackupScheduleUpdated),
							resDatabaseAttrMysql(resDatabaseAttrMysqlIPFilter) + ".#": validateString("0"),
							resDatabaseAttrMysql(resDatabaseAttrMysqlSettings):        validation.ToDiagFunc(validation.NoZeroValues),
						},
						s[0].Attributes)
				},
			},
		},
	})
}
