package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "exoscale_compute_ipaddress" "ip_address" {
  zone = "%s"
}`, testAccDataSourceIPAddressZoneName),
				ExpectError: regexp.MustCompile(`You must set at least one attribute "id", "ip_address", "tags" or "description"`),
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_compute_ipaddress" "ip_address" {
  zone = "%s"
  id   = "${exoscale_ipaddress.eip.id}"
}`,
					testAccDataSourceIPAddressConfigCreate,
					testAccDataSourceIPAddressZoneName,
				),
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
  zone        = "%s"
  description = "${exoscale_ipaddress.eip.description}"
}`,
					testAccDataSourceIPAddressConfigCreate,
					testAccDataSourceIPAddressZoneName,
				),
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
  zone       = "%s"
  ip_address = "${exoscale_ipaddress.eip.ip_address}"
}`,
					testAccDataSourceIPAddressConfigCreate,
					testAccDataSourceIPAddressZoneName,
				),
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
  zone = "%s"
  tags = "${exoscale_ipaddress.eip.tags}"
}`,
					testAccDataSourceIPAddressConfigCreate,
					testAccDataSourceIPAddressZoneName,
				),
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
