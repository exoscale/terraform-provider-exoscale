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
	testAccResourceDatabaseKafkaIPFilter = []string{"1.2.3.4/32"}
	testAccResourceDatabaseKafkaVersion  = "3.3"
	testAccResourceDatabasePlanKafka     = "business-4"

	testAccResourceDatabaseConfigCreateKafka = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_database" "test" {
  zone                   = local.zone
  type                   = "kafka"
  name                   = "%s"
  plan                   = "%s"
  maintenance_dow        = "%s"
  maintenance_time       = "%s"
  termination_protection = true

  kafka {
    version = "%s"
    ip_filter = ["%s"]
    enable_cert_auth = true
    kafka_settings = jsonencode({
      num_partitions = 10
    })
  }
}
`,
		testZoneName,
		testAccResourceDatabaseName,
		testAccResourceDatabasePlanKafka,
		testAccResourceDatabaseMaintenanceDOW,
		testAccResourceDatabaseMaintenanceTime,
		testAccResourceDatabaseKafkaVersion,
		testAccResourceDatabaseKafkaIPFilter[0],
	)

	testAccResourceDatabaseConfigUpdateKafka = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_database" "test" {
  zone                   = local.zone
  type                   = "kafka"
  name                   = "%s"
  plan                   = "%s"
  maintenance_dow        = "%s"
  maintenance_time       = "%s"
  termination_protection = false

  kafka {
    version = "%s"
    ip_filter = []
    enable_cert_auth = false
    enable_sasl_auth = true
    enable_kafka_rest = true
    enable_kafka_connect = true

    kafka_settings = jsonencode({
      num_partitions   = 10
      compression_type = "gzip"
    })

    kafka_rest_settings = jsonencode({
      consumer_request_max_bytes = 100000
    })

    kafka_connect_settings = jsonencode({
      session_timeout_ms = 6000
    })
  }
}
`,
		testZoneName,
		testAccResourceDatabaseName,
		testAccResourceDatabasePlanKafka,
		testAccResourceDatabaseMaintenanceDOWUpdated,
		testAccResourceDatabaseMaintenanceTimeUpdated,
		testAccResourceDatabaseKafkaVersion,
	)
)

func TestAccResourceDatabase_Kafka(t *testing.T) {
	var (
		r               = "exoscale_database.test"
		databaseService interface{}
	)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceDatabaseDestroy("kafka", testAccResourceDatabaseName),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceDatabaseConfigCreateKafka,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceDatabaseExists(r, &databaseService),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.True(*databaseService.(*oapi.DbaasServiceKafka).AuthenticationMethods.Certificate)
						a.False(*databaseService.(*oapi.DbaasServiceKafka).AuthenticationMethods.Sasl)
						a.Equal(
							testAccResourceDatabaseKafkaIPFilter,
							*databaseService.(*oapi.DbaasServiceKafka).IpFilter,
						)
						a.Equal(oapi.DbaasServiceMaintenance{
							Dow:     oapi.DbaasServiceMaintenanceDow(testAccResourceDatabaseMaintenanceDOW),
							Time:    testAccResourceDatabaseMaintenanceTime,
							Updates: []oapi.DbaasServiceUpdate{},
						}, *databaseService.(*oapi.DbaasServiceKafka).Maintenance)
						a.Equal(
							map[string]interface{}{"num_partitions": float64(10)},
							*databaseService.(*oapi.DbaasServiceKafka).KafkaSettings,
						)
						a.Equal(testAccResourceDatabasePlanKafka, databaseService.(*oapi.DbaasServiceKafka).Plan)
						a.True(*databaseService.(*oapi.DbaasServiceKafka).TerminationProtection)
						a.Equal(testAccResourceDatabaseKafkaVersion, *databaseService.(*oapi.DbaasServiceKafka).Version)

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
						resDatabaseAttrKafka(resDatabaseAttrKafkaIPFilter) + ".0": validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile(testAccResourceDatabaseKafkaIPFilter[0]), ""),
						),
						resDatabaseAttrKafka(resDatabaseAttrKafkaSettings): validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile("num_partitions"), ""),
						),
						resDatabaseAttrKafka(resDatabaseAttrKafkaVersion): validateString(testAccResourceDatabaseKafkaVersion),
					})),
				),
			},
			{
				// Update
				Config: testAccResourceDatabaseConfigUpdateKafka,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceDatabaseExists(r, &databaseService),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.False(*databaseService.(*oapi.DbaasServiceKafka).AuthenticationMethods.Certificate)
						a.True(*databaseService.(*oapi.DbaasServiceKafka).AuthenticationMethods.Sasl)
						a.Empty(*databaseService.(*oapi.DbaasServiceKafka).IpFilter)
						a.Equal(oapi.DbaasServiceMaintenance{
							Dow:     oapi.DbaasServiceMaintenanceDow(testAccResourceDatabaseMaintenanceDOWUpdated),
							Time:    testAccResourceDatabaseMaintenanceTimeUpdated,
							Updates: []oapi.DbaasServiceUpdate{},
						}, *databaseService.(*oapi.DbaasServiceKafka).Maintenance)
						a.True(*databaseService.(*oapi.DbaasServiceKafka).KafkaConnectEnabled)
						a.Equal(
							map[string]interface{}{"session_timeout_ms": float64(6000)},
							*databaseService.(*oapi.DbaasServiceKafka).KafkaConnectSettings,
						)
						a.True(*databaseService.(*oapi.DbaasServiceKafka).KafkaRestEnabled)
						a.Equal(
							map[string]interface{}{"consumer_request_max_bytes": float64(100000)},
							*databaseService.(*oapi.DbaasServiceKafka).KafkaRestSettings,
						)
						a.Equal(
							map[string]interface{}{
								"num_partitions":   float64(10),
								"compression_type": "gzip",
							},
							*databaseService.(*oapi.DbaasServiceKafka).KafkaSettings,
						)
						a.False(*databaseService.(*oapi.DbaasServiceKafka).TerminationProtection)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resDatabaseAttrMaintenanceDOW:                             validateString(testAccResourceDatabaseMaintenanceDOWUpdated),
						resDatabaseAttrMaintenanceTime:                            validateString(testAccResourceDatabaseMaintenanceTimeUpdated),
						resDatabaseAttrKafka(resDatabaseAttrKafkaEnableCertAuth):  validateString("false"),
						resDatabaseAttrKafka(resDatabaseAttrKafkaEnableSASLAuth):  validateString("true"),
						resDatabaseAttrKafka(resDatabaseAttrKafkaIPFilter) + ".#": validateString("0"),
						resDatabaseAttrKafka(resDatabaseAttrKafkaConnectSettings): validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile("session_timeout_ms"), ""),
						),
						resDatabaseAttrKafka(resDatabaseAttrKafkaRESTSettings): validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile("consumer_request_max_bytes"), ""),
						),
						resDatabaseAttrKafka(resDatabaseAttrKafkaSettings): validation.ToDiagFunc(
							validation.StringMatch(regexp.MustCompile("compression_type"), ""),
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
							resDatabaseAttrPlan:            validateString(testAccResourceDatabasePlanKafka),
							resDatabaseAttrState:           validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrCA:              validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrType:            validateString("kafka"),
							resDatabaseAttrURI:             validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrUpdatedAt:       validation.ToDiagFunc(validation.NoZeroValues),
							resDatabaseAttrKafka(resDatabaseAttrKafkaEnableCertAuth):  validateString("false"),
							resDatabaseAttrKafka(resDatabaseAttrKafkaEnableSASLAuth):  validateString("true"),
							resDatabaseAttrKafka(resDatabaseAttrKafkaIPFilter) + ".#": validateString("0"),
							resDatabaseAttrKafka(resDatabaseAttrKafkaSettings):        validation.ToDiagFunc(validation.NoZeroValues),
						},
						s[0].Attributes)
				},
			},
		},
	})
}
