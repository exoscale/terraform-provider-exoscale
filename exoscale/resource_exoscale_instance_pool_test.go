package exoscale

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	exov2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/ssgreg/repeat"
	"github.com/stretchr/testify/require"
)

var (
	testAccResourceInstancePoolAntiAffinityGroupName        = acctest.RandomWithPrefix(testPrefix)
	testAccResourceInstancePoolDescription                  = testDescription
	testAccResourceInstancePoolDiskSize               int64 = 10
	testAccResourceInstancePoolDiskSizeUpdated              = testAccResourceInstancePoolDiskSize * 2
	testAccResourceInstancePoolKeyPair                      = acctest.RandomWithPrefix(testPrefix)
	testAccResourceInstancePoolName                         = acctest.RandomWithPrefix(testPrefix)
	testAccResourceInstancePoolNameUpdated                  = testAccResourceInstancePoolName + "-updated"
	testAccResourceInstancePoolInstancePrefix               = "test"
	testAccResourceInstancePoolNetwork                      = acctest.RandomWithPrefix(testPrefix)
	testAccResourceInstancePoolServiceOffering              = "tiny"
	testAccResourceInstancePoolServiceOfferingUpdated       = "small"
	testAccResourceInstancePoolSize                   int64 = 1
	testAccResourceInstancePoolSizeUpdated                  = testAccResourceInstancePoolSize * 2
	testAccResourceInstancePoolZoneName                     = testZoneName
	testAccResourceInstancePoolUserData                     = acctest.RandString(10)
	testAccResourceInstancePoolUserDataUpdated              = testAccResourceInstancePoolUserData + "-updated"

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
  service_offering = "%s"
  size = %d
  disk_size = %d
  ipv6 = true
  affinity_group_ids = [exoscale_affinity.test.id]
  security_group_ids = [data.exoscale_security_group.default.id]
  instance_prefix = "%s"
  user_data = "%s"

  timeouts {
    delete = "10m"
  }
}
`,
		testAccResourceInstancePoolZoneName,
		testAccResourceInstancePoolAntiAffinityGroupName,
		testAccResourceInstancePoolName,
		testAccResourceInstancePoolDescription,
		testAccResourceInstancePoolServiceOffering,
		testAccResourceInstancePoolSize,
		testAccResourceInstancePoolDiskSize,
		testAccResourceInstancePoolInstancePrefix,
		testAccResourceInstancePoolUserData,
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
  template_id = data.exoscale_compute_template.debian.id
  service_offering = "%s"
  size = %d
  disk_size = %d
  key_pair = exoscale_ssh_keypair.test.name
  affinity_group_ids = [exoscale_affinity.test.id]
  network_ids = [exoscale_network.test.id]
  elastic_ip_ids = [exoscale_ipaddress.test.id]
  user_data = "%s"

  timeouts {
    delete = "10m"
  }
}
`,
		testAccResourceInstancePoolZoneName,
		testAccResourceInstancePoolNetwork,
		testAccResourceInstancePoolKeyPair,
		testAccResourceInstancePoolAntiAffinityGroupName,
		testAccResourceInstancePoolNameUpdated,
		testAccResourceInstancePoolServiceOfferingUpdated,
		testAccResourceInstancePoolSizeUpdated,
		testAccResourceInstancePoolDiskSizeUpdated,
		testAccResourceInstancePoolUserDataUpdated,
	)
)

func TestAccResourceInstancePool(t *testing.T) {
	var (
		r            = "exoscale_instance_pool.test"
		instancePool exov2.InstancePool
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

						expectedUserData, err := encodeUserData(testAccResourceInstancePoolUserData)
						if err != nil {
							return err
						}

						a.Len(instancePool.AntiAffinityGroupIDs, 1)
						a.Equal(testAccResourceInstancePoolDescription, instancePool.Description)
						a.Equal(testAccResourceInstancePoolDiskSize, instancePool.DiskSize)
						a.Equal(testAccResourceInstancePoolInstancePrefix, instancePool.InstancePrefix)
						a.Len(instancePool.InstanceIDs, int(testAccResourceInstancePoolSize))
						a.Equal(testInstanceTypeIDTiny, instancePool.InstanceTypeID)
						a.True(instancePool.IPv6Enabled)
						a.Equal(testAccResourceInstancePoolName, instancePool.Name)
						a.Equal(testAccResourceInstancePoolSize, instancePool.Size)
						a.Equal(templateID, instancePool.TemplateID)
						a.Equal(expectedUserData, instancePool.UserData)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resInstancePoolAttrAffinityGroupIDs + ".#": ValidateString("1"),
						resInstancePoolAttrDescription:             ValidateString(testAccResourceInstancePoolDescription),
						resInstancePoolAttrDiskSize:                ValidateString(fmt.Sprint(testAccResourceInstancePoolDiskSize)),
						resInstancePoolAttrID:                      validation.ToDiagFunc(validation.IsUUID),
						resInstancePoolAttrInstancePrefix:          ValidateString(testAccResourceInstancePoolInstancePrefix),
						resInstancePoolAttrIPv6:                    ValidateString("true"),
						resInstancePoolAttrName:                    ValidateString(testAccResourceInstancePoolName),
						resInstancePoolAttrSecurityGroupIDs + ".#": ValidateString("1"),
						resInstancePoolAttrServiceOffering:         ValidateString(testAccResourceInstancePoolServiceOffering),
						resInstancePoolAttrSize:                    ValidateString(fmt.Sprint(testAccResourceInstancePoolSize)),
						resInstancePoolAttrTemplateID:              validation.ToDiagFunc(validation.IsUUID),
						resInstancePoolAttrVirtualMachines + ".#":  ValidateString(fmt.Sprint(testAccResourceInstancePoolSize)),
						resInstancePoolAttrUserData:                ValidateString(testAccResourceInstancePoolUserData),
						resInstancePoolAttrZone:                    ValidateString(testAccResourceInstancePoolZoneName),
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

						expectedUserData, err := encodeUserData(testAccResourceInstancePoolUserDataUpdated)
						if err != nil {
							return err
						}

						a.Len(instancePool.AntiAffinityGroupIDs, 1)
						a.Empty(instancePool.Description)
						a.Equal(testAccResourceInstancePoolDiskSizeUpdated, instancePool.DiskSize)
						a.Equal(defaultInstancePoolInstancePrefix, instancePool.InstancePrefix)
						a.Len(instancePool.InstanceIDs, int(testAccResourceInstancePoolSizeUpdated))
						a.Equal(testInstanceTypeIDSmall, instancePool.InstanceTypeID)
						a.False(instancePool.IPv6Enabled)
						a.Equal(testAccResourceInstancePoolNameUpdated, instancePool.Name)
						a.Len(instancePool.PrivateNetworkIDs, 1)
						a.Equal(testAccResourceInstancePoolSizeUpdated, instancePool.Size)
						a.Equal(testAccResourceInstancePoolKeyPair, instancePool.SSHKey)
						a.Equal(templateID, instancePool.TemplateID)
						a.Equal(expectedUserData, instancePool.UserData)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resInstancePoolAttrAffinityGroupIDs + ".#": ValidateString("1"),
						resInstancePoolAttrDescription:             ValidateString(""),
						resInstancePoolAttrDiskSize:                ValidateString(fmt.Sprint(testAccResourceInstancePoolDiskSizeUpdated)),
						resInstancePoolAttrElasticIPIDs + ".#":     ValidateString("1"),
						resInstancePoolAttrID:                      validation.ToDiagFunc(validation.IsUUID),
						resInstancePoolAttrInstancePrefix:          ValidateString(defaultInstancePoolInstancePrefix),
						resInstancePoolAttrIPv6:                    ValidateString("false"),
						resInstancePoolAttrKeyPair:                 ValidateString(testAccResourceInstancePoolKeyPair),
						resInstancePoolAttrName:                    ValidateString(testAccResourceInstancePoolNameUpdated),
						resInstancePoolAttrNetworkIDs + ".#":       ValidateString("1"),
						resInstancePoolAttrServiceOffering:         ValidateString(testAccResourceInstancePoolServiceOfferingUpdated),
						resInstancePoolAttrSize:                    ValidateString(fmt.Sprint(testAccResourceInstancePoolSizeUpdated)),
						resInstancePoolAttrUserData:                ValidateString(testAccResourceInstancePoolUserDataUpdated),
					})),
					resource.TestCheckNoResourceAttr(r, resInstancePoolAttrSecurityGroupIDs+".#"),
				),
			},
			{
				// Import
				ResourceName:            r,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"state"},
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resInstancePoolAttrAffinityGroupIDs + ".#": ValidateString("1"),
							resInstancePoolAttrDescription:             ValidateString(""),
							resInstancePoolAttrDiskSize:                ValidateString(fmt.Sprint(testAccResourceInstancePoolDiskSizeUpdated)),
							resInstancePoolAttrElasticIPIDs + ".#":     ValidateString("1"),
							resInstancePoolAttrID:                      validation.ToDiagFunc(validation.IsUUID),
							resInstancePoolAttrInstancePrefix:          ValidateString(defaultInstancePoolInstancePrefix),
							resInstancePoolAttrIPv6:                    ValidateString("false"),
							resInstancePoolAttrKeyPair:                 ValidateString(testAccResourceInstancePoolKeyPair),
							resInstancePoolAttrName:                    ValidateString(testAccResourceInstancePoolNameUpdated),
							resInstancePoolAttrNetworkIDs + ".#":       ValidateString("1"),
							resInstancePoolAttrServiceOffering:         ValidateString(testAccResourceInstancePoolServiceOfferingUpdated),
							resInstancePoolAttrSize:                    ValidateString(fmt.Sprint(testAccResourceInstancePoolSizeUpdated)),
							resInstancePoolAttrUserData:                ValidateString(testAccResourceInstancePoolUserDataUpdated),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceInstancePoolExists(r string, instancePool *exov2.InstancePool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client := GetComputeClient(testAccProvider.Meta())

		res, err := client.Client.GetInstancePool(
			context.Background(),
			testAccResourceInstancePoolZoneName,
			rs.Primary.ID)
		if err != nil {
			return err
		}

		*instancePool = *res
		return nil
	}
}

func testAccCheckResourceInstancePoolDestroy(instancePool *exov2.InstancePool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		ctx := context.Background()
		client := GetComputeClient(testAccProvider.Meta())

		// The Exoscale API can be a bit slow to reflect the deletion operation
		// in the Instance Pool state, so we give it the benefit of the doubt
		// by retrying a few times before returning an error.
		return repeat.Repeat(
			repeat.Fn(func() error {
				instancePool, err := client.Client.GetInstancePool(
					ctx,
					testAccResourceInstancePoolZoneName,
					instancePool.ID,
				)
				if err != nil {
					if errors.Is(err, exoapi.ErrNotFound) {
						return nil
					}
					return err
				}

				if instancePool.State == "destroying" {
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
