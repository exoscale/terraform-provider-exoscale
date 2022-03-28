package exoscale

import (
	"context"
	"errors"
	"fmt"
	"testing"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

var (
	testAccResourceComputeInstanceAntiAffinityGroupName       = acctest.RandomWithPrefix(testPrefix)
	testAccResourceComputeInstanceDiskSize              int64 = 10
	testAccResourceComputeInstanceDiskSizeUpdated             = testAccResourceComputeInstanceDiskSize * 2
	testAccResourceComputeInstanceLabelValue                  = acctest.RandomWithPrefix(testPrefix)
	testAccResourceComputeInstanceLabelValueUpdated           = testAccResourceComputeInstanceLabelValue + "-updated"
	testAccResourceComputeInstanceName                        = acctest.RandomWithPrefix(testPrefix)
	testAccResourceComputeInstanceNameUpdated                 = testAccResourceComputeInstanceName + "-updated"
	testAccResourceComputeInstancePrivateNetworkName          = acctest.RandomWithPrefix(testPrefix)
	testAccResourceComputeInstanceSSHKeyName                  = acctest.RandomWithPrefix(testPrefix)
	testAccResourceComputeInstanceSecurityGroupName           = acctest.RandomWithPrefix(testPrefix)
	testAccResourceComputeInstanceState                       = "stopped"
	testAccResourceComputeInstanceType                        = "standard.tiny"
	testAccResourceComputeInstanceTypeUpdated                 = "standard.small"
	testAccResourceComputeInstanceUserData                    = acctest.RandString(10)
	testAccResourceComputeInstanceUserDataUpdated             = testAccResourceComputeInstanceUserData + "-updated"

	testAccResourceComputeInstanceConfigCreate = fmt.Sprintf(`
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
  security_group_ids      = [
    data.exoscale_security_group.default.id,
    exoscale_security_group.test.id,
  ]
  elastic_ip_ids          = [exoscale_elastic_ip.test.id]
  user_data               = "%s"
  ssh_key                 = exoscale_ssh_key.test.name
  state                   = "%s"

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
		testAccResourceComputeInstanceSecurityGroupName,
		testAccResourceComputeInstanceAntiAffinityGroupName,
		testAccResourceComputeInstancePrivateNetworkName,
		testAccResourceComputeInstanceSSHKeyName,
		testAccResourceComputeInstanceName,
		testAccResourceComputeInstanceType,
		testAccResourceComputeInstanceDiskSize,
		testAccResourceComputeInstanceUserData,
		testAccResourceComputeInstanceState,
		testAccResourceComputeInstanceLabelValue,
	)

	testAccResourceComputeInstanceConfigUpdate = fmt.Sprintf(`
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
  security_group_ids      = [data.exoscale_security_group.default.id]
  elastic_ip_ids          = []
  user_data               = "%s"
  ssh_key                 = exoscale_ssh_key.test.name
  state                   = "%s"

  labels = {
    test = "%s"
  }

  timeouts {
    delete = "10m"
  }
}
`,
		testZoneName,
		testAccResourceComputeInstanceSecurityGroupName,
		testAccResourceComputeInstanceAntiAffinityGroupName,
		testAccResourceComputeInstancePrivateNetworkName,
		testAccResourceComputeInstanceSSHKeyName,
		testAccResourceComputeInstanceNameUpdated,
		testAccResourceComputeInstanceTypeUpdated,
		testAccResourceComputeInstanceDiskSizeUpdated,
		testAccResourceComputeInstanceUserDataUpdated,
		testAccResourceComputeInstanceState,
		testAccResourceComputeInstanceLabelValueUpdated,
	)
)

func TestAccResourceComputeInstance(t *testing.T) {
	var (
		r                     = "exoscale_compute_instance.test"
		computeInstance       egoscale.Instance
		testAntiAffinityGroup egoscale.AntiAffinityGroup
		testPrivateNetwork    egoscale.PrivateNetwork
		testSecurityGroup     egoscale.SecurityGroup
		testElasticIP         egoscale.ElasticIP
		testSSHKey            egoscale.SSHKey
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceComputeInstanceDestroy(&computeInstance),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceComputeInstanceConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSecurityGroupExists("exoscale_security_group.test", &testSecurityGroup),
					testAccCheckResourceAntiAffinityGroupExists("exoscale_anti_affinity_group.test", &testAntiAffinityGroup),
					testAccCheckResourcePrivateNetworkExists("exoscale_private_network.test", &testPrivateNetwork),
					testAccCheckResourceElasticIPExists("exoscale_elastic_ip.test", &testElasticIP),
					testAccCheckResourceSSHKeyExists("exoscale_ssh_key.test", &testSSHKey),
					testAccCheckResourceComputeInstanceExists(r, &computeInstance),
					func(s *terraform.State) error {
						a := require.New(t)

						templateID, err := attrFromState(s, "data.exoscale_compute_template.ubuntu", "id")
						a.NoError(err, "unable to retrieve template ID from state")

						defaultSecurityGroupID, err := attrFromState(s, "data.exoscale_security_group.default", "id")
						a.NoError(err, "unable to retrieve default Security Group ID from state")

						expectedUserData, err := encodeUserData(testAccResourceComputeInstanceUserData)
						if err != nil {
							return err
						}

						a.NotNil(computeInstance.AntiAffinityGroupIDs)
						a.ElementsMatch([]string{*testAntiAffinityGroup.ID}, *computeInstance.AntiAffinityGroupIDs)
						a.Equal(testAccResourceComputeInstanceDiskSize, *computeInstance.DiskSize)
						a.NotNil(computeInstance.ElasticIPIDs)
						a.ElementsMatch([]string{*testElasticIP.ID}, *computeInstance.ElasticIPIDs)
						a.Equal(testInstanceTypeIDTiny, *computeInstance.InstanceTypeID)
						a.True(*computeInstance.IPv6Enabled)
						a.Equal(testAccResourceComputeInstanceLabelValue, (*computeInstance.Labels)["test"])
						a.Equal(testAccResourceComputeInstanceName, *computeInstance.Name)
						a.NotNil(computeInstance.PrivateNetworkIDs)
						a.ElementsMatch([]string{*testPrivateNetwork.ID}, *computeInstance.PrivateNetworkIDs)
						a.Equal(*testSSHKey.Name, *computeInstance.SSHKey)
						a.NotNil(computeInstance.SecurityGroupIDs)
						a.ElementsMatch([]string{defaultSecurityGroupID, *testSecurityGroup.ID}, *computeInstance.SecurityGroupIDs)
						a.Equal(testAccResourceComputeInstanceState, *computeInstance.State)
						a.Equal(templateID, *computeInstance.TemplateID)
						a.Equal(expectedUserData, *computeInstance.UserData)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resComputeInstanceAttrAntiAffinityGroupIDs + ".#": validateString("1"),
						resComputeInstanceAttrCreatedAt:                   validation.ToDiagFunc(validation.NoZeroValues),
						resComputeInstanceAttrDiskSize:                    validateString(fmt.Sprint(testAccResourceComputeInstanceDiskSize)),
						resComputeInstanceAttrElasticIPIDs + ".#":         validateString("1"),
						resComputeInstanceAttrIPv6:                        validateString("true"),
						resComputeInstanceAttrIPv6Address:                 validation.ToDiagFunc(validation.IsIPv6Address),
						resComputeInstanceAttrLabels + ".test":            validateString(testAccResourceComputeInstanceLabelValue),
						resComputeInstanceAttrName:                        validateString(testAccResourceComputeInstanceName),
						resComputeInstanceAttrNetworkInterface + ".#":     validateString("1"),
						resComputeInstanceAttrPublicIPAddress:             validation.ToDiagFunc(validation.IsIPv4Address),
						resComputeInstanceAttrSSHKey:                      validateString(testAccResourceComputeInstanceSSHKeyName),
						resComputeInstanceAttrSecurityGroupIDs + ".#":     validateString("2"),
						resComputeInstanceAttrState:                       validateString("stopped"),
						resComputeInstanceAttrTemplateID:                  validation.ToDiagFunc(validation.IsUUID),
						resComputeInstanceAttrType:                        validateString(testAccResourceComputeInstanceType),
						resComputeInstanceAttrUserData:                    validateString(testAccResourceComputeInstanceUserData),
						resComputeInstanceAttrZone:                        validateString(testZoneName),
					})),
				),
			},
			{
				// Update
				Config: testAccResourceComputeInstanceConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceComputeInstanceExists(r, &computeInstance),
					func(s *terraform.State) error {
						a := require.New(t)

						defaultSecurityGroupID, err := attrFromState(s, "data.exoscale_security_group.default", "id")
						a.NoError(err, "unable to retrieve default Security Group ID from state")

						expectedUserData, err := encodeUserData(testAccResourceComputeInstanceUserDataUpdated)
						if err != nil {
							return err
						}

						a.NotNil(computeInstance.AntiAffinityGroupIDs)
						a.ElementsMatch([]string{*testAntiAffinityGroup.ID}, *computeInstance.AntiAffinityGroupIDs)
						a.Equal(testAccResourceComputeInstanceDiskSizeUpdated, *computeInstance.DiskSize)
						a.Nil(computeInstance.ElasticIPIDs)
						a.Equal(testInstanceTypeIDSmall, *computeInstance.InstanceTypeID)
						a.Equal(testAccResourceComputeInstanceLabelValueUpdated, (*computeInstance.Labels)["test"])
						a.Equal(testAccResourceComputeInstanceNameUpdated, *computeInstance.Name)
						a.Nil(computeInstance.PrivateNetworkIDs)
						a.NotNil(computeInstance.SecurityGroupIDs)
						a.ElementsMatch([]string{defaultSecurityGroupID}, *computeInstance.SecurityGroupIDs)
						a.Equal(testAccResourceComputeInstanceState, *computeInstance.State)
						a.Equal(expectedUserData, *computeInstance.UserData)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resComputeInstanceAttrDiskSize:                validateString(fmt.Sprint(testAccResourceComputeInstanceDiskSizeUpdated)),
						resComputeInstanceAttrLabels + ".test":        validateString(testAccResourceComputeInstanceLabelValueUpdated),
						resComputeInstanceAttrName:                    validateString(testAccResourceComputeInstanceNameUpdated),
						resComputeInstanceAttrSecurityGroupIDs + ".#": validateString("1"),
						resComputeInstanceAttrState:                   validateString("stopped"),
						resComputeInstanceAttrType:                    validateString(testAccResourceComputeInstanceTypeUpdated),
						resComputeInstanceAttrUserData:                validateString(testAccResourceComputeInstanceUserDataUpdated),
					})),
					resource.TestCheckNoResourceAttr(r, resComputeInstanceAttrElasticIPIDs+".#"),
					resource.TestCheckNoResourceAttr(r, resComputeInstanceAttrNetworkInterface+".#"),
				),
			},
			{
				// Import
				ResourceName: r,
				ImportStateIdFunc: func(computeInstance *egoscale.Instance) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", *computeInstance.ID, testZoneName), nil
					}
				}(&computeInstance),
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{resComputeInstanceAttrPrivateNetworkIDs},
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resComputeInstanceAttrDiskSize:                validateString(fmt.Sprint(testAccResourceComputeInstanceDiskSizeUpdated)),
							resComputeInstanceAttrLabels + ".test":        validateString(testAccResourceComputeInstanceLabelValueUpdated),
							resComputeInstanceAttrName:                    validateString(testAccResourceComputeInstanceNameUpdated),
							resComputeInstanceAttrSecurityGroupIDs + ".#": validateString("1"),
							resComputeInstanceAttrState:                   validateString("stopped"),
							resComputeInstanceAttrType:                    validateString(testAccResourceComputeInstanceTypeUpdated),
							resComputeInstanceAttrUserData:                validateString(testAccResourceComputeInstanceUserDataUpdated),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceComputeInstanceExists(r string, computeInstance *egoscale.Instance) resource.TestCheckFunc {
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

		res, err := client.Client.GetInstance(ctx, testZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*computeInstance = *res
		return nil
	}
}

func testAccCheckResourceComputeInstanceDestroy(computeInstance *egoscale.Instance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testEnvironment, testZoneName))

		_, err := client.GetInstance(ctx, testZoneName, *computeInstance.ID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("Compute instance still exists")
	}
}
