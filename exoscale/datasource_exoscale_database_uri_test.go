package exoscale

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	testAccDataSourceDatabaseURIName = acctest.RandomWithPrefix(testPrefix)

	testAccDataSourceDatabaseURIPlanKafka      = "startup-2"
	testAccDataSourceDatabaseURIPlanMysql      = "startup-4"
	testAccDataSourceDatabaseURIPlanOpensearch = "startup-4"
	testAccDataSourceDatabaseURIPlanPg         = "startup-4"
	testAccDataSourceDatabaseURIPlanRedis      = "startup-4"
)

func TestAccDataSourceDatabaseURI(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_database" "test_kafka" {
  zone                   = local.zone
  type                   = "kafka"
  name                   = "%s"
  plan                   = "%s"
  termination_protection = false

  kafka {}
}

data "exoscale_database_uri" "test_kafka" {
  zone = local.zone
  name = exoscale_database.test_kafka.name
  type = exoscale_database.test_kafka.type
}
`,
					testZoneName,
					testAccDataSourceDatabaseURIName,
					testAccDataSourceDatabaseURIPlanKafka,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceDatabaseURIAttributes("data.exoscale_database_uri.test_kafka", testAttrs{
						dsDatabaseAttrURI: validation.ToDiagFunc(validation.NoZeroValues),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_database" "test_mysql" {
  zone                   = local.zone
  type                   = "mysql"
  name                   = "%s"
  plan                   = "%s"
  termination_protection = false

  mysql {}
}

data "exoscale_database_uri" "test_mysql" {
  zone = local.zone
  name = exoscale_database.test_mysql.name
  type = exoscale_database.test_mysql.type
}
`,
					testZoneName,
					testAccDataSourceDatabaseURIName,
					testAccDataSourceDatabaseURIPlanMysql,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceDatabaseURIAttributes("data.exoscale_database_uri.test_mysql", testAttrs{
						dsDatabaseAttrURI: validation.ToDiagFunc(validation.NoZeroValues),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_database" "test_opensearch" {
  zone                   = local.zone
  type                   = "opensearch"
  name                   = "%s"
  plan                   = "%s"
  termination_protection = false

  opensearch {
    dashboards {
        enabled = false
    }
  }
}

data "exoscale_database_uri" "test_opensearch" {
  zone = local.zone
  name = exoscale_database.test_opensearch.name
  type = exoscale_database.test_opensearch.type
}
`,
					testZoneName,
					testAccDataSourceDatabaseURIName,
					testAccDataSourceDatabaseURIPlanOpensearch,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceDatabaseURIAttributes("data.exoscale_database_uri.test_opensearch", testAttrs{
						dsDatabaseAttrURI: validation.ToDiagFunc(validation.NoZeroValues),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_database" "test_pg" {
  zone                   = local.zone
  type                   = "pg"
  name                   = "%s"
  plan                   = "%s"
  termination_protection = false

  pg {}
}

data "exoscale_database_uri" "test_pg" {
  zone = local.zone
  name = exoscale_database.test_pg.name
  type = exoscale_database.test_pg.type
}
`,
					testZoneName,
					testAccDataSourceDatabaseURIName,
					testAccDataSourceDatabaseURIPlanPg,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceDatabaseURIAttributes("data.exoscale_database_uri.test_pg", testAttrs{
						dsDatabaseAttrURI: validation.ToDiagFunc(validation.NoZeroValues),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_database" "test_redis" {
  zone                   = local.zone
  type                   = "redis"
  name                   = "%s"
  plan                   = "%s"
  termination_protection = false

  redis {}
}

data "exoscale_database_uri" "test_redis" {
  zone = local.zone
  name = exoscale_database.test_redis.name
  type = exoscale_database.test_redis.type
}
`,
					testZoneName,
					testAccDataSourceDatabaseURIName,
					testAccDataSourceDatabaseURIPlanRedis,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceDatabaseURIAttributes("data.exoscale_database_uri.test_redis", testAttrs{
						dsDatabaseAttrURI: validation.ToDiagFunc(validation.NoZeroValues),
					}),
				),
			},
		},
	})
}

func testAccDataSourceDatabaseURIAttributes(ds string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return checkResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("exoscale_database_uri data source not found in the state")
	}
}
