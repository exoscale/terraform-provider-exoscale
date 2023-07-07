package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var (
	testAccDataSourcePrivateNetworkDescription = acctest.RandString(10)
	testAccDataSourcePrivateNetworkEndIP       = "10.0.0.50"
	testAccDataSourcePrivateNetworkLabelValue  = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourcePrivateNetworkName        = acctest.RandString(10)
	testAccDataSourcePrivateNetworkNetmask     = "255.255.0.0"
	testAccDataSourcePrivateNetworkStartIP     = "10.0.0.10"
	testAccDataSourcePrivateNetworkZone        = testZoneName
)

func TestAccDataSourcePrivateNetwork(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      `data "exoscale_private_network" "test" { zone = "" }`,
				ExpectError: regexp.MustCompile("either name or id must be specified"),
			},
			{
				Config: fmt.Sprintf(`
resource "exoscale_private_network" "test" {
  zone = "%s"
  name = "%s"
}

data "exoscale_private_network" "by-id" {
  zone = exoscale_private_network.test.zone
  id   = exoscale_private_network.test.id
}`,
					testAccDataSourcePrivateNetworkZone,
					testAccDataSourcePrivateNetworkName,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourcePrivateNetworkAttributes("data.exoscale_private_network.by-id", testAttrs{
						dsPrivateNetworkAttrID:   validation.ToDiagFunc(validation.IsUUID),
						dsPrivateNetworkAttrName: validateString(testAccDataSourcePrivateNetworkName),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "exoscale_private_network" "test" {
  zone = "%s"
  name = "%s"
}

data "exoscale_private_network" "by-name" {
  zone = exoscale_private_network.test.zone
  name = exoscale_private_network.test.name
}`,
					testAccDataSourcePrivateNetworkZone,
					testAccDataSourcePrivateNetworkName,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourcePrivateNetworkAttributes("data.exoscale_private_network.by-name", testAttrs{
						dsPrivateNetworkAttrID:   validation.ToDiagFunc(validation.IsUUID),
						dsPrivateNetworkAttrName: validateString(testAccDataSourcePrivateNetworkName),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "exoscale_private_network" "test" {
  zone        = "%s"
  name        = "%s"
  description = "%s"
  start_ip    = "%s"
  end_ip      = "%s"
  netmask     = "%s"
  labels = {
    test = "%s"
  }
}

data "exoscale_private_network" "test" {
  zone = exoscale_private_network.test.zone
  name = exoscale_private_network.test.name
}`,
					testAccDataSourcePrivateNetworkZone,
					testAccDataSourcePrivateNetworkName,
					testAccDataSourcePrivateNetworkDescription,
					testAccDataSourcePrivateNetworkStartIP,
					testAccDataSourcePrivateNetworkEndIP,
					testAccDataSourcePrivateNetworkNetmask,
					testAccDataSourcePrivateNetworkLabelValue,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAntiAffinityGroupAttributes("data.exoscale_private_network.test", testAttrs{
						dsPrivateNetworkAttrDescription:      validateString(testAccDataSourcePrivateNetworkDescription),
						dsPrivateNetworkAttrEndIP:            validateString(testAccDataSourcePrivateNetworkEndIP),
						dsPrivateNetworkAttrName:             validateString(testAccDataSourcePrivateNetworkName),
						dsPrivateNetworkAttrNetmask:          validateString(testAccDataSourcePrivateNetworkNetmask),
						dsPrivateNetworkAttrStartIP:          validateString(testAccDataSourcePrivateNetworkStartIP),
						dsPrivateNetworkAttrLabels + ".test": validateString(testAccDataSourcePrivateNetworkLabelValue),
					}),
				),
			},
		},
	})
}

func testAccDataSourcePrivateNetworkAttributes(ds string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return checkResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("exoscale_private_network data source not found in the state")
	}
}
