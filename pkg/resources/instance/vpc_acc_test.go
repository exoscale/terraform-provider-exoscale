package instance_test

import (
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	v3 "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

type vpcInterfaceSpec struct {
	Subnet string
	IP     string
}

type vpcTestdataSpec struct {
	testutils.TestdataSpec
	Interfaces []vpcInterfaceSpec
}

func testResourceVPC(t *testing.T) {
	t.Parallel()

	var (
		testInstance v3.Instance
		spec         = vpcTestdataSpec{
			TestdataSpec: testutils.TestdataSpec{
				ID:   time.Now().UnixNano(),
				Zone: testutils.TestZoneName,
			},
		}
		a = vpcInterfaceSpec{Subnet: "a", IP: "10.0.0.22"}
		b = vpcInterfaceSpec{Subnet: "b", IP: "auto"}
	)

	config := func(ifaces ...vpcInterfaceSpec) string {
		spec.Interfaces = ifaces
		return testutils.ParseTestdataConfig("./testdata/017.instance_vpc.tf.tmpl", &spec)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testutils.CheckInstanceDestroyV3(&testInstance),
		Steps: []resource.TestStep{
			// Create attached to [a, b].
			{
				Config: config(a, b),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.CheckInstanceExistsV3(instanceResource, &testInstance),
					resource.TestCheckResourceAttrPair(instanceResource, "vpc.0.id", "exoscale_vpc.test_vpc", "id"),
					resource.TestCheckResourceAttr(instanceResource, "vpc.0.interfaces.#", "2"),
					resource.TestCheckResourceAttrPair(instanceResource, "vpc.0.interfaces.0.subnet_id", "exoscale_vpc_subnet.test_a", "id"),
					resource.TestCheckResourceAttr(instanceResource, "vpc.0.interfaces.0.ipv4_address_allocated", "10.0.0.22"),
					resource.TestCheckResourceAttrPair(instanceResource, "vpc.0.interfaces.1.subnet_id", "exoscale_vpc_subnet.test_b", "id"),
					resource.TestCheckResourceAttr(instanceResource, "vpc.0.interfaces.1.ipv4_address", "auto"),
					resource.TestCheckResourceAttrSet(instanceResource, "vpc.0.interfaces.1.ipv4_address_allocated"),
				),
			},
			// Import: an "auto" address is imported as the allocated one.
			{
				ResourceName: instanceResource,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return s.RootModule().Resources[instanceResource].Primary.ID + "@" + spec.Zone, nil
				},
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"vpc.0.interfaces.1.ipv4_address",
					"timeouts",
				},
			},
			// Swap: both are reattached, the fixed IP is reused.
			{
				Config: config(b, a),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(instanceResource, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(instanceResource, "vpc.0.interfaces.0.subnet_id", "exoscale_vpc_subnet.test_b", "id"),
					resource.TestCheckResourceAttrPair(instanceResource, "vpc.0.interfaces.1.subnet_id", "exoscale_vpc_subnet.test_a", "id"),
					resource.TestCheckResourceAttr(instanceResource, "vpc.0.interfaces.1.ipv4_address_allocated", "10.0.0.22"),
				),
			},
			// Detach the last interface.
			{
				Config: config(b),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(instanceResource, "vpc.0.interfaces.#", "1"),
					resource.TestCheckResourceAttrPair(instanceResource, "vpc.0.interfaces.0.subnet_id", "exoscale_vpc_subnet.test_b", "id"),
				),
			},
			// Removing the block detaches everything.
			{
				Config: config(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(instanceResource, "vpc.#", "0"),
				),
			},
		},
	})
}
