package exoscale

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
)

var (
	testAccResourceDatabaseRedisIPFilter = []string{"1.2.3.4/32"}
	testAccResourceDatabasePlanRedis     = "hobbyist-2"

	testAccResourceDatabaseConfigCreateRedis = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_database" "test" {
  zone                   = local.zone
  type                   = "redis"
  name                   = "%s"
  plan                   = "%s"
  maintenance_dow        = "%s"
  maintenance_time       = "%s"
  termination_protection = true

  redis {
    ip_filter       = ["%s"]

    redis_settings = jsonencode({
      lfu_decay_time         = 1
      lfu_log_factor         = 10
      maxmemory_policy       = "noeviction"
      notify_keyspace_events = ""
      persistence            = "rdb"
      ssl                    = true
      timeout                = 300
    })
  }
}
`,
		testZoneName,
		testAccResourceDatabaseName,
		testAccResourceDatabasePlanRedis,
		testAccResourceDatabaseMaintenanceDOW,
		testAccResourceDatabaseMaintenanceTime,
		testAccResourceDatabaseRedisIPFilter[0],
	)

	testAccResourceDatabaseConfigUpdateRedis = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_database" "test" {
  zone                   = local.zone
  type                   = "redis"
  name                   = "%s"
  plan                   = "%s"
  maintenance_dow        = "%s"
  maintenance_time       = "%s"
  termination_protection = false

  redis {
    ip_filter       = []

    redis_settings = jsonencode({
      lfu_decay_time                    = 1
      lfu_log_factor                    = 10
      maxmemory_policy                  = "noeviction"
      notify_keyspace_events            = ""
      persistence                       = "rdb"
      pubsub_client_output_buffer_limit = 64
      ssl                               = true
      timeout                           = 300
    })
  }
}
`,
		testZoneName,
		testAccResourceDatabaseName,
		testAccResourceDatabasePlanRedis,
		testAccResourceDatabaseMaintenanceDOWUpdated,
		testAccResourceDatabaseMaintenanceTimeUpdated,
	)
)

func TestAccResourceDatabase_Redis(t *testing.T) {
	var (
		r               = "exoscale_database.test"
		databaseService interface{}
	)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceDatabaseDestroy("redis", testAccResourceDatabaseName),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceDatabaseConfigCreateRedis,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceDatabaseExists(r, &databaseService),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(
							testAccResourceDatabaseRedisIPFilter,
							*databaseService.(*oapi.DbaasServiceRedis).IpFilter,
						)
						a.Equal(oapi.DbaasServiceMaintenance{
							Dow:     oapi.DbaasServiceMaintenanceDow(testAccResourceDatabaseMaintenanceDOW),
							Time:    testAccResourceDatabaseMaintenanceTime,
							Updates: []oapi.DbaasServiceUpdate{},
						}, *databaseService.(*oapi.DbaasServiceRedis).Maintenance)
						a.Equal(
							map[string]interface{}{
								"lfu_decay_time":         float64(1),
								"lfu_log_factor":         float64(10),
								"maxmemory_policy":       "noeviction",
								"notify_keyspace_events": "",
								"persistence":            "rdb",
								"ssl":                    true,
								"timeout":                float64(300),
							},
							*databaseService.(*oapi.DbaasServiceRedis).RedisSettings,
						)
						a.Equal(testAccResourceDatabasePlanRedis, databaseService.(*oapi.DbaasServiceRedis).Plan)
						a.True(*databaseService.(*oapi.DbaasServiceRedis).TerminationProtection)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resDatabaseAttrCreatedAt:       validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrDiskSize:        validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrMaintenanceDOW:  validateString(testAccResourceDatabaseMaintenanceDOW),
						resDatabaseAttrMaintenanceTime: validateString(testAccResourceDatabaseMaintenanceTime),
						resDatabaseAttrNodeCPUs:        validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrNodeMemory:      validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrNodes:           validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrState:           validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrCA:              validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrUpdatedAt:       validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrURI:             validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrRedis(resDatabaseAttrRedisIPFilter) + ".0": validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile(testAccResourceDatabaseRedisIPFilter[0]), ""),
						),
						resDatabaseAttrRedis(resDatabaseAttrRedisSettings): validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile("persistence"), ""),
						),
					})),
				),
			},
			{
				// Update
				Config: testAccResourceDatabaseConfigUpdateRedis,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceDatabaseExists(r, &databaseService),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Empty(*databaseService.(*oapi.DbaasServiceRedis).IpFilter)
						a.Equal(oapi.DbaasServiceMaintenance{
							Dow:     oapi.DbaasServiceMaintenanceDow(testAccResourceDatabaseMaintenanceDOWUpdated),
							Time:    testAccResourceDatabaseMaintenanceTimeUpdated,
							Updates: []oapi.DbaasServiceUpdate{},
						}, *databaseService.(*oapi.DbaasServiceRedis).Maintenance)
						a.Equal(
							map[string]interface{}{
								"lfu_decay_time":                    float64(1),
								"lfu_log_factor":                    float64(10),
								"maxmemory_policy":                  "noeviction",
								"notify_keyspace_events":            "",
								"persistence":                       "rdb",
								"pubsub_client_output_buffer_limit": float64(64),
								"ssl":                               true,
								"timeout":                           float64(300),
							},
							*databaseService.(*oapi.DbaasServiceRedis).RedisSettings,
						)
						a.False(*databaseService.(*oapi.DbaasServiceRedis).TerminationProtection)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resDatabaseAttrMaintenanceDOW:                             validateString(testAccResourceDatabaseMaintenanceDOWUpdated),
						resDatabaseAttrMaintenanceTime:                            validateString(testAccResourceDatabaseMaintenanceTimeUpdated),
						resDatabaseAttrRedis(resDatabaseAttrRedisIPFilter) + ".#": validateString("0"),
						resDatabaseAttrRedis(resDatabaseAttrRedisSettings): validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile("pubsub_client_output_buffer_limit"), ""),
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
							resDatabaseAttrPlan:            validateString(testAccResourceDatabasePlanRedis),
							resDatabaseAttrState:           validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrCA:              validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrType:            validateString("redis"),
							resDatabaseAttrURI:             validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrUpdatedAt:       validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrRedis(resDatabaseAttrRedisIPFilter) + ".#": validateString("0"),
							resDatabaseAttrRedis(resDatabaseAttrRedisSettings):        validation.ToDiagFunc(validation.NoZeroValues),
						},
						s[0].Attributes)
				},
			},
		},
	})
}
