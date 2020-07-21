package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var (
	testAccDataSourceNetworkZone           = testZoneName
	testAccDataSourceNetworkName           = testPrefix + "-" + testRandomString()
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
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
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
						"zone":        ValidateString(testAccDataSourceNetworkZone),
						"id":          ValidateUUID(),
						"name":        ValidateString(testAccDataSourceNetworkName),
						"description": ValidateString(testAccDataSourceNetworkDescription),
						"start_ip":    ValidateString(testAccDataSourceNetworkStartIP),
						"end_ip":      ValidateString(testAccDataSourceNetworkEndIP),
						"netmask":     ValidateString(testAccDataSourceNetworkNetmask),
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
						"zone":        ValidateString(testAccDataSourceNetworkZone),
						"id":          ValidateUUID(),
						"name":        ValidateString(testAccDataSourceNetworkName),
						"description": ValidateString(testAccDataSourceNetworkDescription),
						"start_ip":    ValidateString(testAccDataSourceNetworkStartIP),
						"end_ip":      ValidateString(testAccDataSourceNetworkEndIP),
						"netmask":     ValidateString(testAccDataSourceNetworkNetmask),
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
