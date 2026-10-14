package instance_pool_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"

	v3 "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/resources/instance_pool"
	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const poolResource = "exoscale_instance_pool.test"

func testResource(t *testing.T) {
	t.Parallel()

	var (
		instancePool v3.InstancePool

		testdataSpec = testutils.TestdataSpec{
			ID:   time.Now().UnixNano(),
			Zone: testutils.TestZoneName,
		}
		name        = testutils.ResourceName(testdataSpec.ID)
		description = fmt.Sprintf("description-%d", testdataSpec.ID)
		label       = fmt.Sprintf("label-%d", testdataSpec.ID)
		userData    = fmt.Sprintf("user-data-%d", testdataSpec.ID)
		aagResource = "exoscale_anti_affinity_group.test"
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testutils.CheckInstancePoolDestroy(&instancePool),
		Steps: []resource.TestStep{
			// 1 Create
			{
				Config: testutils.ParseTestdataConfig("./testdata/001.resource_create.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.CheckInstancePoolExists(poolResource, &instancePool),
					func(s *terraform.State) error {
						a := require.New(t)

						templateID, err := testutils.AttrFromState(s, "data.exoscale_template.ubuntu", "id")
						a.NoError(err, "unable to retrieve template ID from state")

						expectedUserData, _, err := utils.EncodeUserData(userData)
						a.NoError(err)

						a.Len(instancePool.AntiAffinityGroups, 1)
						a.Equal(description, instancePool.Description)
						a.Equal(int64(10), instancePool.DiskSize)
						a.Equal("test", instancePool.InstancePrefix)
						a.Len(instancePool.Instances, 1)
						a.Equal(testutils.TestInstanceTypeIDTiny, instancePool.InstanceType.ID.String())
						a.True(*instancePool.Ipv6Enabled)
						a.Equal(label, instancePool.Labels["test"])
						a.Equal(name, instancePool.Name)
						a.Len(instancePool.SecurityGroups, 1)
						a.Equal(int64(1), instancePool.Size)
						a.Equal(templateID, instancePool.Template.ID.String())
						a.Equal(expectedUserData, instancePool.UserData)

						return nil
					},
					resource.TestCheckResourceAttr(poolResource, "anti_affinity_group_ids.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(poolResource, "anti_affinity_group_ids.*", aagResource, "id"),
					resource.TestCheckResourceAttr(poolResource, "description", description),
					resource.TestCheckResourceAttr(poolResource, "disk_size", "10"),
					resource.TestCheckResourceAttr(poolResource, "ipv6", "true"),
					resource.TestCheckResourceAttr(poolResource, "instance_prefix", "test"),
					resource.TestCheckResourceAttr(poolResource, "instance_type", "standard.tiny"),
					resource.TestCheckResourceAttr(poolResource, "labels.%", "1"),
					resource.TestCheckResourceAttr(poolResource, "labels.test", label),
					resource.TestCheckResourceAttr(poolResource, "name", name),
					resource.TestCheckResourceAttr(poolResource, "security_group_ids.#", "1"),
					resource.TestCheckResourceAttrPair(poolResource, "security_group_ids.0", "data.exoscale_security_group.default", "id"),
					resource.TestCheckResourceAttr(poolResource, "size", "1"),
					resource.TestCheckResourceAttr(poolResource, "min_available", "0"),
					resource.TestCheckResourceAttrSet(poolResource, "state"),
					resource.TestCheckResourceAttrPair(poolResource, "template_id", "data.exoscale_template.ubuntu", "id"),
					resource.TestCheckResourceAttr(poolResource, "user_data", userData),
					resource.TestCheckResourceAttr(poolResource, "virtual_machines.#", "1"),
					resource.TestCheckResourceAttr(poolResource, "instances.#", "1"),
					resource.TestCheckResourceAttrSet(poolResource, "instances.0.id"),
					resource.TestCheckResourceAttrSet(poolResource, "instances.0.name"),
					resource.TestCheckResourceAttr(poolResource, "zone", testdataSpec.Zone),
				),
			},

			// 2 Update
			{
				Config: testutils.ParseTestdataConfig("./testdata/002.resource_update.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.CheckInstancePoolExists(poolResource, &instancePool),
					func(s *terraform.State) error {
						a := require.New(t)

						templateID, err := testutils.AttrFromState(s, "data.exoscale_template.debian", "id")
						a.NoError(err, "unable to retrieve template ID from state")

						expectedUserData, _, err := utils.EncodeUserData(userData + "-updated")
						a.NoError(err)

						a.Len(instancePool.AntiAffinityGroups, 1)
						a.Equal(description+"-updated", instancePool.Description)
						a.Equal(int64(20), instancePool.DiskSize)
						a.Equal(instance_pool.DefaultInstancePrefix, instancePool.InstancePrefix)
						a.Len(instancePool.Instances, 2)
						a.Equal(testutils.TestInstanceTypeIDSmall, instancePool.InstanceType.ID.String())
						a.True(*instancePool.Ipv6Enabled)
						a.Equal(label+"-updated", instancePool.Labels["test"])
						a.Equal(name+"-updated", instancePool.Name)
						a.Len(instancePool.PrivateNetworks, 1)
						a.Empty(instancePool.SecurityGroups)
						a.Equal(int64(2), instancePool.Size)
						a.Equal(int64(1), instancePool.MinAvailable)
						a.Equal(name, instancePool.SSHKey.Name)
						a.Equal(templateID, instancePool.Template.ID.String())
						a.Equal(expectedUserData, instancePool.UserData)

						return nil
					},
					resource.TestCheckResourceAttr(poolResource, "anti_affinity_group_ids.#", "1"),
					resource.TestCheckResourceAttr(poolResource, "description", description+"-updated"),
					resource.TestCheckResourceAttr(poolResource, "disk_size", "20"),
					resource.TestCheckResourceAttr(poolResource, "instance_prefix", instance_pool.DefaultInstancePrefix),
					resource.TestCheckResourceAttr(poolResource, "instance_type", "standard.small"),
					resource.TestCheckResourceAttr(poolResource, "ipv6", "true"),
					resource.TestCheckResourceAttrPair(poolResource, "key_pair", "exoscale_ssh_key.test", "name"),
					resource.TestCheckResourceAttr(poolResource, "labels.test", label+"-updated"),
					resource.TestCheckResourceAttr(poolResource, "name", name+"-updated"),
					resource.TestCheckResourceAttr(poolResource, "network_ids.#", "1"),
					resource.TestCheckResourceAttrPair(poolResource, "network_ids.0", "exoscale_private_network.test", "id"),
					resource.TestCheckNoResourceAttr(poolResource, "security_group_ids.#"),
					resource.TestCheckResourceAttr(poolResource, "size", "2"),
					resource.TestCheckResourceAttr(poolResource, "min_available", "1"),
					resource.TestCheckResourceAttrPair(poolResource, "template_id", "data.exoscale_template.debian", "id"),
					resource.TestCheckResourceAttr(poolResource, "user_data", userData+"-updated"),
					resource.TestCheckResourceAttr(poolResource, "virtual_machines.#", "2"),
					resource.TestCheckResourceAttr(poolResource, "instances.#", "2"),
				),
			},

			// 3 Clear the optional attributes.
			{
				Config: testutils.ParseTestdataConfig("./testdata/003.resource_clear.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.CheckInstancePoolExists(poolResource, &instancePool),
					func(s *terraform.State) error {
						a := require.New(t)

						a.Empty(instancePool.Description)
						a.Empty(instancePool.Labels)
						a.Empty(instancePool.UserData)
						a.Empty(instancePool.PrivateNetworks)
						a.Len(instancePool.AntiAffinityGroups, 1)
						a.Equal(int64(2), instancePool.Size)

						return nil
					},
					resource.TestCheckNoResourceAttr(poolResource, "description"),
					resource.TestCheckNoResourceAttr(poolResource, "labels.%"),
					resource.TestCheckNoResourceAttr(poolResource, "user_data"),
					resource.TestCheckNoResourceAttr(poolResource, "key_pair"),
					resource.TestCheckNoResourceAttr(poolResource, "network_ids.#"),
					resource.TestCheckResourceAttr(poolResource, "anti_affinity_group_ids.#", "1"),
					resource.TestCheckResourceAttr(poolResource, "instances.#", "2"),
				),
			},

			// 4 Import: <ID>@<ZONE>.
			{
				ResourceName: poolResource,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf(
						"%s@%s",
						s.RootModule().Resources[poolResource].Primary.ID,
						testdataSpec.Zone,
					), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					// The pool may still be scaling when the test imports it.
					"state",
					"timeouts",
				},
			},
		},
	})
}
