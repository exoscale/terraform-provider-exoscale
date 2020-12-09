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
	testAccDataSourceNLBZone           = testZoneName
	testAccDataSourceNLBName           = testPrefix + "-" + testRandomString()
	testAccDataSourceNLBDescription    = testDescription
	testAccDataSourceNLBResourceConfig = fmt.Sprintf(`
resource "exoscale_nlb" "test" {
  zone        = "%s"
  name        = "%s"
  description = "%s"
}`,
		testAccDataSourceNLBZone,
		testAccDataSourceNLBName,
		testAccDataSourceNLBDescription,
	)
)

func TestAccDataSourceNLB(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`%s
data "exoscale_nlb" "test" {
  zone = exoscale_nlb.test.zone
}`,
					testAccDataSourceNLBResourceConfig),
				ExpectError: regexp.MustCompile("either name or id must be specified"),
			},
			{
				Config: fmt.Sprintf(`%s
data "exoscale_nlb" "by-id" {
  zone = exoscale_nlb.test.zone
  id = exoscale_nlb.test.id
}`,
					testAccDataSourceNLBResourceConfig),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceNLBAttributes("data.exoscale_nlb.by-id", testAttrs{
						"zone":        ValidateString(testAccDataSourceNLBZone),
						"id":          ValidateUUID(),
						"name":        ValidateString(testAccDataSourceNLBName),
						"description": ValidateString(testAccDataSourceNLBDescription),
						"created_at":  ValidateStringNot(""),
						"state":       ValidateStringNot(""),
						"ip_address":  ValidateIPv4String(),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`%s
data "exoscale_nlb" "by-name" {
  zone = exoscale_nlb.test.zone
  name = exoscale_nlb.test.name
}`,
					testAccDataSourceNLBResourceConfig),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceNLBAttributes("data.exoscale_nlb.by-name", testAttrs{
						"zone":        ValidateString(testAccDataSourceNLBZone),
						"id":          ValidateUUID(),
						"name":        ValidateString(testAccDataSourceNLBName),
						"description": ValidateString(testAccDataSourceNLBDescription),
					}),
				),
			},
		},
	})
}

func testAccDataSourceNLBAttributes(ds string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return checkResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("exoscale_nlb data source not found in the state")
	}
}
