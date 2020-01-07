package exoscale

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var testAccDataSourceDomainName = testPrefix + "-" + testRandomString() + ".net"

func TestAccDataSourceDomain(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "exoscale_domain" "exo" {
  name = "%s"
}
data "exoscale_domain" "domain" {
  name = exoscale_domain.exo.name
}`, testAccDataSourceDomainName),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceDomainAttributes(testAttrs{
						"name": ValidateString(testAccDataSourceDomainName),
					}),
				),
			},
		},
	})
}

func testAccDataSourceDomainAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_domain" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("exoscale_domain data source not found in the state")
	}
}
