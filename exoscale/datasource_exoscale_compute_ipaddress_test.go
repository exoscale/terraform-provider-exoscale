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
	testAccDataSourceIPAddressZoneName    = testZoneName
	testAccDataSourceIPAddressDescription = testDescription

	testAccDataSourceIPAddressConfigCreate = fmt.Sprintf(`
resource "exoscale_ipaddress" "eip" {
  zone = "%s"
  description = "%s"
  tags = {
    test = "acceptance"
  }
}
`,
		testAccDataSourceIPAddressZoneName,
		testAccDataSourceIPAddressDescription,
	)
)

func TestAccDatasourceComputeIPAddress(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
data "exoscale_compute_ipaddress" "ip_address" {
  zone = "ch-gva-2"
}`,
				ExpectError: regexp.MustCompile(`You must set at least one attribute "id", "ip_address", "tags" or "description"`),
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_compute_ipaddress" "ip_address" {
  zone = "ch-gva-2"
  id   = "${exoscale_ipaddress.eip.id}"
}`, testAccDataSourceIPAddressConfigCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceComputeIPAddressAttributes(testAttrs{
						"description": ValidateString(testAccDataSourceIPAddressDescription),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_compute_ipaddress" "ip_address" {
  zone        = "ch-gva-2"
  description = "${exoscale_ipaddress.eip.description}"
}`, testAccDataSourceIPAddressConfigCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceComputeIPAddressAttributes(testAttrs{
						"description": ValidateString(testAccDataSourceIPAddressDescription),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_compute_ipaddress" "ip_address" {
  zone       = "ch-gva-2"
  ip_address = "${exoscale_ipaddress.eip.ip_address}"
}`, testAccDataSourceIPAddressConfigCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceComputeIPAddressAttributes(testAttrs{
						"description": ValidateString(testAccDataSourceIPAddressDescription),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_compute_ipaddress" "ip_address" {
  zone       = "ch-gva-2"
  tags = "${exoscale_ipaddress.eip.tags}"
}`, testAccDataSourceIPAddressConfigCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceComputeIPAddressAttributes(testAttrs{
						"description": ValidateString(testAccDataSourceIPAddressDescription),
					}),
				),
			},
		},
	})
}

func testAccDataSourceComputeIPAddressAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_compute_ipaddress" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("compute_ipaddress data source not found in the state")
	}
}
