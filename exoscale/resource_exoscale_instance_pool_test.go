// NOTE: remove build tag once 53111 is fixed
//go:build ignore
// +build ignore

package exoscale

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/ssgreg/repeat"
	"github.com/stretchr/testify/require"
)

var (
	testAccResourceInstancePoolAntiAffinityGroupName       = acctest.RandomWithPrefix(testPrefix)
	testAccResourceInstancePoolDescription                 = acctest.RandString(10)
	testAccResourceInstancePoolDescriptionUpdated          = testAccResourceInstancePoolDescription + "-updated"
	testAccResourceInstancePoolDiskSize              int64 = 10
	testAccResourceInstancePoolDiskSizeUpdated             = testAccResourceInstancePoolDiskSize * 2
	testAccResourceInstancePoolKeyPair                     = acctest.RandomWithPrefix(testPrefix)
	testAccResourceInstancePoolLabelValue                  = acctest.RandomWithPrefix(testPrefix)
	testAccResourceInstancePoolLabelValueUpdated           = testAccResourceInstancePoolLabelValue + "-updated"
	testAccResourceInstancePoolName                        = acctest.RandomWithPrefix(testPrefix)
	testAccResourceInstancePoolNameUpdated                 = testAccResourceInstancePoolName + "-updated"
	testAccResourceInstancePoolInstancePrefix              = "test"
	testAccResourceInstancePoolNetwork                     = acctest.RandomWithPrefix(testPrefix)
	testAccResourceInstancePoolInstanceType                = "standard.tiny"
	testAccResourceInstancePoolInstanceTypeUpdated         = "standard.small"
	testAccResourceInstancePoolSize                  int64 = 1
	testAccResourceInstancePoolSizeUpdated                 = testAccResourceInstancePoolSize * 2
	testAccResourceInstancePoolUserData                    = acctest.RandString(10)
	testAccResourceInstancePoolUserDataUpdated             = testAccResourceInstancePoolUserData + "-updated"

	testAccResourceInstancePoolConfigCreate = fmt.Sprintf(`
locals {
  zone = "%s"
}

data "exoscale_compute_template" "ubuntu" {
  zone = local.zone
  name = "Linux Ubuntu 20.04 LTS 64-bit"
}

data "exoscale_security_group" "default" {
  name = "default"
}

resource "exoscale_affinity" "test" {
  name = "%s"
}

resource "exoscale_instance_pool" "test" {
  zone = local.zone
  name = "%s"
  description = "%s"
  template_id = data.exoscale_compute_template.ubuntu.id
  instance_type = "%s"
  size = %d
  disk_size = %d
  ipv6 = true
  affinity_group_ids = [exoscale_affinity.test.id]
  security_group_ids = [data.exoscale_security_group.default.id]
  instance_prefix = "%s"
  user_data = "%s"
  labels = {
    test = "%s"
  }

  timeouts {
    delete = "10m"
  }
}
`,
		testZoneName,
		testAccResourceInstancePoolAntiAffinityGroupName,
		testAccResourceInstancePoolName,
		testAccResourceInstancePoolDescription,
		testAccResourceInstancePoolInstanceType,
		testAccResourceInstancePoolSize,
		testAccResourceInstancePoolDiskSize,
		testAccResourceInstancePoolInstancePrefix,
		testAccResourceInstancePoolUserData,
		testAccResourceInstancePoolLabelValue,
	)

	testAccResourceInstancePoolConfigUpdate = fmt.Sprintf(`
locals {
  zone = "%s"
}

data "exoscale_compute_template" "debian" {
  zone = local.zone
  name = "Linux Debian 10 (Buster) 64-bit"
}

resource "exoscale_network" "test" {
  zone = local.zone
  name = "%s"
}

resource "exoscale_ssh_keypair" "test" {
  name = "%s"
}

resource "exoscale_affinity" "test" {
  name = "%s"
}

resource "exoscale_ipaddress" "test" {
  zone = local.zone
}

resource "exoscale_instance_pool" "test" {
  zone = local.zone
  name = "%s"
  description = "%s"
  template_id = data.exoscale_compute_template.debian.id
  instance_type = "%s"
  size = %d
  disk_size = %d
  ipv6 = false
  key_pair = exoscale_ssh_keypair.test.name
  affinity_group_ids = [exoscale_affinity.test.id]
  network_ids = [exoscale_network.test.id]
  elastic_ip_ids = [exoscale_ipaddress.test.id]
  user_data = "%s"
  labels = {
    test = "%s"
  }

  timeouts {
    delete = "10m"
  }
}
`,
		testZoneName,
		testAccResourceInstancePoolNetwork,
		testAccResourceInstancePoolKeyPair,
		testAccResourceInstancePoolAntiAffinityGroupName,
		testAccResourceInstancePoolNameUpdated,
		testAccResourceInstancePoolDescriptionUpdated,
		testAccResourceInstancePoolInstanceTypeUpdated,
		testAccResourceInstancePoolSizeUpdated,
		testAccResourceInstancePoolDiskSizeUpdated,
		testAccResourceInstancePoolUserDataUpdated,
		testAccResourceInstancePoolLabelValueUpdated,
	)
)

func TestAccResourceInstancePool(t *testing.T) {
	var (
		r            = "exoscale_instance_pool.test"
		instancePool egoscale.InstancePool
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceInstancePoolDestroy(&instancePool),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceInstancePoolConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceInstancePoolExists(r, &instancePool),
					func(s *terraform.State) error {
						a := require.New(t)

						templateID, err := attrFromState(s, "data.exoscale_compute_template.ubuntu", "id")
						a.NoError(err, "unable to retrieve template ID from state")

						expectedUserData, _, err := encodeUserData(testAccResourceInstancePoolUserData)
						if err != nil {
							return err
						}

						a.Len(*instancePool.AntiAffinityGroupIDs, 1)
						a.Equal(testAccResourceInstancePoolDescription, *instancePool.Description)
						a.Equal(testAccResourceInstancePoolDiskSize, *instancePool.DiskSize)
						a.Equal(testAccResourceInstancePoolInstancePrefix, *instancePool.InstancePrefix)
						a.Len(*instancePool.InstanceIDs, int(testAccResourceInstancePoolSize))
						a.Equal(testInstanceTypeIDTiny, *instancePool.InstanceTypeID)
						a.True(*instancePool.IPv6Enabled)
						a.Equal(testAccResourceInstancePoolLabelValue, (*instancePool.Labels)["test"])
						a.Equal(testAccResourceInstancePoolName, *instancePool.Name)
						a.Equal(testAccResourceInstancePoolSize, *instancePool.Size)
						a.Equal(templateID, *instancePool.TemplateID)
						a.Equal(expectedUserData, *instancePool.UserData)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resInstancePoolAttrAffinityGroupIDs + ".#": validateString("1"),
						resInstancePoolAttrDescription:             validateString(testAccResourceInstancePoolDescription),
						resInstancePoolAttrDiskSize:                validateString(fmt.Sprint(testAccResourceInstancePoolDiskSize)),
						resInstancePoolAttrIPv6:                    validateString("true"),
						resInstancePoolAttrInstancePrefix:          validateString(testAccResourceInstancePoolInstancePrefix),
						resInstancePoolAttrInstanceType:            validateString(testAccResourceInstancePoolInstanceType),
						resInstancePoolAttrLabels + ".test":        validateString(testAccResourceInstancePoolLabelValue),
						resInstancePoolAttrName:                    validateString(testAccResourceInstancePoolName),
						resInstancePoolAttrSecurityGroupIDs + ".#": validateString("1"),
						resInstancePoolAttrSize:                    validateString(fmt.Sprint(testAccResourceInstancePoolSize)),
						resInstancePoolAttrState:                   validation.ToDiagFunc(validation.NoZeroValues),
						resInstancePoolAttrTemplateID:              validation.ToDiagFunc(validation.IsUUID),
						resInstancePoolAttrUserData:                validateString(testAccResourceInstancePoolUserData),
						resInstancePoolAttrVirtualMachines + ".#":  validateString(fmt.Sprint(testAccResourceInstancePoolSize)),
						resInstancePoolAttrInstances + ".#":        validateString(fmt.Sprint(testAccResourceInstancePoolSize)),
						resInstancePoolAttrZone:                    validateString(testZoneName),
					})),
				),
			},
			{
				// Update
				Config: testAccResourceInstancePoolConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceInstancePoolExists(r, &instancePool),
					func(s *terraform.State) error {
						a := require.New(t)

						templateID, err := attrFromState(s, "data.exoscale_compute_template.debian", "id")
						a.NoError(err, "unable to retrieve template ID from state")

						expectedUserData, _, err := encodeUserData(testAccResourceInstancePoolUserDataUpdated)
						if err != nil {
							return err
						}

						a.Len(*instancePool.AntiAffinityGroupIDs, 1)
						a.Equal(testAccResourceInstancePoolDescriptionUpdated, *instancePool.Description)
						a.Equal(testAccResourceInstancePoolDiskSizeUpdated, *instancePool.DiskSize)
						a.Equal(defaultInstancePoolInstancePrefix, *instancePool.InstancePrefix)
						a.Len(*instancePool.InstanceIDs, int(testAccResourceInstancePoolSizeUpdated))
						a.Equal(testInstanceTypeIDSmall, *instancePool.InstanceTypeID)
						a.False(*instancePool.IPv6Enabled)
						a.Equal(testAccResourceInstancePoolLabelValueUpdated, (*instancePool.Labels)["test"])
						a.Equal(testAccResourceInstancePoolNameUpdated, *instancePool.Name)
						a.Len(*instancePool.PrivateNetworkIDs, 1)
						a.Equal(testAccResourceInstancePoolSizeUpdated, *instancePool.Size)
						a.Equal(testAccResourceInstancePoolKeyPair, *instancePool.SSHKey)
						a.Equal(templateID, *instancePool.TemplateID)
						a.Equal(expectedUserData, *instancePool.UserData)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resInstancePoolAttrAffinityGroupIDs + ".#": validateString("1"),
						resInstancePoolAttrDescription:             validateString(testAccResourceInstancePoolDescriptionUpdated),
						resInstancePoolAttrDiskSize:                validateString(fmt.Sprint(testAccResourceInstancePoolDiskSizeUpdated)),
						resInstancePoolAttrElasticIPIDs + ".#":     validateString("1"),
						resInstancePoolAttrInstancePrefix:          validateString(defaultInstancePoolInstancePrefix),
						resInstancePoolAttrInstanceType:            validateString(testAccResourceInstancePoolInstanceTypeUpdated),
						resInstancePoolAttrIPv6:                    validateString("false"),
						resInstancePoolAttrKeyPair:                 validateString(testAccResourceInstancePoolKeyPair),
						resInstancePoolAttrLabels + ".test":        validateString(testAccResourceInstancePoolLabelValueUpdated),
						resInstancePoolAttrName:                    validateString(testAccResourceInstancePoolNameUpdated),
						resInstancePoolAttrNetworkIDs + ".#":       validateString("1"),
						resInstancePoolAttrSize:                    validateString(fmt.Sprint(testAccResourceInstancePoolSizeUpdated)),
						resInstancePoolAttrState:                   validation.ToDiagFunc(validation.NoZeroValues),
						resInstancePoolAttrUserData:                validateString(testAccResourceInstancePoolUserDataUpdated),
					})),
					resource.TestCheckNoResourceAttr(r, resInstancePoolAttrSecurityGroupIDs+".#"),
				),
			},
			{
				// Import
				ResourceName: r,
				ImportStateIdFunc: func(instancePool *egoscale.InstancePool) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", *instancePool.ID, testZoneName), nil
					}
				}(&instancePool),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resInstancePoolAttrAffinityGroupIDs + ".#": validateString("1"),
							resInstancePoolAttrDescription:             validateString(testAccResourceInstancePoolDescriptionUpdated),
							resInstancePoolAttrDiskSize:                validateString(fmt.Sprint(testAccResourceInstancePoolDiskSizeUpdated)),
							resInstancePoolAttrElasticIPIDs + ".#":     validateString("1"),
							resInstancePoolAttrInstancePrefix:          validateString(defaultInstancePoolInstancePrefix),
							resInstancePoolAttrInstanceType:            validateString(testAccResourceInstancePoolInstanceTypeUpdated),
							resInstancePoolAttrIPv6:                    validateString("false"),
							resInstancePoolAttrKeyPair:                 validateString(testAccResourceInstancePoolKeyPair),
							resInstancePoolAttrLabels + ".test":        validateString(testAccResourceInstancePoolLabelValueUpdated),
							resInstancePoolAttrName:                    validateString(testAccResourceInstancePoolNameUpdated),
							resInstancePoolAttrNetworkIDs + ".#":       validateString("1"),
							resInstancePoolAttrSize:                    validateString(fmt.Sprint(testAccResourceInstancePoolSizeUpdated)),
							resInstancePoolAttrState:                   validation.ToDiagFunc(validation.NoZeroValues),
							resInstancePoolAttrUserData:                validateString(testAccResourceInstancePoolUserDataUpdated),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceInstancePoolExists(r string, instancePool *egoscale.InstancePool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testZoneName),
		)

		res, err := client.Client.GetInstancePool(ctx, testZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*instancePool = *res
		return nil
	}
}

func testAccCheckResourceInstancePoolDestroy(instancePool *egoscale.InstancePool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testZoneName),
		)

		// The Exoscale API can be a bit slow to reflect the deletion operation
		// in the Instance Pool state, so we give it the benefit of the doubt
		// by retrying a few times before returning an error.
		return repeat.Repeat(
			repeat.Fn(func() error {
				instancePool, err := client.Client.GetInstancePool(ctx, testZoneName, *instancePool.ID)
				if err != nil {
					if errors.Is(err, exoapi.ErrNotFound) {
						return nil
					}
					return err
				}

				if *instancePool.State == "destroying" {
					return nil
				}

				return errors.New("Instance Pool still exists")
			}),
			repeat.StopOnSuccess(),
			repeat.LimitMaxTries(10),
			repeat.WithDelay(
				repeat.FixedBackoff(3*time.Second).Set(),
				repeat.SetContext(ctx),
			),
		)
	}
}
