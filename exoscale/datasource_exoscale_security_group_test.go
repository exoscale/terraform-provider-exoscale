package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var testAccDataSourceSecurityGroupName = testPrefix + "-" + testRandomString()

func TestAccDataSourceSecurityGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
data "exoscale_security_group" "test" {
}`,
				ExpectError: regexp.MustCompile("either name or id must be specified"),
			},
			{
				Config: fmt.Sprintf(`
resource "exoscale_security_group" "test" {
  name = "%s"
}

data "exoscale_security_group" "by-id" {
  id = exoscale_security_group.test.id
}`, testAccDataSourceSecurityGroupName),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceSecurityGroupAttributes("data.exoscale_security_group.by-id", testAttrs{
						"id":   ValidateUUID(),
						"name": ValidateString(testAccDataSourceSecurityGroupName),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "exoscale_security_group" "test" {
  name = "%s"
}

data "exoscale_security_group" "by-name" {
  name = exoscale_security_group.test.name
}`, testAccDataSourceSecurityGroupName),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceSecurityGroupAttributes("data.exoscale_security_group.by-name", testAttrs{
						"id":   ValidateUUID(),
						"name": ValidateString(testAccDataSourceSecurityGroupName),
					}),
				),
			},
		},
	})
}

func testAccDataSourceSecurityGroupAttributes(ds string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return checkResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("exoscale_security_group data source not found in the state")
	}
}
