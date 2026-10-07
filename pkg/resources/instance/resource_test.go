package instance_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"

	v3 "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const instanceResource = "exoscale_compute_instance.test_instance"

func testResource(t *testing.T) {
	t.Parallel()

	var (
		testInstance v3.Instance

		testdataSpec = testutils.TestdataSpec{
			ID:   time.Now().UnixNano(),
			Zone: testutils.TestZoneName,
		}
		name        = testutils.ResourceName(testdataSpec.ID)
		label       = fmt.Sprintf("label-%d", testdataSpec.ID)
		userData    = fmt.Sprintf("user-data-%d", testdataSpec.ID)
		sgResource  = "exoscale_security_group.test_sg"
		aagResource = "exoscale_anti_affinity_group.test_aag"
		pnResource  = "exoscale_private_network.test_pn"
		eipResource = "exoscale_elastic_ip.test_eip"
		keyResource = "exoscale_ssh_key.test_key"
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testutils.CheckInstanceDestroyV3(&testInstance),
		Steps: []resource.TestStep{
			// 1 Create a stopped instance, with every attachment.
			{
				Config: testutils.ParseTestdataConfig("./testdata/001.instance_create_stopped.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.CheckInstanceExistsV3(instanceResource, &testInstance),
					func(s *terraform.State) error {
						a := require.New(t)

						expectedUserData, _, err := utils.EncodeUserData(userData)
						a.NoError(err)

						a.Equal(testutils.TestInstanceTypeIDTiny, testInstance.InstanceType.ID.String())
						a.Equal(v3.PublicIPAssignmentInet4, testInstance.PublicIPAssignment)
						a.Equal(v3.InstanceStateStopped, testInstance.State)
						a.Equal(expectedUserData, testInstance.UserData)
						a.Len(testInstance.AntiAffinityGroups, 1)
						a.Len(testInstance.ElasticIPS, 1)
						a.Len(testInstance.PrivateNetworks, 1)
						a.Len(testInstance.SecurityGroups, 2)
						a.NotNil(testInstance.SSHKey)

						return nil
					},
					resource.TestCheckResourceAttr(instanceResource, "name", name),
					resource.TestCheckResourceAttr(instanceResource, "type", "standard.tiny"),
					resource.TestCheckResourceAttr(instanceResource, "disk_size", "10"),
					resource.TestCheckResourceAttr(instanceResource, "state", "stopped"),
					resource.TestCheckResourceAttr(instanceResource, "ipv6", "false"),
					resource.TestCheckResourceAttr(instanceResource, "enable_tpm", "false"),
					resource.TestCheckResourceAttr(instanceResource, "enable_secure_boot", "true"),
					resource.TestCheckResourceAttr(instanceResource, "user_data", userData),
					resource.TestCheckResourceAttr(instanceResource, "reverse_dns", "tf-provider-test.exoscale.com"),
					resource.TestCheckResourceAttr(instanceResource, "labels.%", "1"),
					resource.TestCheckResourceAttr(instanceResource, "labels.test", label),
					resource.TestCheckResourceAttr(instanceResource, "anti_affinity_group_ids.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(instanceResource, "anti_affinity_group_ids.*", aagResource, "id"),
					resource.TestCheckResourceAttr(instanceResource, "security_group_ids.#", "2"),
					resource.TestCheckTypeSetElemAttrPair(instanceResource, "security_group_ids.*", sgResource, "id"),
					resource.TestCheckResourceAttr(instanceResource, "elastic_ip_ids.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(instanceResource, "elastic_ip_ids.*", eipResource, "id"),
					resource.TestCheckResourceAttrPair(instanceResource, "ssh_key", keyResource, "name"),
					resource.TestCheckResourceAttr(instanceResource, "network_interface.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(instanceResource, "network_interface.*.network_id", pnResource, "id"),
					resource.TestCheckResourceAttrSet(instanceResource, "network_interface.0.mac_address"),
					resource.TestCheckResourceAttr(instanceResource, "private_network_ids.#", "1"),
					resource.TestCheckResourceAttrSet(instanceResource, "created_at"),
					resource.TestCheckResourceAttrSet(instanceResource, "mac_address"),
					resource.TestCheckResourceAttrSet(instanceResource, "public_ip_address"),
					resource.TestCheckResourceAttrPair(instanceResource, "template_id", "data.exoscale_template.test_template", "id"),
					resource.TestCheckResourceAttr(instanceResource, "zone", testdataSpec.Zone),
				),
			},

			// 2 Update the stopped instance: scale, resize, enable IPv6 and TPM,
			// detach the Elastic IP, the Private Network and a Security Group.
			{
				Config: testutils.ParseTestdataConfig("./testdata/002.instance_update_stopped.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.CheckInstanceExistsV3(instanceResource, &testInstance),
					func(s *terraform.State) error {
						a := require.New(t)

						a.Equal(testutils.TestInstanceTypeIDSmall, testInstance.InstanceType.ID.String())
						a.Equal(v3.PublicIPAssignmentDual, testInstance.PublicIPAssignment)
						a.Equal(v3.InstanceStateStopped, testInstance.State)
						a.Equal(int64(20), testInstance.DiskSize)
						a.Empty(testInstance.ElasticIPS)
						a.Empty(testInstance.PrivateNetworks)
						a.Len(testInstance.SecurityGroups, 1)

						return nil
					},
					resource.TestCheckResourceAttr(instanceResource, "name", name+"-updated"),
					resource.TestCheckResourceAttr(instanceResource, "type", "standard.small"),
					resource.TestCheckResourceAttr(instanceResource, "disk_size", "20"),
					resource.TestCheckResourceAttr(instanceResource, "state", "stopped"),
					resource.TestCheckResourceAttr(instanceResource, "ipv6", "true"),
					resource.TestCheckResourceAttrSet(instanceResource, "ipv6_address"),
					resource.TestCheckResourceAttr(instanceResource, "enable_tpm", "true"),
					resource.TestCheckResourceAttr(instanceResource, "user_data", userData+"-updated"),
					resource.TestCheckResourceAttr(instanceResource, "reverse_dns", "tf-provider-updated-test.exoscale.com"),
					resource.TestCheckResourceAttr(instanceResource, "labels.test", label+"-updated"),
					resource.TestCheckResourceAttr(instanceResource, "security_group_ids.#", "1"),
					resource.TestCheckResourceAttr(instanceResource, "elastic_ip_ids.#", "0"),
					resource.TestCheckResourceAttr(instanceResource, "network_interface.#", "0"),
				),
			},

			// 3 Start the instance and clear its reverse DNS.
			{
				Config: testutils.ParseTestdataConfig("./testdata/003.instance_start.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.CheckInstanceExistsV3(instanceResource, &testInstance),
					func(s *terraform.State) error {
						require.Equal(t, v3.InstanceStateRunning, testInstance.State)
						return nil
					},
					resource.TestCheckResourceAttr(instanceResource, "state", "running"),
					resource.TestCheckResourceAttr(instanceResource, "reverse_dns", ""),
					resource.TestCheckResourceAttr(instanceResource, "disk_size", "20"),
				),
			},

			// 4 Resize the disk of the running instance: it is stopped, then
			// started again.
			{
				Config: testutils.ParseTestdataConfig("./testdata/004.instance_update_started.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.CheckInstanceExistsV3(instanceResource, &testInstance),
					func(s *terraform.State) error {
						a := require.New(t)

						a.Equal(int64(30), testInstance.DiskSize)
						a.Equal(v3.InstanceStateRunning, testInstance.State)

						return nil
					},
					resource.TestCheckResourceAttr(instanceResource, "disk_size", "30"),
					resource.TestCheckResourceAttr(instanceResource, "state", "running"),
				),
			},

			// 5 Import the instance: <ID>@<ZONE>.
			{
				ResourceName: instanceResource,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf(
						"%s@%s",
						s.RootModule().Resources[instanceResource].Primary.ID,
						testdataSpec.Zone,
					), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					// SSH keys are only used at creation time.
					"ssh_key",
					"ssh_keys",
					// An instance without reverse DNS is imported with none, while the
					// state still holds the empty string step 3 configured.
					"reverse_dns",
					// An empty list in the configuration is imported as no list.
					"elastic_ip_ids",
					"timeouts",
				},
			},

			// 6 Drop the anti-affinity group, which replaces the instance.
			{
				Config: testutils.ParseTestdataConfig("./testdata/005.instance_detach.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.CheckInstanceExistsV3(instanceResource, &testInstance),
					func(s *terraform.State) error {
						a := require.New(t)

						a.Empty(testInstance.AntiAffinityGroups)
						a.Empty(testInstance.ElasticIPS)
						a.Empty(testInstance.PrivateNetworks)
						a.Len(testInstance.SecurityGroups, 1)

						return nil
					},
					resource.TestCheckNoResourceAttr(instanceResource, "anti_affinity_group_ids.#"),
					resource.TestCheckResourceAttr(instanceResource, "security_group_ids.#", "1"),
				),
			},
		},
	})
}

func testResourceManagedNetworkInterface(t *testing.T) {
	t.Parallel()

	var (
		testInstance v3.Instance

		testdataSpec = testutils.TestdataSpec{
			ID:   time.Now().UnixNano(),
			Zone: testutils.TestZoneName,
		}
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testutils.CheckInstanceDestroyV3(&testInstance),
		Steps: []resource.TestStep{
			{
				Config: testutils.ParseTestdataConfig("./testdata/006.instance_managed_nif.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.CheckInstanceExistsV3(instanceResource, &testInstance),
					func(s *terraform.State) error {
						require.Len(t, testInstance.PrivateNetworks, 1)
						return nil
					},
					resource.TestCheckResourceAttr(instanceResource, "network_interface.#", "1"),
					resource.TestCheckResourceAttr(instanceResource, "network_interface.0.ip_address", "10.0.0.100"),
					resource.TestCheckResourceAttrPair(instanceResource, "network_interface.0.network_id", "exoscale_private_network.test_pn", "id"),
					resource.TestCheckResourceAttrSet(instanceResource, "network_interface.0.mac_address"),
				),
			},
		},
	})
}

func testResourceSSHKeys(t *testing.T) {
	t.Parallel()

	var (
		testInstance v3.Instance

		testdataSpec = testutils.TestdataSpec{
			ID:   time.Now().UnixNano(),
			Zone: testutils.TestZoneName,
		}
		name = testutils.ResourceName(testdataSpec.ID)
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testutils.CheckInstanceDestroyV3(&testInstance),
		Steps: []resource.TestStep{
			{
				Config: testutils.ParseTestdataConfig("./testdata/007.instance_ssh_keys.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.CheckInstanceExistsV3(instanceResource, &testInstance),
					func(s *terraform.State) error {
						a := require.New(t)

						a.Len(testInstance.SSHKeys, 2)
						a.ElementsMatch(
							[]string{name, name + "-2"},
							[]string{testInstance.SSHKeys[0].Name, testInstance.SSHKeys[1].Name},
						)

						return nil
					},
					resource.TestCheckResourceAttr(instanceResource, "ssh_keys.#", "2"),
					resource.TestCheckTypeSetElemAttr(instanceResource, "ssh_keys.*", name),
					resource.TestCheckTypeSetElemAttr(instanceResource, "ssh_keys.*", name+"-2"),
				),
			},
		},
	})
}
