package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var testAccDataSourceAffinityName = testPrefix + "-" + testRandomString()

func TestAccDataSourceAffinity(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
data "exoscale_affinity" "test" {
}`,
				ExpectError: regexp.MustCompile("either name or id must be specified"),
			},
			{
				Config: fmt.Sprintf(`
resource "exoscale_affinity" "test" {
  name = "%s"
}

data "exoscale_affinity" "by-id" {
  id = exoscale_affinity.test.id
}`, testAccDataSourceAffinityName),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAffinityAttributes(testAttrs{
						"id":   ValidateUUID(),
						"name": ValidateString(testAccDataSourceAffinityName),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "exoscale_affinity" "test" {
  name = "%s"
}

data "exoscale_affinity" "by-name" {
  name = exoscale_affinity.test.name
}`, testAccDataSourceAffinityName),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAffinityAttributes(testAttrs{
						"id":   ValidateUUID(),
						"name": ValidateString(testAccDataSourceAffinityName),
					}),
				),
			},
		},
	})
}

func testAccDataSourceAffinityAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_affinity" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("exoscale_affinity data source not found in the state")
	}
}
