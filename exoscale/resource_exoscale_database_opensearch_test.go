package exoscale

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	"github.com/exoscale/egoscale/v2/oapi"
)

func TestAccResourceDatabase_Opensearch(t *testing.T) {
	var cfgTmpl = `resource "exoscale_database" "%s" {
    maintenance_dow =        "%s"
    maintenance_time =       "%s"
    name =                   "%s"
    plan =                   "hobbyist-2"
    termination_protection = %s
    type =                   "opensearch"
    zone = 	"%s"

    opensearch {
        index_pattern {
            max_index_count =   %s
            pattern =           "log.?"
            sorting_algorithm = "alphabetical"
        }

        index_pattern {
            max_index_count =   12
            pattern =           "internet.*"
            sorting_algorithm = "creation_date"
        }

        index_template {
            mapping_nested_objects_limit = 5
            number_of_replicas =           4
            number_of_shards =             3
        }

        ip_filter = [%s]
        keep_index_refresh_interval = true
         %s
        dashboards {
            enabled =            true
            max_old_space_size = %s
            request_timeout =    %s
        }

        settings = jsonencode({
        })
        version =  1
    }
}
`

	var (
		resName   = "exoscale_database." + testAccResourceDatabaseName
		cfgCreate = fmt.Sprintf(cfgTmpl, testAccResourceDatabaseName, testAccResourceDatabaseMaintenanceDOW, testAccResourceDatabaseMaintenanceTime, testAccResourceDatabaseName, "true", testZoneName, "2", "\"0.0.0.0/0\"", "", "129", "30001")
		// NOTE: Replace "" with "max_index_count = 4" when upstream bug is fixed.
		cfgUpdate          = fmt.Sprintf(cfgTmpl, testAccResourceDatabaseName, testAccResourceDatabaseMaintenanceDOWUpdated, testAccResourceDatabaseMaintenanceTimeUpdated, testAccResourceDatabaseName, "false", testZoneName, "6", "\"1.1.1.1/32\"", "", "132", "30006")
		databaseServiceInt interface{}
	)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceDatabaseDestroy("opensearch", testAccResourceDatabaseName),
		Steps: []resource.TestStep{
			{
				// Create
				Config: cfgCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceDatabaseExists(resName, &databaseServiceInt),
					func(s *terraform.State) error {
						a := assert.New(t)

						databaseService := databaseServiceInt.(*oapi.DbaasServiceOpensearch)

						a.True(*databaseService.TerminationProtection)
						a.Equal("1", *databaseService.Version)
						a.True(*databaseService.KeepIndexRefreshInterval)
						// NOTE: Uncomment when upstream bug is fixed
						// a.Equal(int64(0), *databaseService.MaxIndexCount)
						a.Equal(int64(2), *(*databaseService.IndexPatterns)[0].MaxIndexCount)
						a.Equal(2, len(*databaseService.IndexPatterns))
						a.Equal("log.?", *(*databaseService.IndexPatterns)[0].Pattern)
						a.Equal(oapi.DbaasServiceOpensearchIndexPatternsSortingAlgorithm("alphabetical"), *(*databaseService.IndexPatterns)[0].SortingAlgorithm)
						a.Equal(int64(5), *databaseService.IndexTemplate.MappingNestedObjectsLimit)
						a.Equal(int64(4), *databaseService.IndexTemplate.NumberOfReplicas)
						a.Equal(int64(3), *databaseService.IndexTemplate.NumberOfShards)
						a.Equal(true, *databaseService.OpensearchDashboards.Enabled)
						a.Equal(int64(129), *databaseService.OpensearchDashboards.MaxOldSpaceSize)
						a.Equal(int64(30001), *databaseService.OpensearchDashboards.OpensearchRequestTimeout)

						return nil
					},
					checkResourceState(resName, checkResourceStateValidateAttributes(testAttrs{
						resDatabaseAttrCreatedAt:                                   validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrDiskSize:                                    validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrName:                                        validateString(testAccResourceDatabaseName),
						resDatabaseAttrNodeCPUs:                                    validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrNodeMemory:                                  validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrNodes:                                       validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrPlan:                                        validateString("hobbyist-2"),
						resDatabaseAttrState:                                       validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrTerminationProtection:                       validation.ToDiagFunc(validation.NoZeroValues), //
						resDatabaseAttrType:                                        validateString("opensearch"),
						resDatabaseAttrUpdatedAt:                                   validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrMaintenanceDOW:                              validateString(testAccResourceDatabaseMaintenanceDOW),
						resDatabaseAttrMaintenanceTime:                             validateString(testAccResourceDatabaseMaintenanceTime),
						"opensearch.0." + resDatabaseAttrOpensearchIPFilter + ".0": validateString("0.0.0.0/0"),
						"opensearch.0." + resDatabaseAttrOpensearchKeepIndexRefreshInterval:                                                                             validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchMaxIndexCount:                                                                                        validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchVersion:                                                                                              validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchOpensearchSettings:                                                                                   validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".0." + resDatabaseAttrOpensearchMaxIndexCount:                                       validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".0." + resDatabaseAttrOpensearchIndexPatternsPattern:                                validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".0." + resDatabaseAttrOpensearchIndexPatternsSortingAlgorithm:                       validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".1." + resDatabaseAttrOpensearchMaxIndexCount:                                       validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".1." + resDatabaseAttrOpensearchIndexPatternsPattern:                                validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".1." + resDatabaseAttrOpensearchIndexPatternsSortingAlgorithm:                       validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".#":                                                                                 validateString("2"),
						"opensearch.0." + resDatabaseAttrOpensearchIndexTemplate + ".0." + resDatabaseAttrOpensearchIndexTemplateMappingNestedObjectsLimit:              validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexTemplate + ".0." + resDatabaseAttrOpensearchIndexTemplateNumberOfReplicas:                       validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexTemplate + ".0." + resDatabaseAttrOpensearchIndexTemplateNumberOfShards:                         validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchOpensearchDashboards + ".0." + resDatabaseAttrOpensearchOpensearchDashboardsEnabled:                  validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchOpensearchDashboards + ".0." + resDatabaseAttrOpensearchOpensearchDashboardsMaxOldSpaceSize:          validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchOpensearchDashboards + ".0." + resDatabaseAttrOpensearchOpensearchDashboardsOpensearchRequestTimeout: validation.ToDiagFunc(validation.NoZeroValues),
					})),
				),
			},
			{
				// Update
				Config: cfgUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceDatabaseExists(resName, &databaseServiceInt),
					func(s *terraform.State) error {
						a := assert.New(t)

						databaseService := databaseServiceInt.(*oapi.DbaasServiceOpensearch)

						a.False(*databaseService.TerminationProtection)
						a.Equal("1", *databaseService.Version)
						a.True(*databaseService.KeepIndexRefreshInterval)
						// NOTE: Uncomment when upstream bug is fixed
						// a.Equal(int64(4), *databaseService.MaxIndexCount)
						a.Equal(2, len(*databaseService.IndexPatterns))
						a.Equal(int64(6), *(*databaseService.IndexPatterns)[0].MaxIndexCount)
						a.Equal("log.?", *(*databaseService.IndexPatterns)[0].Pattern)
						a.Equal(oapi.DbaasServiceOpensearchIndexPatternsSortingAlgorithm("alphabetical"), *(*databaseService.IndexPatterns)[0].SortingAlgorithm)
						a.Equal(int64(5), *databaseService.IndexTemplate.MappingNestedObjectsLimit)
						a.Equal(int64(4), *databaseService.IndexTemplate.NumberOfReplicas)
						a.Equal(int64(3), *databaseService.IndexTemplate.NumberOfShards)
						a.Equal(true, *databaseService.OpensearchDashboards.Enabled)
						a.Equal(int64(132), *databaseService.OpensearchDashboards.MaxOldSpaceSize)
						a.Equal(int64(30006), *databaseService.OpensearchDashboards.OpensearchRequestTimeout)

						time.Sleep(10 * time.Second)

						return nil
					},
					checkResourceState(resName, checkResourceStateValidateAttributes(testAttrs{
						resDatabaseAttrCreatedAt:                                   validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrDiskSize:                                    validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrName:                                        validateString(testAccResourceDatabaseName),
						resDatabaseAttrNodeCPUs:                                    validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrNodeMemory:                                  validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrNodes:                                       validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrPlan:                                        validateString("hobbyist-2"),
						resDatabaseAttrState:                                       validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrTerminationProtection:                       validation.ToDiagFunc(validation.NoZeroValues), //
						resDatabaseAttrType:                                        validateString("opensearch"),
						resDatabaseAttrUpdatedAt:                                   validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrMaintenanceDOW:                              validateString(testAccResourceDatabaseMaintenanceDOWUpdated),
						resDatabaseAttrMaintenanceTime:                             validateString(testAccResourceDatabaseMaintenanceTimeUpdated),
						"opensearch.0." + resDatabaseAttrOpensearchIPFilter + ".0": validateString("1.1.1.1/32"),
						"opensearch.0." + resDatabaseAttrOpensearchIPFilter + ".#": validateString("1"),
						"opensearch.0." + resDatabaseAttrOpensearchKeepIndexRefreshInterval:                                                                             validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchMaxIndexCount:                                                                                        validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchVersion:                                                                                              validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchOpensearchSettings:                                                                                   validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".0." + resDatabaseAttrOpensearchMaxIndexCount:                                       validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".0." + resDatabaseAttrOpensearchIndexPatternsPattern:                                validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".0." + resDatabaseAttrOpensearchIndexPatternsSortingAlgorithm:                       validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".1." + resDatabaseAttrOpensearchMaxIndexCount:                                       validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".1." + resDatabaseAttrOpensearchIndexPatternsPattern:                                validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".1." + resDatabaseAttrOpensearchIndexPatternsSortingAlgorithm:                       validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".#":                                                                                 validateString("2"),
						"opensearch.0." + resDatabaseAttrOpensearchIndexTemplate + ".0." + resDatabaseAttrOpensearchIndexTemplateMappingNestedObjectsLimit:              validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexTemplate + ".0." + resDatabaseAttrOpensearchIndexTemplateNumberOfReplicas:                       validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexTemplate + ".0." + resDatabaseAttrOpensearchIndexTemplateNumberOfShards:                         validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchOpensearchDashboards + ".0." + resDatabaseAttrOpensearchOpensearchDashboardsEnabled:                  validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchOpensearchDashboards + ".0." + resDatabaseAttrOpensearchOpensearchDashboardsMaxOldSpaceSize:          validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchOpensearchDashboards + ".0." + resDatabaseAttrOpensearchOpensearchDashboardsOpensearchRequestTimeout: validation.ToDiagFunc(validation.NoZeroValues),
					})),
				),
			},
			{
				// Import
				ResourceName: resName,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", testAccResourceDatabaseName, testZoneName), nil
					}
				}(),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(testAttrs{
						resDatabaseAttrCreatedAt:                                   validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrDiskSize:                                    validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrName:                                        validateString(testAccResourceDatabaseName),
						resDatabaseAttrNodeCPUs:                                    validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrNodeMemory:                                  validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrNodes:                                       validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrPlan:                                        validateString("hobbyist-2"),
						resDatabaseAttrState:                                       validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrTerminationProtection:                       validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrType:                                        validateString("opensearch"),
						resDatabaseAttrUpdatedAt:                                   validation.ToDiagFunc(validation.NoZeroValues),
						resDatabaseAttrMaintenanceDOW:                              validateString(testAccResourceDatabaseMaintenanceDOWUpdated),
						resDatabaseAttrMaintenanceTime:                             validateString(testAccResourceDatabaseMaintenanceTimeUpdated),
						"opensearch.0." + resDatabaseAttrOpensearchIPFilter + ".0": validateString("1.1.1.1/32"),
						"opensearch.0." + resDatabaseAttrOpensearchKeepIndexRefreshInterval:                                                                             validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchMaxIndexCount:                                                                                        validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchVersion:                                                                                              validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchOpensearchSettings:                                                                                   validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".0." + resDatabaseAttrOpensearchMaxIndexCount:                                       validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".0." + resDatabaseAttrOpensearchIndexPatternsPattern:                                validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".0." + resDatabaseAttrOpensearchIndexPatternsSortingAlgorithm:                       validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".1." + resDatabaseAttrOpensearchMaxIndexCount:                                       validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".1." + resDatabaseAttrOpensearchIndexPatternsPattern:                                validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexPatterns + ".1." + resDatabaseAttrOpensearchIndexPatternsSortingAlgorithm:                       validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexTemplate + ".0." + resDatabaseAttrOpensearchIndexTemplateMappingNestedObjectsLimit:              validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexTemplate + ".0." + resDatabaseAttrOpensearchIndexTemplateNumberOfReplicas:                       validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchIndexTemplate + ".0." + resDatabaseAttrOpensearchIndexTemplateNumberOfShards:                         validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchOpensearchDashboards + ".0." + resDatabaseAttrOpensearchOpensearchDashboardsEnabled:                  validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchOpensearchDashboards + ".0." + resDatabaseAttrOpensearchOpensearchDashboardsMaxOldSpaceSize:          validation.ToDiagFunc(validation.NoZeroValues),
						"opensearch.0." + resDatabaseAttrOpensearchOpensearchDashboards + ".0." + resDatabaseAttrOpensearchOpensearchDashboardsOpensearchRequestTimeout: validation.ToDiagFunc(validation.NoZeroValues),
					},
						s[0].Attributes)
				},
			},
		},
	})
}
