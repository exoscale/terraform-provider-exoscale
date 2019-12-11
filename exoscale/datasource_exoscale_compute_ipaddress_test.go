package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccDatasourceComputeIPAddress(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccIPAddressConfigCreate,
			},
			{
				Config: `
data "exoscale_compute_ipaddress" "ip_address" {
  zone = "ch-gva-2"
}`,
				ExpectError: regexp.MustCompile(`You must set at least one attribute "is", "ip_address" or "description"`),
			},
			{
				Config: fmt.Sprintf(`
data "exoscale_compute_ipaddress" "ubuntu_lts" {
  zone = "ch-gva-2"
  id   = "${resource.exoscale_ipaddress.eip.id}"
}`),
				Check: resource.ComposeTestCheckFunc(
					testAccDatasourceComputeTemplateAttributes(testAttrs{
						"description": ValidateString(testIPDescription1),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
data "exoscale_compute_ipaddress" "ubuntu_lts" {
  zone        = "ch-gva-2"
  description = "${resource.exoscale_ipaddress.eip.description}"
}`),
				Check: resource.ComposeTestCheckFunc(
					testAccDatasourceComputeTemplateAttributes(testAttrs{
						"description": ValidateString(testIPDescription1),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
data "exoscale_compute_ipaddress" "ubuntu_lts" {
  zone       = "ch-gva-2"
  ip_address = "${resource.exoscale_ipaddress.eip.ip_address}"
}`),
				Check: resource.ComposeTestCheckFunc(
					testAccDatasourceComputeTemplateAttributes(testAttrs{
						"description": ValidateString(testIPDescription1),
					}),
				),
			},
		},
	})
}

func testAccDatasourceComputeIPAddressAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_compute_ipaddress" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("compute_ipaddress datasource not found in the state")
	}
}
