package exoscale

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	testAccDataSourceInstancePoolListDiskSize     = "10"
	testAccDataSourceInstancePoolListInstanceType = "standard.tiny"
	testAccDataSourceInstancePoolListKeyPair      = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceInstancePoolListName         = acctest.RandomWithPrefix(testPrefix)
)

var testAccDataSourceInstancePoolListConfig = fmt.Sprintf(`
locals {
  zone = "%s"
	instance_type = "%s"
	disk_size = "%s"
}
resource "exoscale_ssh_keypair" "test" {
  name = "%s"
}
data "exoscale_compute_template" "ubuntu" {
  zone = local.zone
  name = "Linux Ubuntu 20.04 LTS 64-bit"
}
resource "exoscale_instance_pool" "test1" {
  zone = local.zone
  name = "%s"
  template_id = data.exoscale_compute_template.ubuntu.id
  instance_type = local.instance_type
  size = 1
  disk_size = local.disk_size
  key_pair = exoscale_ssh_keypair.test.name
}
resource "exoscale_instance_pool" "test2" {
  zone = local.zone
  name = "%s"
  template_id = data.exoscale_compute_template.ubuntu.id
  instance_type = local.instance_type
  size = 1
  disk_size = local.disk_size
  key_pair = exoscale_ssh_keypair.test.name
}`,
	testZoneName,
	testAccDataSourceInstancePoolListInstanceType,
	testAccDataSourceInstancePoolListDiskSize,
	testAccDataSourceInstancePoolListKeyPair,
	testAccDataSourceInstancePoolListName+"_1",
	testAccDataSourceInstancePoolListName+"_2",
)

func TestAccDataSourceInstancePoolList(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceInstancePoolListConfig,
			},
			{
				Config: fmt.Sprintf(`
%s
data "exoscale_instance_pool_list" "test" {
  zone = local.zone
}
`,
					testAccDataSourceInstancePoolListConfig,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceInstancePoolListAttributes("data.exoscale_instance_pool_list.test", testAttrs{
						"pools.#":             validateString("2"),
						"pools.0.id":          validation.ToDiagFunc(validation.NoZeroValues),
						"pools.0.instances.#": validateString("1"),
						"pools.1.id":          validation.ToDiagFunc(validation.NoZeroValues),
						"pools.1.instances.#": validateString("1"),
					}),
				),
			},
		},
	})
}

func testAccDataSourceInstancePoolListAttributes(ds string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return checkResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("exoscale_instance_pool_list data source not found in the state")
	}
}
