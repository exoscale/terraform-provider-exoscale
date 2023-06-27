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
	testAccDataSourceComputeZone        = testZoneName
	testAccDataSourceComputeTagValue    = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceComputeNetworkName = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceComputeTemplate    = testInstanceTemplateName
	testAccDataSourceComputeName        = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceComputeSize        = "Small"
	testAccDataSourceComputeDiskSize    = "15"

	testAccDataSourceComputeAttrs = testAttrs{
		"cpu":                            validation.ToDiagFunc(validation.NoZeroValues),
		"created":                        validation.ToDiagFunc(validation.NoZeroValues),
		"disk_size":                      validateString(testAccDataSourceComputeDiskSize),
		"hostname":                       validateString(testAccDataSourceComputeName),
		"id":                             validation.ToDiagFunc(validation.NoZeroValues),
		"ip6_address":                    validation.ToDiagFunc(validation.IsIPv6Address),
		"ip_address":                     validation.ToDiagFunc(validation.IsIPv4Address),
		"memory":                         validation.ToDiagFunc(validation.NoZeroValues),
		"private_network_ip_addresses.#": validateString("1"),
		"size":                           validateString(testAccDataSourceComputeSize),
		"state":                          validateString("Running"),
		"tags.test":                      validateString(testAccDataSourceComputeTagValue),
		"template":                       validateString(testAccDataSourceComputeTemplate),
		"zone":                           validateString(testAccDataSourceComputeZone),
	}

	testAccDataSourceComputeCreate = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_compute" "test" {
  zone = local.zone
  hostname = "%s"
  template = "%s"
  size = "%s"
  disk_size = "%s"
  ip6 = true
  tags = {
    test = "%s"
  }
}
  
resource "exoscale_network" "test" {
  zone = local.zone
  name = "%s"
  start_ip = "10.0.0.50"
  end_ip = "10.0.0.250"
  netmask = "255.255.255.0"
}
  
resource "exoscale_nic" "test" {
  compute_id = exoscale_compute.test.id
  network_id = exoscale_network.test.id
}
`,
		testAccDataSourceComputeZone,
		testAccDataSourceComputeName,
		testAccDataSourceComputeTemplate,
		testAccDataSourceComputeSize,
		testAccDataSourceComputeDiskSize,
		testAccDataSourceComputeTagValue,
		testAccDataSourceComputeNetworkName,
	)
)

func TestAccDatasourceCompute(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`%s
data "exoscale_compute" "error" {
}`, testAccDataSourceComputeCreate),
				ExpectError: regexp.MustCompile("either hostname, id or tags must be specified"),
			},
			{
				Config: fmt.Sprintf(`%s
data "exoscale_compute" "by-hostname" {
  hostname = exoscale_compute.test.hostname
  depends_on = [exoscale_nic.test]
}`, testAccDataSourceComputeCreate),
				Check: testAccDataSourceComputeAttributes("data.exoscale_compute.by-hostname",
					testAccDataSourceComputeAttrs),
			},
			{
				Config: fmt.Sprintf(`%s
data "exoscale_compute" "by-id" {
  id = exoscale_compute.test.id
  depends_on = [exoscale_nic.test]
}`, testAccDataSourceComputeCreate),
				Check: testAccDataSourceComputeAttributes("data.exoscale_compute.by-id",
					testAccDataSourceComputeAttrs),
			},
			{
				Config: fmt.Sprintf(`%s
data "exoscale_compute" "by-tags" {
  tags = exoscale_compute.test.tags
  depends_on = [exoscale_nic.test]
}`, testAccDataSourceComputeCreate),
				Check: testAccDataSourceComputeAttributes("data.exoscale_compute.by-tags",
					testAccDataSourceComputeAttrs),
			},
		},
	})
}

func testAccDataSourceComputeAttributes(ds string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, rs := range s.RootModule().Resources {
			if name == ds {
				return checkResourceAttributes(expected, rs.Primary.Attributes)
			}
		}

		return errors.New("compute data source not found in the state")
	}
}
