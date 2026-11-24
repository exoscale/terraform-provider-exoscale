package instance_pool_test

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"

	v3 "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

type privateTestdataSpec struct {
	testutils.TestdataSpec
	Private bool
}

func testResourcePrivate(t *testing.T) {
	t.Parallel()

	var (
		instancePool v3.InstancePool

		testdataSpec = privateTestdataSpec{
			TestdataSpec: testutils.TestdataSpec{
				ID:   time.Now().UnixNano(),
				Zone: testutils.TestZoneName,
			},
			Private: true,
		}
		publicSpec = privateTestdataSpec{TestdataSpec: testdataSpec.TestdataSpec}
		ds         = "data.exoscale_instance_pool.test"
	)

	checkPublicIPAssignment := func(expected v3.PublicIPAssignment) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			require.Equal(t, expected, instancePool.PublicIPAssignment)
			return nil
		}
	}

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
					checkPublicIPAssignment(v3.PublicIPAssignmentNone),
					resource.TestCheckResourceAttr(poolResource, "private", "true"),
					resource.TestCheckResourceAttr(poolResource, "ipv6", "false"),
					resource.TestCheckResourceAttr(poolResource, "instances.#", "1"),
					resource.TestCheckResourceAttr(poolResource, "instances.0.public_ip_address", ""),
					resource.TestCheckResourceAttr(ds, "private", "true"),
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

			// 4 The public IP assignment of a private pool can't be updated.
			{
				Config: testutils.ParseTestdataConfig("./testdata/009.resource_private.tf.tmpl", &publicSpec),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(poolResource, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.CheckInstancePoolExists(poolResource, &instancePool),
					checkPublicIPAssignment(v3.PublicIPAssignmentInet4),
					resource.TestCheckResourceAttr(poolResource, "private", "false"),
					resource.TestCheckResourceAttrSet(poolResource, "instances.0.public_ip_address"),
					resource.TestCheckResourceAttr(ds, "private", "false"),
				),
			},
		},
	})
}
