package instance_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"

	egoscale "github.com/exoscale/egoscale/v2"

	"github.com/exoscale/terraform-provider-exoscale/pkg/resources/instance"
	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

var (
	rAntiAffinityGroupName       = acctest.RandomWithPrefix(testutils.Prefix)
	rDiskSize              int64 = 10
	rDiskSizeUpdated             = rDiskSize * 2
	rDiskSizeUpdated2            = rDiskSize * 3
	rLabelValue                  = acctest.RandomWithPrefix(testutils.Prefix)
	rLabelValueUpdated           = rLabelValue + "-updated"
	rName                        = acctest.RandomWithPrefix(testutils.Prefix)
	rNameUpdated                 = rName + "-updated"
	rPrivateNetworkName          = acctest.RandomWithPrefix(testutils.Prefix)
	rSSHKeyName                  = acctest.RandomWithPrefix(testutils.Prefix)
	rSecurityGroupName           = acctest.RandomWithPrefix(testutils.Prefix)
	rStateStopped                = "stopped"
	rStateRunning                = "running"
	rType                        = "standard.tiny"
	rTypeUpdated                 = "standard.small"
	rReverseDNS                  = "tf-provider-test.exoscale.com"
	rReverseDNSUpdated           = "tf-provider-updated-test.exoscale.com"
	rUserData                    = acctest.RandString(10)
	rUserDataUpdated             = rUserData + "-updated"

	rConfigCreateStopped = fmt.Sprintf(`
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
  template_id             = data.exoscale_template.ubuntu.id
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
	reverse_dns             = "%s"

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
		testutils.TestZoneName,
		rSecurityGroupName,
		rAntiAffinityGroupName,
		rPrivateNetworkName,
		rSSHKeyName,
		rName,
		rType,
		rDiskSize,
		rUserData,
		rStateStopped,
		rReverseDNS,
		rLabelValue,
	)

	rConfigUpdateStopped = fmt.Sprintf(`
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
  template_id             = data.exoscale_template.ubuntu.id
  ipv6                    = true
  anti_affinity_group_ids = [exoscale_anti_affinity_group.test.id]
  security_group_ids      = [data.exoscale_security_group.default.id]
  elastic_ip_ids          = []
  user_data               = "%s"
  ssh_key                 = exoscale_ssh_key.test.name
	state                   = "%s"
  reverse_dns             = "%s"

  labels = {
    test = "%s"
  }

  timeouts {
    delete = "10m"
  }
}
`,
		testutils.TestZoneName,
		rSecurityGroupName,
		rAntiAffinityGroupName,
		rPrivateNetworkName,
		rSSHKeyName,
		rNameUpdated,
		rTypeUpdated,
		rDiskSizeUpdated,
		rUserDataUpdated,
		rStateStopped,
		rReverseDNSUpdated,
		rLabelValueUpdated,
	)

	rConfigStart = fmt.Sprintf(`
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
  template_id             = data.exoscale_template.ubuntu.id
  ipv6                    = true
  anti_affinity_group_ids = [exoscale_anti_affinity_group.test.id]
  security_group_ids      = [data.exoscale_security_group.default.id]
  elastic_ip_ids          = []
  user_data               = "%s"
  ssh_key                 = exoscale_ssh_key.test.name
	state                   = "%s"
  reverse_dns             = ""

  labels = {
    test = "%s"
  }

  timeouts {
    delete = "10m"
  }
}
`,
		testutils.TestZoneName,
		rSecurityGroupName,
		rAntiAffinityGroupName,
		rPrivateNetworkName,
		rSSHKeyName,
		rNameUpdated,
		rTypeUpdated,
		rDiskSizeUpdated,
		rUserDataUpdated,
		rStateRunning,
		rLabelValueUpdated,
	)

	rConfigUpdateStarted = fmt.Sprintf(`
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
  template_id             = data.exoscale_template.ubuntu.id
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
		testutils.TestZoneName,
		rSecurityGroupName,
		rAntiAffinityGroupName,
		rPrivateNetworkName,
		rSSHKeyName,
		rNameUpdated,
		rTypeUpdated,
		rDiskSizeUpdated2,
		rUserDataUpdated,
		rStateRunning,
		rLabelValueUpdated,
	)

	rConfigCreateManaged = fmt.Sprintf(`
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

resource "exoscale_private_network" "test" {
  zone = local.zone
  name = "%s"
	netmask  = "255.255.255.0"
  start_ip = "10.0.0.50"
  end_ip   = "10.0.0.250"
}

resource "exoscale_compute_instance" "test" {
  zone                    = local.zone
  name                    = "%s"
  type                    = "%s"
  disk_size               = %d
  template_id             = data.exoscale_template.ubuntu.id
  security_group_ids      = [data.exoscale_security_group.default.id]

  network_interface {
	  network_id = exoscale_private_network.test.id
		ip_address = "10.0.0.100"
  }

  timeouts {
    delete = "10m"
  }
}
`,
		testutils.TestZoneName,
		rPrivateNetworkName,
		rName,
		rType,
		rDiskSize,
	)
)

func testResource(t *testing.T) {
	var (
		r                     = "exoscale_compute_instance.test"
		testInstance          egoscale.Instance
		testAntiAffinityGroup egoscale.AntiAffinityGroup
		testPrivateNetwork    egoscale.PrivateNetwork
		testSecurityGroup     egoscale.SecurityGroup
		testElasticIP         egoscale.ElasticIP
		testSSHKey            egoscale.SSHKey
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testutils.AccPreCheck(t) },
		ProviderFactories: testutils.Providers(),
		CheckDestroy:      testutils.CheckInstanceDestroy(&testInstance),
		Steps: []resource.TestStep{
			{
				// Create stopped testInstance
				Config: rConfigCreateStopped,
				Check: resource.ComposeTestCheckFunc(
					testutils.CheckSecurityGroupExists("exoscale_security_group.test", &testSecurityGroup),
					testutils.CheckAntiAffinityGroupExists("exoscale_anti_affinity_group.test", &testAntiAffinityGroup),
					testutils.CheckPrivateNetworkExists("exoscale_private_network.test", &testPrivateNetwork),
					testutils.CheckElasticIPExists("exoscale_elastic_ip.test", &testElasticIP),
					testutils.CheckSSHKeyExists("exoscale_ssh_key.test", &testSSHKey),
					testutils.CheckInstanceExists(r, &testInstance),
					func(s *terraform.State) error {
						a := require.New(t)

						templateID, err := testutils.AttrFromState(s, "data.exoscale_template.ubuntu", "id")
						a.NoError(err, "unable to retrieve template ID from state")

						defaultSecurityGroupID, err := testutils.AttrFromState(s, "data.exoscale_security_group.default", "id")
						a.NoError(err, "unable to retrieve default Security Group ID from state")

						expectedUserData, _, err := utils.EncodeUserData(rUserData)
						if err != nil {
							return err
						}

						a.NotNil(testInstance.AntiAffinityGroupIDs)
						a.ElementsMatch([]string{*testAntiAffinityGroup.ID}, *testInstance.AntiAffinityGroupIDs)
						a.Equal(rDiskSize, *testInstance.DiskSize)
						a.NotNil(testInstance.ElasticIPIDs)
						a.ElementsMatch([]string{*testElasticIP.ID}, *testInstance.ElasticIPIDs)
						a.Equal(testutils.TestInstanceTypeIDTiny, *testInstance.InstanceTypeID)
						a.True(*testInstance.IPv6Enabled)
						a.Equal(rLabelValue, (*testInstance.Labels)["test"])
						a.Equal(rName, *testInstance.Name)
						a.NotNil(testInstance.PrivateNetworkIDs)
						a.ElementsMatch([]string{*testPrivateNetwork.ID}, *testInstance.PrivateNetworkIDs)
						a.Equal(*testSSHKey.Name, *testInstance.SSHKey)
						a.NotNil(testInstance.SecurityGroupIDs)
						a.ElementsMatch([]string{defaultSecurityGroupID, *testSecurityGroup.ID}, *testInstance.SecurityGroupIDs)
						a.Equal(rStateStopped, *testInstance.State)
						a.Equal(templateID, *testInstance.TemplateID)
						a.Equal(expectedUserData, *testInstance.UserData)

						return nil
					},
					testutils.CheckResourceState(r, testutils.CheckResourceStateValidateAttributes(testutils.TestAttrs{
						instance.AttrAntiAffinityGroupIDs + ".#": testutils.ValidateString("1"),
						instance.AttrCreatedAt:                   validation.ToDiagFunc(validation.NoZeroValues),
						instance.AttrDiskSize:                    testutils.ValidateString(fmt.Sprint(rDiskSize)),
						instance.AttrElasticIPIDs + ".#":         testutils.ValidateString("1"),
						instance.AttrIPv6:                        testutils.ValidateString("true"),
						instance.AttrIPv6Address:                 validation.ToDiagFunc(validation.IsIPv6Address),
						instance.AttrLabels + ".test":            testutils.ValidateString(rLabelValue),
						instance.AttrName:                        testutils.ValidateString(rName),
						instance.AttrNetworkInterface + ".#":     testutils.ValidateString("1"),
						instance.AttrPublicIPAddress:             validation.ToDiagFunc(validation.IsIPv4Address),
						instance.AttrSSHKey:                      testutils.ValidateString(rSSHKeyName),
						instance.AttrSecurityGroupIDs + ".#":     testutils.ValidateString("2"),
						instance.AttrState:                       testutils.ValidateString("stopped"),
						instance.AttrReverseDNS:                  testutils.ValidateString(rReverseDNS),
						instance.AttrTemplateID:                  validation.ToDiagFunc(validation.IsUUID),
						instance.AttrType:                        testutils.ValidateString(rType),
						instance.AttrUserData:                    testutils.ValidateString(rUserData),
						instance.AttrZone:                        testutils.ValidateString(testutils.TestZoneName),
					})),
				),
			},
			{
				// Update stopped testInstance
				Config: rConfigUpdateStopped,
				Check: resource.ComposeTestCheckFunc(
					testutils.CheckInstanceExists(r, &testInstance),
					func(s *terraform.State) error {
						a := require.New(t)

						defaultSecurityGroupID, err := testutils.AttrFromState(s, "data.exoscale_security_group.default", "id")
						a.NoError(err, "unable to retrieve default Security Group ID from state")

						expectedUserData, _, err := utils.EncodeUserData(rUserDataUpdated)
						if err != nil {
							return err
						}

						a.NotNil(testInstance.AntiAffinityGroupIDs)
						a.ElementsMatch([]string{*testAntiAffinityGroup.ID}, *testInstance.AntiAffinityGroupIDs)
						a.Equal(rDiskSizeUpdated, *testInstance.DiskSize)
						a.Nil(testInstance.ElasticIPIDs)
						a.Equal(testutils.TestInstanceTypeIDSmall, *testInstance.InstanceTypeID)
						a.Equal(rLabelValueUpdated, (*testInstance.Labels)["test"])
						a.Equal(rNameUpdated, *testInstance.Name)
						a.Nil(testInstance.PrivateNetworkIDs)
						a.NotNil(testInstance.SecurityGroupIDs)
						a.ElementsMatch([]string{defaultSecurityGroupID}, *testInstance.SecurityGroupIDs)
						a.Equal(rStateStopped, *testInstance.State)
						a.Equal(expectedUserData, *testInstance.UserData)

						return nil
					},
					testutils.CheckResourceState(r, testutils.CheckResourceStateValidateAttributes(testutils.TestAttrs{
						instance.AttrDiskSize:                testutils.ValidateString(fmt.Sprint(rDiskSizeUpdated)),
						instance.AttrLabels + ".test":        testutils.ValidateString(rLabelValueUpdated),
						instance.AttrName:                    testutils.ValidateString(rNameUpdated),
						instance.AttrSecurityGroupIDs + ".#": testutils.ValidateString("1"),
						instance.AttrState:                   testutils.ValidateString("stopped"),
						instance.AttrReverseDNS:              testutils.ValidateString(rReverseDNSUpdated),
						instance.AttrType:                    testutils.ValidateString(rTypeUpdated),
						instance.AttrUserData:                testutils.ValidateString(rUserDataUpdated),
					})),
					resource.TestCheckNoResourceAttr(r, instance.AttrElasticIPIDs+".#"),
					resource.TestCheckNoResourceAttr(r, instance.AttrNetworkInterface+".#"),
				),
			},
			{
				// Start testInstance
				Config: rConfigStart,
				Check: resource.ComposeTestCheckFunc(
					testutils.CheckInstanceExists(r, &testInstance),
					func(s *terraform.State) error {
						a := require.New(t)

						defaultSecurityGroupID, err := testutils.AttrFromState(s, "data.exoscale_security_group.default", "id")
						a.NoError(err, "unable to retrieve default Security Group ID from state")

						expectedUserData, _, err := utils.EncodeUserData(rUserDataUpdated)
						if err != nil {
							return err
						}

						a.NotNil(testInstance.AntiAffinityGroupIDs)
						a.ElementsMatch([]string{*testAntiAffinityGroup.ID}, *testInstance.AntiAffinityGroupIDs)
						a.Equal(rDiskSizeUpdated, *testInstance.DiskSize)
						a.Nil(testInstance.ElasticIPIDs)
						a.Equal(testutils.TestInstanceTypeIDSmall, *testInstance.InstanceTypeID)
						a.Equal(rLabelValueUpdated, (*testInstance.Labels)["test"])
						a.Equal(rNameUpdated, *testInstance.Name)
						a.Nil(testInstance.PrivateNetworkIDs)
						a.NotNil(testInstance.SecurityGroupIDs)
						a.ElementsMatch([]string{defaultSecurityGroupID}, *testInstance.SecurityGroupIDs)
						a.Equal(rStateRunning, *testInstance.State)
						a.Equal(expectedUserData, *testInstance.UserData)

						return nil
					},
					testutils.CheckResourceState(r, testutils.CheckResourceStateValidateAttributes(testutils.TestAttrs{
						instance.AttrDiskSize:                testutils.ValidateString(fmt.Sprint(rDiskSizeUpdated)),
						instance.AttrLabels + ".test":        testutils.ValidateString(rLabelValueUpdated),
						instance.AttrName:                    testutils.ValidateString(rNameUpdated),
						instance.AttrSecurityGroupIDs + ".#": testutils.ValidateString("1"),
						instance.AttrState:                   testutils.ValidateString("running"),
						instance.AttrReverseDNS:              validation.ToDiagFunc(validation.StringIsEmpty),
						instance.AttrType:                    testutils.ValidateString(rTypeUpdated),
						instance.AttrUserData:                testutils.ValidateString(rUserDataUpdated),
					})),
					resource.TestCheckNoResourceAttr(r, instance.AttrElasticIPIDs+".#"),
					resource.TestCheckNoResourceAttr(r, instance.AttrNetworkInterface+".#"),
				),
			},
			{
				// Update running Instance
				Config: rConfigUpdateStarted,
				Check: resource.ComposeTestCheckFunc(
					testutils.CheckInstanceExists(r, &testInstance),
					func(s *terraform.State) error {
						a := require.New(t)

						defaultSecurityGroupID, err := testutils.AttrFromState(s, "data.exoscale_security_group.default", "id")
						a.NoError(err, "unable to retrieve default Security Group ID from state")

						expectedUserData, _, err := utils.EncodeUserData(rUserDataUpdated)
						if err != nil {
							return err
						}

						a.NotNil(testInstance.AntiAffinityGroupIDs)
						a.ElementsMatch([]string{*testAntiAffinityGroup.ID}, *testInstance.AntiAffinityGroupIDs)
						a.Equal(rDiskSizeUpdated2, *testInstance.DiskSize)
						a.Nil(testInstance.ElasticIPIDs)
						a.Equal(testutils.TestInstanceTypeIDSmall, *testInstance.InstanceTypeID)
						a.Equal(rLabelValueUpdated, (*testInstance.Labels)["test"])
						a.Equal(rNameUpdated, *testInstance.Name)
						a.Nil(testInstance.PrivateNetworkIDs)
						a.NotNil(testInstance.SecurityGroupIDs)
						a.ElementsMatch([]string{defaultSecurityGroupID}, *testInstance.SecurityGroupIDs)
						a.Equal(rStateRunning, *testInstance.State)
						a.Equal(expectedUserData, *testInstance.UserData)

						return nil
					},
					testutils.CheckResourceState(r, testutils.CheckResourceStateValidateAttributes(testutils.TestAttrs{
						instance.AttrDiskSize:                testutils.ValidateString(fmt.Sprint(rDiskSizeUpdated2)),
						instance.AttrLabels + ".test":        testutils.ValidateString(rLabelValueUpdated),
						instance.AttrName:                    testutils.ValidateString(rNameUpdated),
						instance.AttrSecurityGroupIDs + ".#": testutils.ValidateString("1"),
						instance.AttrState:                   testutils.ValidateString("running"),
						instance.AttrReverseDNS:              validation.ToDiagFunc(validation.StringIsEmpty),
						instance.AttrType:                    testutils.ValidateString(rTypeUpdated),
						instance.AttrUserData:                testutils.ValidateString(rUserDataUpdated),
					})),
					resource.TestCheckNoResourceAttr(r, instance.AttrElasticIPIDs+".#"),
					resource.TestCheckNoResourceAttr(r, instance.AttrNetworkInterface+".#"),
				),
			},
			{
				// Import
				ResourceName: r,
				ImportStateIdFunc: func(testInstance *egoscale.Instance) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", *testInstance.ID, testutils.TestZoneName), nil
					}
				}(&testInstance),
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{instance.AttrPrivateNetworkIDs, instance.AttrPrivate},
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return testutils.CheckResourceAttributes(
						testutils.TestAttrs{
							instance.AttrDiskSize:                testutils.ValidateString(fmt.Sprint(rDiskSizeUpdated2)),
							instance.AttrLabels + ".test":        testutils.ValidateString(rLabelValueUpdated),
							instance.AttrName:                    testutils.ValidateString(rNameUpdated),
							instance.AttrSecurityGroupIDs + ".#": testutils.ValidateString("1"),
							instance.AttrState:                   testutils.ValidateString("running"),
							instance.AttrType:                    testutils.ValidateString(rTypeUpdated),
							instance.AttrUserData:                testutils.ValidateString(rUserDataUpdated),
						},
						func(s []*terraform.InstanceState) map[string]string {
							for _, state := range s {
								if state.ID == *testInstance.ID {
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

	// Test for managed network interface
	testInstance = egoscale.Instance{}
	testPrivateNetwork = egoscale.PrivateNetwork{}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testutils.AccPreCheck(t) },
		ProviderFactories: testutils.Providers(),
		CheckDestroy:      testutils.CheckInstanceDestroy(&testInstance),
		Steps: []resource.TestStep{
			{
				Config: rConfigCreateManaged,
				Check: resource.ComposeTestCheckFunc(
					testutils.CheckInstanceExists(r, &testInstance),
					testutils.CheckPrivateNetworkExists("exoscale_private_network.test", &testPrivateNetwork),
					func(s *terraform.State) error {
						a := require.New(t)

						a.Equal(rDiskSize, *testInstance.DiskSize)
						a.Equal(testutils.TestInstanceTypeIDTiny, *testInstance.InstanceTypeID)
						a.Equal(rName, *testInstance.Name)
						a.NotNil(testInstance.PrivateNetworkIDs)
						a.ElementsMatch([]string{*testPrivateNetwork.ID}, *testInstance.PrivateNetworkIDs)

						return nil
					},
					testutils.CheckResourceState(r, testutils.CheckResourceStateValidateAttributes(testutils.TestAttrs{
						instance.AttrCreatedAt:                          validation.ToDiagFunc(validation.NoZeroValues),
						instance.AttrDiskSize:                           testutils.ValidateString(fmt.Sprint(rDiskSize)),
						instance.AttrName:                               testutils.ValidateString(rName),
						instance.AttrNetworkInterface + ".#":            testutils.ValidateString("1"),
						instance.AttrNetworkInterface + ".0.ip_address": testutils.ValidateString("10.0.0.100"),
						instance.AttrTemplateID:                         validation.ToDiagFunc(validation.IsUUID),
						instance.AttrType:                               testutils.ValidateString(rType),
						instance.AttrZone:                               testutils.ValidateString(testutils.TestZoneName),
					})),
				),
			},
		},
	})
}
