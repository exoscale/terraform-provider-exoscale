package exoscale

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	testAccDataSourceComputeInstanceListSecurityGroupName       = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceComputeInstanceListDiskSize          int64 = 10
	testAccDataSourceComputeInstanceListName                    = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceComputeInstanceListSSHKeyName              = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceComputeInstanceListType                    = "standard.tiny"

	testAccDataSourceComputeInstanceListConfig = fmt.Sprintf(`
locals {
  zone = "%s"
}
data "exoscale_compute_template" "ubuntu" {
  zone = local.zone
  name = "Linux Ubuntu 20.04 LTS 64-bit"
}
resource "exoscale_security_group" "test" {
  name = "%s"
}
resource "exoscale_ssh_key" "test" {
  name       = "%s"
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIB8bfA67mQWv4eGND/XVtPx1JW6RAqafub1lV1EcpB+b test"
}
resource "exoscale_compute_instance" "test" {
  zone                    = local.zone
  name                    = "%s"
  type                    = "%s"
  disk_size               = %d
  template_id             = data.exoscale_compute_template.ubuntu.id
  ipv6                    = true
  ssh_key                 = exoscale_ssh_key.test.name
  timeouts {
    delete = "10m"
  }
}
`,
		testZoneName,
		testAccDataSourceComputeInstanceListSecurityGroupName,
		testAccDataSourceComputeInstanceListSSHKeyName,
		testAccDataSourceComputeInstanceListName,
		testAccDataSourceComputeInstanceListType,
		testAccDataSourceComputeInstanceListDiskSize,
	)
)

func TestAccDataSourceComputeInstanceList(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:             testAccDataSourceComputeInstanceListConfig,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_compute_instance_list" "test" {
  zone = local.zone
}
`,
					testAccDataSourceComputeInstanceListConfig,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceComputeInstanceListAttributes("data.exoscale_compute_instance_list.test", testAttrs{
						"instances.#":         validateString("1"),
						"instances.0.name":    validateString(testAccDataSourceComputeInstanceListName),
						"instances.0.type":    validateString(testAccDataSourceComputeInstanceListType),
						"instances.0.ssh_key": validateString(testAccDataSourceComputeInstanceListSSHKeyName),
					}),
				),
			},
		},
	})
}

func testAccDataSourceComputeInstanceListAttributes(ds string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return checkResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("exoscale_compute_instance data source not found in the state")
	}
}
