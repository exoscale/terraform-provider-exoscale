package instance_pool_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"

	egoscale "github.com/exoscale/egoscale/v2"

	"github.com/exoscale/terraform-provider-exoscale/pkg/resources/instance_pool"
	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

var (
	rAntiAffinityGroupName       = acctest.RandomWithPrefix(testutils.Prefix)
	rDescription                 = acctest.RandString(10)
	rDescriptionUpdated          = rDescription + "-updated"
	rDiskSize              int64 = 10
	rDiskSizeUpdated             = rDiskSize * 2
	rKeyPair                     = acctest.RandomWithPrefix(testutils.Prefix)
	rLabelValue                  = acctest.RandomWithPrefix(testutils.Prefix)
	rLabelValueUpdated           = rLabelValue + "-updated"
	rName                        = acctest.RandomWithPrefix(testutils.Prefix)
	rNameUpdated                 = rName + "-updated"
	rInstancePrefix              = "test"
	rNetwork                     = acctest.RandomWithPrefix(testutils.Prefix)
	rInstanceType                = "standard.tiny"
	rInstanceTypeUpdated         = "standard.small"
	rSize                  int64 = 1
	rSizeUpdated                 = rSize * 2
	rUserData                    = acctest.RandString(10)
	rUserDataUpdated             = rUserData + "-updated"

	rConfigCreate = fmt.Sprintf(`
locals {
  zone = "%s"
}

data "exoscale_template" "ubuntu" {
  zone = local.zone
  name = "Linux Ubuntu 20.04 LTS 64-bit"
}

data "exoscale_security_group" "default" {
  name = "default"
}

resource "exoscale_anti_affinity_group" "test" {
  name = "%s"
}

resource "exoscale_instance_pool" "test" {
  zone = local.zone
  name = "%s"
  description = "%s"
  template_id = data.exoscale_template.ubuntu.id
  instance_type = "%s"
  size = %d
  disk_size = %d
  ipv6 = true
  anti_affinity_group_ids = [exoscale_anti_affinity_group.test.id]
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
		testutils.TestZoneName,
		rAntiAffinityGroupName,
		rName,
		rDescription,
		rInstanceType,
		rSize,
		rDiskSize,
		rInstancePrefix,
		rUserData,
		rLabelValue,
	)

	rConfigUpdate = fmt.Sprintf(`
locals {
  zone = "%s"
}

data "exoscale_template" "debian" {
  zone = local.zone
  name = "Linux Debian 12 (Bookworm) 64-bit"
}

resource "exoscale_private_network" "test" {
  zone = local.zone
  name = "%s"
}

resource "exoscale_ssh_key" "test" {
  name = "%s"
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIB8bfA67mQWv4eGND/XVtPx1JW6RAqafub1lV1EcpB+b test"
}

resource "exoscale_anti_affinity_group" "test" {
  name = "%s"
}

resource "exoscale_instance_pool" "test" {
  zone = local.zone
  name = "%s"
  description = "%s"
  template_id = data.exoscale_template.debian.id
  instance_type = "%s"
  size = %d
  disk_size = %d
  ipv6 = false
  key_pair = exoscale_ssh_key.test.name
  anti_affinity_group_ids = [exoscale_anti_affinity_group.test.id]
  network_ids = [exoscale_private_network.test.id]
  user_data = "%s"
  labels = {
    test = "%s"
  }

  timeouts {
    delete = "10m"
  }
}
`,
		testutils.TestZoneName,
		rNetwork,
		rKeyPair,
		rAntiAffinityGroupName,
		rNameUpdated,
		rDescriptionUpdated,
		rInstanceTypeUpdated,
		rSizeUpdated,
		rDiskSizeUpdated,
		rUserDataUpdated,
		rLabelValueUpdated,
	)
)

func testResource(t *testing.T) {
	var (
		r            = "exoscale_instance_pool.test"
		instancePool egoscale.InstancePool
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testutils.AccPreCheck(t) },
		ProviderFactories: testutils.Providers(),
		CheckDestroy:      testutils.CheckInstancePoolDestroy(&instancePool),
		Steps: []resource.TestStep{
			{
				// Create
				Config: rConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testutils.CheckInstancePoolExists(r, &instancePool),
					func(s *terraform.State) error {
						a := require.New(t)

						templateID, err := testutils.AttrFromState(s, "data.exoscale_template.ubuntu", "id")
						a.NoError(err, "unable to retrieve template ID from state")

						expectedUserData, _, err := utils.EncodeUserData(rUserData)
						if err != nil {
							return err
						}

						a.Len(*instancePool.AntiAffinityGroupIDs, 1)
						a.Equal(rDescription, *instancePool.Description)
						a.Equal(rDiskSize, *instancePool.DiskSize)
						a.Equal(rInstancePrefix, *instancePool.InstancePrefix)
						a.Len(*instancePool.InstanceIDs, int(rSize))
						a.Equal(testutils.TestInstanceTypeIDTiny, *instancePool.InstanceTypeID)
						a.True(*instancePool.IPv6Enabled)
						a.Equal(rLabelValue, (*instancePool.Labels)["test"])
						a.Equal(rName, *instancePool.Name)
						a.Equal(rSize, *instancePool.Size)
						a.Equal(templateID, *instancePool.TemplateID)
						a.Equal(expectedUserData, *instancePool.UserData)

						return nil
					},
					testutils.CheckResourceState(r, testutils.CheckResourceStateValidateAttributes(testutils.TestAttrs{
						instance_pool.AttrAntiAffinityGroupIDs + ".#": testutils.ValidateString("1"),
						instance_pool.AttrDescription:                 testutils.ValidateString(rDescription),
						instance_pool.AttrDiskSize:                    testutils.ValidateString(fmt.Sprint(rDiskSize)),
						instance_pool.AttrIPv6:                        testutils.ValidateString("true"),
						instance_pool.AttrInstancePrefix:              testutils.ValidateString(rInstancePrefix),
						instance_pool.AttrInstanceType:                testutils.ValidateString(rInstanceType),
						instance_pool.AttrLabels + ".test":            testutils.ValidateString(rLabelValue),
						instance_pool.AttrName:                        testutils.ValidateString(rName),
						instance_pool.AttrSecurityGroupIDs + ".#":     testutils.ValidateString("1"),
						instance_pool.AttrSize:                        testutils.ValidateString(fmt.Sprint(rSize)),
						instance_pool.AttrState:                       validation.ToDiagFunc(validation.NoZeroValues),
						instance_pool.AttrTemplateID:                  validation.ToDiagFunc(validation.IsUUID),
						instance_pool.AttrUserData:                    testutils.ValidateString(rUserData),
						instance_pool.AttrVirtualMachines + ".#":      testutils.ValidateString(fmt.Sprint(rSize)),
						instance_pool.AttrInstances + ".#":            testutils.ValidateString(fmt.Sprint(rSize)),
						instance_pool.AttrZone:                        testutils.ValidateString(testutils.TestZoneName),
					})),
				),
			},
			{
				// Update
				Config: rConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testutils.CheckInstancePoolExists(r, &instancePool),
					func(s *terraform.State) error {
						a := require.New(t)

						templateID, err := testutils.AttrFromState(s, "data.exoscale_template.debian", "id")
						a.NoError(err, "unable to retrieve template ID from state")

						expectedUserData, _, err := utils.EncodeUserData(rUserDataUpdated)
						if err != nil {
							return err
						}

						a.Len(*instancePool.AntiAffinityGroupIDs, 1)
						a.Equal(rDescriptionUpdated, *instancePool.Description)
						a.Equal(rDiskSizeUpdated, *instancePool.DiskSize)
						a.Equal(instance_pool.DefaultInstancePrefix, *instancePool.InstancePrefix)
						a.Len(*instancePool.InstanceIDs, int(rSizeUpdated))
						a.Equal(testutils.TestInstanceTypeIDSmall, *instancePool.InstanceTypeID)
						a.False(*instancePool.IPv6Enabled)
						a.Equal(rLabelValueUpdated, (*instancePool.Labels)["test"])
						a.Equal(rNameUpdated, *instancePool.Name)
						a.Len(*instancePool.PrivateNetworkIDs, 1)
						a.Equal(rSizeUpdated, *instancePool.Size)
						a.Equal(rKeyPair, *instancePool.SSHKey)
						a.Equal(templateID, *instancePool.TemplateID)
						a.Equal(expectedUserData, *instancePool.UserData)

						return nil
					},
					testutils.CheckResourceState(r, testutils.CheckResourceStateValidateAttributes(testutils.TestAttrs{
						instance_pool.AttrAntiAffinityGroupIDs + ".#": testutils.ValidateString("1"),
						instance_pool.AttrDescription:                 testutils.ValidateString(rDescriptionUpdated),
						instance_pool.AttrDiskSize:                    testutils.ValidateString(fmt.Sprint(rDiskSizeUpdated)),
						instance_pool.AttrInstancePrefix:              testutils.ValidateString(instance_pool.DefaultInstancePrefix),
						instance_pool.AttrInstanceType:                testutils.ValidateString(rInstanceTypeUpdated),
						instance_pool.AttrIPv6:                        testutils.ValidateString("false"),
						instance_pool.AttrKeyPair:                     testutils.ValidateString(rKeyPair),
						instance_pool.AttrLabels + ".test":            testutils.ValidateString(rLabelValueUpdated),
						instance_pool.AttrName:                        testutils.ValidateString(rNameUpdated),
						instance_pool.AttrNetworkIDs + ".#":           testutils.ValidateString("1"),
						instance_pool.AttrSize:                        testutils.ValidateString(fmt.Sprint(rSizeUpdated)),
						instance_pool.AttrState:                       validation.ToDiagFunc(validation.NoZeroValues),
						instance_pool.AttrUserData:                    testutils.ValidateString(rUserDataUpdated),
					})),
					resource.TestCheckNoResourceAttr(r, instance_pool.AttrSecurityGroupIDs+".#"),
				),
			},
			{
				// Import
				ResourceName: r,
				ImportStateIdFunc: func(instancePool *egoscale.InstancePool) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", *instancePool.ID, testutils.TestZoneName), nil
					}
				}(&instancePool),
				ImportState: true,
				// We will not verify state as we are unable to set AAG while both AffinityGroup & AntiAffinityGroup exist.
				// Once AffinityGroup is completely removed we can reenable this check.
				// ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return testutils.CheckResourceAttributes(
						testutils.TestAttrs{
							// AAG is unset because SDK provides no way to determine if AffinityGroup or AntiAffinityGroup
							// are set in config during import.
							// Once AffinityGroup is completely removed we can reenable this check.
							// instance_pool.AttrAntiAffinityGroupIDs + ".#": testutils.ValidateString("1"),
							instance_pool.AttrDescription:       testutils.ValidateString(rDescriptionUpdated),
							instance_pool.AttrDiskSize:          testutils.ValidateString(fmt.Sprint(rDiskSizeUpdated)),
							instance_pool.AttrInstancePrefix:    testutils.ValidateString(instance_pool.DefaultInstancePrefix),
							instance_pool.AttrInstanceType:      testutils.ValidateString(rInstanceTypeUpdated),
							instance_pool.AttrIPv6:              testutils.ValidateString("false"),
							instance_pool.AttrKeyPair:           testutils.ValidateString(rKeyPair),
							instance_pool.AttrLabels + ".test":  testutils.ValidateString(rLabelValueUpdated),
							instance_pool.AttrName:              testutils.ValidateString(rNameUpdated),
							instance_pool.AttrNetworkIDs + ".#": testutils.ValidateString("1"),
							instance_pool.AttrSize:              testutils.ValidateString(fmt.Sprint(rSizeUpdated)),
							instance_pool.AttrState:             validation.ToDiagFunc(validation.NoZeroValues),
							instance_pool.AttrUserData:          testutils.ValidateString(rUserDataUpdated),
						},
						func(s []*terraform.InstanceState) map[string]string {
							for _, state := range s {
								if state.ID == *instancePool.ID {
									return state.Attributes
								}
							}
							return nil
						}(s),
					)
				},
			},
		},
	})
}
