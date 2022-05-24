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
	testAccDataSourceInstancePoolAttrAffinityGroupName = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceInstancePoolDescription           = acctest.RandString(10)
	testAccDataSourceInstancePoolDiskSize              = "10"
	testAccDataSourceInstancePoolInstancePrefix        = "test"
	testAccDataSourceInstancePoolInstanceType          = "standard.tiny"
	testAccDataSourceInstancePoolKeyPair               = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceInstancePoolLabelValue            = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceInstancePoolNetwork               = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceInstancePoolName                  = acctest.RandomWithPrefix(testPrefix)
	testAccDataSourceInstancePoolSize                  = "2"
	testAccDataSourceInstancePoolTemplateID            = testInstanceTemplateID
	testAccDataSourceInstancePoolUserData              = acctest.RandString(10)
)

func TestAccDataSourceInstancePool(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      `data "exoscale_instance_pool" "test" { zone = "lolnope" }`,
				ExpectError: regexp.MustCompile("either name or id must be specified"),
			},
			{
				Config: fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_affinity" "test" {
  name = "%s"
}

resource "exoscale_network" "test" {
  zone = local.zone
  name = "%s"
}

resource "exoscale_ssh_keypair" "test" {
  name = "%s"
}

resource "exoscale_ipaddress" "test" {
  zone = local.zone
}

resource "exoscale_instance_pool" "test" {
  zone = local.zone
  name = "%s"
  description = "%s"
  template_id = "%s"
  instance_type = "%s"
  instance_prefix = "%s"
  size = %s
  disk_size = %s
  ipv6 = false
  key_pair = exoscale_ssh_keypair.test.name
  affinity_group_ids = [exoscale_affinity.test.id]
  network_ids = [exoscale_network.test.id]
  elastic_ip_ids = [exoscale_ipaddress.test.id]
  user_data = "%s"
  labels = {
    test = "%s"
  }
}

data "exoscale_instance_pool" "by-id" {
  zone = exoscale_instance_pool.test.zone
  id   = exoscale_instance_pool.test.id
}`,
					testZoneName,
					testAccDataSourceInstancePoolAttrAffinityGroupName,
					testAccDataSourceInstancePoolNetwork,
					testAccDataSourceInstancePoolKeyPair,
					testAccDataSourceInstancePoolName,
					testAccDataSourceInstancePoolDescription,
					testAccDataSourceInstancePoolTemplateID,
					testAccDataSourceInstancePoolInstanceType,
					testAccDataSourceInstancePoolInstancePrefix,
					testAccDataSourceInstancePoolSize,
					testAccDataSourceInstancePoolDiskSize,
					testAccDataSourceInstancePoolUserData,
					testAccDataSourceInstancePoolLabelValue,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceInstancePoolAttributes("data.exoscale_instance_pool.by-id", testAttrs{
						"affinity_group_ids.#": validateString("1"),
						"affinity_group_ids.0": validation.ToDiagFunc(validation.IsUUID),
						"description":          validateString(testAccDataSourceInstancePoolDescription),
						"disk_size":            validateString(testAccDataSourceInstancePoolDiskSize),
						"instance_type":        validateComputeInstanceType,
						"instance_prefix":      validateString(testAccDataSourceInstancePoolInstancePrefix),
						"key_pair":             validateString(testAccDataSourceInstancePoolKeyPair),
						"labels.test":          validateString(testAccDataSourceInstancePoolLabelValue),
						"id":                   validation.ToDiagFunc(validation.IsUUID),
						"name":                 validateString(testAccDataSourceInstancePoolName),
						"network_ids.#":        validateString("1"),
						"network_ids.0":        validation.ToDiagFunc(validation.IsUUID),
						"size":                 validateString(testAccDataSourceInstancePoolSize),
						"state":                validateString("running"),
						"template_id":          validateString(testAccDataSourceInstancePoolTemplateID),
						"user_data":            validateString(testAccDataSourceInstancePoolUserData),
						"instances.#":          validateString("2"),
						"instances.0.id":       validation.ToDiagFunc(validation.IsUUID),
						"instances.1.id":       validation.ToDiagFunc(validation.IsUUID),
					}),
				),
			},
		},
	})
}

func testAccDataSourceInstancePoolAttributes(ds string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return checkResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("exoscale_instance_pool data source not found in the state")
	}
}
