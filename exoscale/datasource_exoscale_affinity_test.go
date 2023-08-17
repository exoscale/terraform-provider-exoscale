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

var testAccDataSourceAffinityName = acctest.RandomWithPrefix(testPrefix)

func TestAccDataSourceAffinity(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
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
					testAccDataSourceAffinityAttributes("data.exoscale_affinity.by-id", testAttrs{
						"id":   validation.ToDiagFunc(validation.IsUUID),
						"name": validateString(testAccDataSourceAffinityName),
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
					testAccDataSourceAffinityAttributes("data.exoscale_affinity.by-name", testAttrs{
						"id":   validation.ToDiagFunc(validation.IsUUID),
						"name": validateString(testAccDataSourceAffinityName),
					}),
				),
			},
		},
	})
}

func testAccDataSourceAffinityAttributes(ds string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return checkResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("exoscale_affinity data source not found in the state")
	}
}
