package exoscale

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccDatasourceDomain(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
%s
data "exoscale_domain" "domain" {
  name = "${exoscale_domain.exo.name}"
}`, testAccDNSDomainCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccDatasourceDomainAttributes(testAttrs{
						"name": ValidateString(testAccResourceDomainName),
					}),
				),
			},
		},
	})
}

func testAccDatasourceDomainAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_domain" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("exoscale_domain datasource not found in the state")
	}
}
