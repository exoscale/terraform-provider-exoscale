package instance_pool_test

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"

	v3 "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func testResourcePrivate(t *testing.T) {
	t.Parallel()

	var (
		instancePool v3.InstancePool

		testdataSpec = testutils.TestdataSpec{
			ID:   time.Now().UnixNano(),
			Zone: testutils.TestZoneName,
		}
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testutils.CheckInstancePoolDestroy(&instancePool),
		Steps: []resource.TestStep{
			// 1 A private pool cannot have IPv6.
			{
				Config:      testutils.ParseTestdataConfig("./testdata/010.resource_private_ipv6.tf.tmpl", &testdataSpec),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile("ipv6 cannot be enabled on a private instance pool"),
			},

			// 2 Create a private pool.
			{
				Config: testutils.ParseTestdataConfig("./testdata/009.resource_private.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.CheckInstancePoolExists(poolResource, &instancePool),
					func(s *terraform.State) error {
						a := require.New(t)

						a.Equal(v3.PublicIPAssignmentNone, instancePool.PublicIPAssignment)
						a.Len(instancePool.PrivateNetworks, 1)

						return nil
					},
					resource.TestCheckResourceAttr(poolResource, "private", "true"),
					resource.TestCheckResourceAttr(poolResource, "ipv6", "false"),
					resource.TestCheckResourceAttr(poolResource, "network_ids.#", "1"),
					resource.TestCheckResourceAttr(poolResource, "instances.#", "1"),
				),
			},

			// 3 Import: <ID>@<ZONE>.
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
					"state",
					"timeouts",
				},
			},
		},
	})
}
