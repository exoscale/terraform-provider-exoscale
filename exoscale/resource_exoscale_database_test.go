package exoscale

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
)

// Note: at the moment we can't test updating the `plan` attribute, as the Aiven API
// prevents us from upgrading the plan of a service that hasn't been backed up at least once.
// (which is the case of a service that has just been created by the TF provider).

var (
	testAccResourceDatabaseMaintenanceDOW         = "monday"
	testAccResourceDatabaseMaintenanceDOWUpdated  = "tuesday"
	testAccResourceDatabaseMaintenanceTime        = "01:23:00"
	testAccResourceDatabaseMaintenanceTimeUpdated = "02:34:00"
	testAccResourceDatabaseName                   = acctest.RandomWithPrefix(testPrefix)
	testAccResourceDatabaseType                   = "pg"
	testAccResourceDatabasePlan                   = "hobbyist-1"
	testAccResourceDatabaseUserConfigIPFilter     = []string{"1.2.3.4/32"}

	testAccResourceDatabaseConfigCreate = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_database" "test" {
  zone                   = local.zone
  name                   = "%s"
  type                   = "%s"
  plan                   = "%s"
  maintenance_dow        = "%s"
  maintenance_time       = "%s"
  termination_protection = true
  user_config = jsonencode({
    backup_hour   = 1
    backup_minute = 1
    ip_filter     = ["%s"]
    pg_version    = "13"
    pglookout = {
      max_failover_replication_time_lag = 60
    }
  })

  timeouts {
    create = "10m"
  }
}
`,
		testZoneName,
		testAccResourceDatabaseName,
		testAccResourceDatabaseType,
		testAccResourceDatabasePlan,
		testAccResourceDatabaseMaintenanceDOW,
		testAccResourceDatabaseMaintenanceTime,
		testAccResourceDatabaseUserConfigIPFilter[0],
	)

	testAccResourceDatabaseConfigUpdate = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_database" "test" {
  zone                   = local.zone
  name                   = "%s"
  type                   = "%s"
  plan                   = "%s"
  maintenance_dow        = "%s"
  maintenance_time       = "%s"
  termination_protection = false
  user_config = jsonencode({
    backup_hour   = 1
    backup_minute = 1
    ip_filter     = []
    pg_version    = "13"
    pglookout = {
      max_failover_replication_time_lag = 60
    }
  })

  timeouts {
    create = "10m"
  }
}
`,
		testZoneName,
		testAccResourceDatabaseName,
		testAccResourceDatabaseType,
		testAccResourceDatabasePlan,
		testAccResourceDatabaseMaintenanceDOWUpdated,
		testAccResourceDatabaseMaintenanceTimeUpdated,
	)
)

func TestAccResourceDatabase(t *testing.T) {
	var (
		r        = "exoscale_database.test"
		database egoscale.DatabaseService
	)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceDatabaseDestroy(&database),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceDatabaseConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceDatabaseExists(r, &database),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(egoscale.DatabaseServiceMaintenance{
							DOW:  testAccResourceDatabaseMaintenanceDOW,
							Time: testAccResourceDatabaseMaintenanceTime,
						}, *database.Maintenance)
						a.Equal(testAccResourceDatabaseName, *database.Name)
						a.Equal(testAccResourceDatabasePlan, *database.Plan)
						a.True(*database.TerminationProtection)
						a.Equal(testAccResourceDatabaseType, *database.Type)
						a.Equal(testAccResourceDatabaseUserConfigIPFilter[0], (*database.UserConfig)["ip_filter"].([]interface{})[0])

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resDatabaseAttrCreatedAt:       validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrDiskSize:        validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrFeatures + ".%": validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrMetadata + ".%": validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrName:            validateString(testAccResourceDatabaseName),
						resDatabaseAttrNodeCPUs:        validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrNodeMemory:      validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrNodes:           validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrPlan:            validateString(testAccResourceDatabasePlan),
						resDatabaseAttrState:           validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrType:            validateString(testAccResourceDatabaseType),
						resDatabaseAttrUpdatedAt:       validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrURI:             validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrUserConfig: validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile(
								testAccResourceDatabaseUserConfigIPFilter[0],
							), ""),
						),
					})),
				),
			},
			{
				// Update
				Config: testAccResourceDatabaseConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceDatabaseExists(r, &database),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(egoscale.DatabaseServiceMaintenance{
							DOW:  testAccResourceDatabaseMaintenanceDOWUpdated,
							Time: testAccResourceDatabaseMaintenanceTimeUpdated,
						}, *database.Maintenance)
						a.Equal(testAccResourceDatabaseName, *database.Name)
						a.Equal(testAccResourceDatabasePlan, *database.Plan)
						a.False(*database.TerminationProtection)
						a.Equal(testAccResourceDatabaseType, *database.Type)
						a.Empty((*database.UserConfig)["ip_filter"])

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resDatabaseAttrCreatedAt:       validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrDiskSize:        validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrFeatures + ".%": validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrMetadata + ".%": validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrName:            validateString(testAccResourceDatabaseName),
						resDatabaseAttrNodeCPUs:        validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrNodeMemory:      validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrNodes:           validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrPlan:            validateString(testAccResourceDatabasePlan),
						resDatabaseAttrState:           validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrType:            validateString(testAccResourceDatabaseType),
						resDatabaseAttrUpdatedAt:       validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrURI:             validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrUserConfig: validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile(`"ip_filter":\[\]`), ""),
						),
					})),
				),
			},
			{
				// Import
				ResourceName: r,
				ImportStateIdFunc: func(database *egoscale.DatabaseService) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", *database.Name, testZoneName), nil
					}
				}(&database),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resDatabaseAttrCreatedAt:       validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrDiskSize:        validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrFeatures + ".%": validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrMetadata + ".%": validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrName:            validateString(testAccResourceDatabaseName),
							resDatabaseAttrNodeCPUs:        validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrNodeMemory:      validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrNodes:           validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrPlan:            validateString(testAccResourceDatabasePlan),
							resDatabaseAttrState:           validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrType:            validateString(testAccResourceDatabaseType),
							resDatabaseAttrUpdatedAt:       validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrURI:             validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrUserConfig:      validation.ToDiagFunc(validation.NoZeroValues),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceDatabaseExists(r string, database *egoscale.DatabaseService) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client := GetComputeClient(testAccProvider.Meta())

		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testZoneName),
		)
		res, err := client.GetDatabaseService(ctx, testZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*database = *res
		return nil
	}
}

func testAccCheckResourceDatabaseDestroy(database *egoscale.DatabaseService) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testZoneName),
		)

		_, err := client.GetDatabaseService(ctx, testZoneName, *database.Name)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("database service still exists")
	}
}
