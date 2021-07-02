package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	testAccDataSourceNLBZone           = testZoneName
	testAccDataSourceNLBName           = acctest.RandomWithPrefix(testPrefix)
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
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
						"id":          validation.ToDiagFunc(validation.IsUUID),
						"name":        ValidateString(testAccDataSourceNLBName),
						"description": ValidateString(testAccDataSourceNLBDescription),
						"created_at":  ValidateStringNot(""),
						"state":       ValidateStringNot(""),
						"ip_address":  validation.ToDiagFunc(validation.IsIPv4Address),
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
						"id":          validation.ToDiagFunc(validation.IsUUID),
						"name":        ValidateString(testAccDataSourceNLBName),
						"description": ValidateString(testAccDataSourceNLBDescription),
						"created_at":  ValidateStringNot(""),
						"state":       ValidateStringNot(""),
						"ip_address":  validation.ToDiagFunc(validation.IsIPv4Address),
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
