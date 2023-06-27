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
	testAccDataSourceNLBZone           = testZoneName
	testAccDataSourceNLBName           = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceNLBDescription    = acctest.RandString(10)
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
						dsNLBAttrCreatedAt:   validation.ToDiagFunc(validation.NoZeroValues),
						dsNLBAttrDescription: validateString(testAccDataSourceNLBDescription),
						dsNLBAttrID:          validation.ToDiagFunc(validation.IsUUID),
						dsNLBAttrIPAddress:   validation.ToDiagFunc(validation.IsIPv4Address),
						dsNLBAttrName:        validateString(testAccDataSourceNLBName),
						dsNLBAttrState:       validation.ToDiagFunc(validation.NoZeroValues),
						dsNLBAttrZone:        validateString(testAccDataSourceNLBZone),
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
						dsNLBAttrCreatedAt:   validation.ToDiagFunc(validation.NoZeroValues),
						dsNLBAttrDescription: validateString(testAccDataSourceNLBDescription),
						dsNLBAttrID:          validation.ToDiagFunc(validation.IsUUID),
						dsNLBAttrIPAddress:   validation.ToDiagFunc(validation.IsIPv4Address),
						dsNLBAttrName:        validateString(testAccDataSourceNLBName),
						dsNLBAttrState:       validation.ToDiagFunc(validation.NoZeroValues),
						dsNLBAttrZone:        validateString(testAccDataSourceNLBZone),
					}),
				),
			},
		},
	})
}

func testAccDataSourceNLBAttributes(r string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ds, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("data source not found in the state")
		}

		return checkResourceAttributes(expected, ds.Primary.Attributes)
	}
}
