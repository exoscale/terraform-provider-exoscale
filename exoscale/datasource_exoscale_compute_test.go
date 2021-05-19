package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var (
	testAccDataSourceComputeZone        = testZoneName
	testAccDataSourceComputeTagValue    = testPrefix + "-" + testRandomString()
	testAccDataSourceComputeNetworkName = testPrefix + "-" + testRandomString()
	testAccDataSourceComputeTemplate    = testInstanceTemplateName
	testAccDataSourceComputeName        = testPrefix + "-" + testRandomString()
	testAccDataSourceComputeSize        = "Small"
	testAccDataSourceComputeDiskSize    = "15"

	testAccDataSourceComputeAttrs = testAttrs{
		"cpu":                            validation.NoZeroValues,
		"created":                        validation.NoZeroValues,
		"disk_size":                      ValidateString(testAccDataSourceComputeDiskSize),
		"hostname":                       ValidateString(testAccDataSourceComputeName),
		"id":                             validation.NoZeroValues,
		"ip6_address":                    validation.IsIPv6Address,
		"ip_address":                     validation.IsIPv4Address,
		"memory":                         validation.NoZeroValues,
		"private_network_ip_addresses.#": ValidateString("1"),
		"size":                           ValidateString(testAccDataSourceComputeSize),
		"state":                          ValidateString("Running"),
		"tags.test":                      ValidateString(testAccDataSourceComputeTagValue),
		"template":                       ValidateString(testAccDataSourceComputeTemplate),
		"zone":                           ValidateString(testAccDataSourceComputeZone),
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
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			/*
				For some reason, running these test steps without `ExpectNonEmptyPlan: true` triggers
				the following error:

				testing.go:684: Step 1 error: After applying this step and refreshing, the plan was not empty:

					DIFF:

					UPDATE: data.exoscale_compute.by-hostname
					  cpu:                          "" => "<computed>"
					  created:                      "" => "<computed>"
					  disk_size:                    "" => "<computed>"
					  hostname:                     "" => "test-terraform-exoscale-provider-tzo3jrm3fg"
					  ip6_address:                  "" => "<computed>"
					  ip_address:                   "" => "<computed>"
					  memory:                       "" => "<computed>"
					  private_network_ip_addresses: "" => "<computed>"
					  size:                         "" => "<computed>"
					  state:                        "" => "<computed>"
					  template:                     "" => "<computed>"
					  zone:                         "" => "<computed>"

				Which seems to me similar to the problem discussed in this GitHub issue:
					https://github.com/hashicorp/terraform/issues/20986

				Note: this problem only manifests itself during tests, not during actual usage of the
				data source (tested OK with the exact same configuration used in these tests).
			*/
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
				ExpectNonEmptyPlan: true,
			},
			{
				Config: fmt.Sprintf(`%s
data "exoscale_compute" "by-id" {
  id = exoscale_compute.test.id
  depends_on = [exoscale_nic.test]
}`, testAccDataSourceComputeCreate),
				Check: testAccDataSourceComputeAttributes("data.exoscale_compute.by-id",
					testAccDataSourceComputeAttrs),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: fmt.Sprintf(`%s
data "exoscale_compute" "by-tags" {
  tags = exoscale_compute.test.tags
  depends_on = [exoscale_nic.test]
}`, testAccDataSourceComputeCreate),
				Check: testAccDataSourceComputeAttributes("data.exoscale_compute.by-tags",
					testAccDataSourceComputeAttrs),
				ExpectNonEmptyPlan: true,
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
