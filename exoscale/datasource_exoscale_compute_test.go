package exoscale

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var (
	testAccDataSourceComputeSSHKeyName   = testPrefix + "-" + testRandomString()
	testAccDataSourceComputeNetworkName  = testPrefix + "-" + testRandomString()
	testAccDataSourceComputeTemplateName = testInstanceTemplateName
	testAccDataSourceComputeName         = testPrefix + "-" + testRandomString()
	testAccDataSourceComputeSize         = "Small"
	testAccDataSourceComputeDiskSize     = "15"

	testAccDataSourceComputeCreate = fmt.Sprintf(`
resource "exoscale_ssh_keypair" "key" {
  name = "%s"
}
  
resource "exoscale_compute" "vm" {
  zone = "ch-gva-2"
  hostname = "%s"
  template = "%s"
  size = "%s"
  disk_size = "%s"
  key_pair = exoscale_ssh_keypair.key.name
  ip6 = true
  tags = {
  	test = "acceptance"
  }
}
  
resource "exoscale_network" "net" {
  zone = "ch-gva-2"
  name = "%s"
  start_ip = "10.0.0.50"
  end_ip = "10.0.0.250"
  netmask = "255.255.255.0"
}
  
resource "exoscale_nic" "nic" {
  compute_id = exoscale_compute.vm.id
  network_id = exoscale_network.net.id
}
`,
		testAccDataSourceComputeSSHKeyName,
		testAccDataSourceComputeName,
		testAccDataSourceComputeTemplateName,
		testAccDataSourceComputeSize,
		testAccDataSourceComputeDiskSize,
		testAccDataSourceComputeNetworkName,
	)
)

func TestAccDatasourceCompute(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_compute" "compute" {
  hostname = "${exoscale_compute.vm.hostname}"
}
`, testAccDataSourceComputeCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceComputeAttributes(
						testAttrs{
							"template":  ValidateString(testAccDataSourceComputeTemplateName),
							"hostname":  ValidateString(testAccDataSourceComputeName),
							"size":      ValidateString(testAccDataSourceComputeSize),
							"disk_size": ValidateString(testAccDataSourceComputeDiskSize),
						},
					),
				),
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_compute" "compute" {
	id = "${exoscale_compute.vm.id}"
}`, testAccDataSourceComputeCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceComputeAttributes(
						testAttrs{
							"template":  ValidateString(testAccDataSourceComputeTemplateName),
							"hostname":  ValidateString(testAccDataSourceComputeName),
							"size":      ValidateString(testAccDataSourceComputeSize),
							"disk_size": ValidateString(testAccDataSourceComputeDiskSize),
						},
					),
				),
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_compute" "compute" {
	tags = "${exoscale_compute.vm.tags}"
}`, testAccDataSourceComputeCreate),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceComputeAttributes(
						testAttrs{
							"template":  ValidateString(testAccDataSourceComputeTemplateName),
							"hostname":  ValidateString(testAccDataSourceComputeName),
							"size":      ValidateString(testAccDataSourceComputeSize),
							"disk_size": ValidateString(testAccDataSourceComputeDiskSize),
						},
					),
				),
			},
		},
	})
}

func testAccDataSourceComputeAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_compute" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("compute data source not found in the state")
	}
}
