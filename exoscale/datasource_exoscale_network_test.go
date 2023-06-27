package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var (
	testAccDataSourceNetworkZone           = testZoneName
	testAccDataSourceNetworkName           = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceNetworkDescription    = testDescription
	testAccDataSourceNetworkStartIP        = "10.0.0.10"
	testAccDataSourceNetworkEndIP          = "10.0.0.50"
	testAccDataSourceNetworkNetmask        = "255.255.0.0"
	testAccDataSourceNetworkResourceConfig = fmt.Sprintf(`
resource "exoscale_network" "test" {
  zone         = "%s"
  name         = "%s"
  display_text = "%s"
  start_ip     = "%s"
  end_ip       = "%s"
  netmask      = "%s"
}`,
		testAccDataSourceNetworkZone,
		testAccDataSourceNetworkName,
		testAccDataSourceNetworkDescription,
		testAccDataSourceNetworkStartIP,
		testAccDataSourceNetworkEndIP,
		testAccDataSourceNetworkNetmask)
)

func TestAccDataSourceNetwork(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`%s
data "exoscale_network" "test" {
  zone = exoscale_network.test.zone
}`,
					testAccDataSourceNetworkResourceConfig),
				ExpectError: regexp.MustCompile("either name or id must be specified"),
			},
			{
				Config: fmt.Sprintf(`%s
data "exoscale_network" "by-id" {
  zone = exoscale_network.test.zone
  id = exoscale_network.test.id
}`,
					testAccDataSourceNetworkResourceConfig),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceNetworkAttributes("data.exoscale_network.by-id", testAttrs{
						"zone":        validateString(testAccDataSourceNetworkZone),
						"id":          validation.ToDiagFunc(validation.IsUUID),
						"name":        validateString(testAccDataSourceNetworkName),
						"description": validateString(testAccDataSourceNetworkDescription),
						"start_ip":    validateString(testAccDataSourceNetworkStartIP),
						"end_ip":      validateString(testAccDataSourceNetworkEndIP),
						"netmask":     validateString(testAccDataSourceNetworkNetmask),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`%s
data "exoscale_network" "by-name" {
  zone = exoscale_network.test.zone
  name = exoscale_network.test.name
}`,
					testAccDataSourceNetworkResourceConfig),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceNetworkAttributes("data.exoscale_network.by-name", testAttrs{
						"zone":        validateString(testAccDataSourceNetworkZone),
						"id":          validation.ToDiagFunc(validation.IsUUID),
						"name":        validateString(testAccDataSourceNetworkName),
						"description": validateString(testAccDataSourceNetworkDescription),
						"start_ip":    validateString(testAccDataSourceNetworkStartIP),
						"end_ip":      validateString(testAccDataSourceNetworkEndIP),
						"netmask":     validateString(testAccDataSourceNetworkNetmask),
					}),
				),
			},
		},
	})
}

func testAccDataSourceNetworkAttributes(ds string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return checkResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("exoscale_network data source not found in the state")
	}
}
