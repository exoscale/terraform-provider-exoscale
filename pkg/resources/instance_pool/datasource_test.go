package instance_pool_test

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/utils"
	"github.com/exoscale/terraform-provider-exoscale/utils/testutils"
)

var (
	dsAntiAffinityGroupName = acctest.RandomWithPrefix(testutils.Prefix)
	dsDescription           = acctest.RandString(10)
	dsDiskSize              = "10"
	dsInstancePrefix        = "test"
	dsInstanceType          = "standard.tiny"
	dsKeyPair               = acctest.RandomWithPrefix(testutils.Prefix)
	dsLabelValue            = acctest.RandomWithPrefix(testutils.Prefix)
	dsNetwork               = acctest.RandomWithPrefix(testutils.Prefix)
	dsName                  = acctest.RandomWithPrefix(testutils.Prefix)
	dsSize                  = "2"
	dsTemplateName          = testutils.TestInstanceTemplateName
	dsUserData              = acctest.RandString(10)
)

func testDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testutils.AccPreCheck(t) },
		ProviderFactories: testutils.Providers(),
		Steps: []resource.TestStep{
			{
				Config:      `data "exoscale_instance_pool" "test" { zone = "ch-gva-2" }`,
				ExpectError: regexp.MustCompile("either name or id must be specified"),
			},
			{
				Config: fmt.Sprintf(`
locals {
  zone = "%s"
}

data "exoscale_template" "template" {
  zone = local.zone
  name = "%s"
}

resource "exoscale_anti_affinity_group" "test" {
  name = "%s"
}

resource "exoscale_private_network" "test" {
  zone = local.zone
  name = "%s"
}

resource "exoscale_ssh_key" "test" {
  name = "%s"
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAINXaeIOCEq0WN7g0N0XB7aRZpyjjTjRTVsb386ZpCa6P test"
}

resource "exoscale_instance_pool" "test" {
  zone               = local.zone
  name               = "%s"
  description        = "%s"
  template_id        = data.exoscale_template.template.id
  instance_type      = "%s"
  instance_prefix    = "%s"
  size               = %s
  disk_size          = %s
  ipv6               = false
  key_pair           = exoscale_ssh_key.test.name
  anti_affinity_group_ids = [exoscale_anti_affinity_group.test.id]
  network_ids        = [exoscale_private_network.test.id]
  user_data          = "%s"

  labels = {
    test = "%s"
  }
}

data "exoscale_instance_pool" "by-id" {
  zone = exoscale_instance_pool.test.zone
  id   = exoscale_instance_pool.test.id
}`,
					testutils.TestZoneName,
					dsTemplateName,
					dsAntiAffinityGroupName,
					dsNetwork,
					dsKeyPair,
					dsName,
					dsDescription,
					dsInstanceType,
					dsInstancePrefix,
					dsSize,
					dsDiskSize,
					dsUserData,
					dsLabelValue,
				),
				Check: resource.ComposeTestCheckFunc(
					dsCheckAttrs("data.exoscale_instance_pool.by-id", testutils.TestAttrs{
						"anti_affinity_group_ids.#": testutils.ValidateString("1"),
						"anti_affinity_group_ids.0": validation.ToDiagFunc(validation.IsUUID),
						"description":               testutils.ValidateString(dsDescription),
						"disk_size":                 testutils.ValidateString(dsDiskSize),
						"instance_type":             utils.ValidateComputeInstanceType,
						"instance_prefix":           testutils.ValidateString(dsInstancePrefix),
						"key_pair":                  testutils.ValidateString(dsKeyPair),
						"labels.test":               testutils.ValidateString(dsLabelValue),
						"id":                        validation.ToDiagFunc(validation.IsUUID),
						"name":                      testutils.ValidateString(dsName),
						"network_ids.#":             testutils.ValidateString("1"),
						"network_ids.0":             validation.ToDiagFunc(validation.IsUUID),
						"size":                      testutils.ValidateString(dsSize),
						// NOTE: state is unreliable atm, improvement suggested in 54808
						// "state":                testutils.ValidateString("running"),
						"template_id":    validation.ToDiagFunc(validation.IsUUID),
						"user_data":      testutils.ValidateString(dsUserData),
						"instances.#":    testutils.ValidateString("2"),
						"instances.0.id": validation.ToDiagFunc(validation.IsUUID),
						"instances.1.id": validation.ToDiagFunc(validation.IsUUID),
					}),
				),
			},
		},
	})
}

func dsCheckAttrs(ds string, expected testutils.TestAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return testutils.CheckResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("exoscale_instance_pool data source not found in the state")
	}
}
