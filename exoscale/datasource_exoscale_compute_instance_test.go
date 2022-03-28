package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	testAccDataSourceComputeInstanceAntiAffinityGroupName       = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceComputeInstanceDiskSize              int64 = 10
	testAccDataSourceComputeInstanceLabelValue                  = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceComputeInstanceName                        = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceComputeInstancePrivateNetworkName          = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceComputeInstanceSSHKeyName                  = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceComputeInstanceSecurityGroupName           = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceComputeInstanceType                        = "standard.tiny"
	testAccDataSourceComputeInstanceUserData                    = acctest.RandString(10)

	testAccDataSourceComputeInstanceConfig = fmt.Sprintf(`
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

resource "exoscale_anti_affinity_group" "test" {
  name = "%s"
}

resource "exoscale_private_network" "test" {
  zone = local.zone
  name = "%s"
}

resource "exoscale_elastic_ip" "test" {
  zone = local.zone
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
  anti_affinity_group_ids = [exoscale_anti_affinity_group.test.id]
  security_group_ids      = [exoscale_security_group.test.id]
  elastic_ip_ids          = [exoscale_elastic_ip.test.id]
  user_data               = "%s"
  ssh_key                 = exoscale_ssh_key.test.name

  network_interface {
	network_id = exoscale_private_network.test.id
  }

  labels = {
    test = "%s"
  }

  timeouts {
    delete = "10m"
  }
}
`,
		testZoneName,
		testAccDataSourceComputeInstanceSecurityGroupName,
		testAccDataSourceComputeInstanceAntiAffinityGroupName,
		testAccDataSourceComputeInstancePrivateNetworkName,
		testAccDataSourceComputeInstanceSSHKeyName,
		testAccDataSourceComputeInstanceName,
		testAccDataSourceComputeInstanceType,
		testAccDataSourceComputeInstanceDiskSize,
		testAccDataSourceComputeInstanceUserData,
		testAccDataSourceComputeInstanceLabelValue,
	)
)

func TestAccDataSourceComputeInstance(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      `data "exoscale_compute_instance" "test" { zone = "lolnope" }`,
				ExpectError: regexp.MustCompile("either name or id must be specified"),
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_compute_instance" "by-id" {
  zone = local.zone
  id   = exoscale_compute_instance.test.id
}
`,
					testAccDataSourceComputeInstanceConfig,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceComputeInstanceAttributes("data.exoscale_compute_instance.by-id", testAttrs{
						dsComputeInstanceAttrID:   validation.ToDiagFunc(validation.IsUUID),
						dsComputeInstanceAttrName: validateString(testAccDataSourceComputeInstanceName),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_compute_instance" "by-name" {
  zone = local.zone
  name = exoscale_compute_instance.test.name
}
`,
					testAccDataSourceComputeInstanceConfig,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceComputeInstanceAttributes("data.exoscale_compute_instance.by-name", testAttrs{
						dsComputeInstanceAttrID:   validation.ToDiagFunc(validation.IsUUID),
						dsComputeInstanceAttrName: validateString(testAccDataSourceComputeInstanceName),
					}),
				),
			},
		},
	})
}

func testAccDataSourceComputeInstanceAttributes(ds string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return checkResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("exoscale_compute_instance data source not found in the state")
	}
}
