package privatenetwork_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	tftest "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func Test_Resource_Private_Network(t *testing.T) {
	t.Parallel()

	resource := "exoscale_private_network.test_pn"
	datasourceByID := "data.exoscale_private_network.test_pn_id"
	datasourceByName := "data.exoscale_private_network.test_pn_name"

	testDataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}

	tftest.Test(t, tftest.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []tftest.TestStep{
			// Create private network and datasource
			{
				Config: testutils.ParseTestdataConfig("./testdata/001.pn_create.tf.tmpl", &testDataSpec),
				Check: tftest.ComposeAggregateTestCheckFunc(
					// test resource
					tftest.TestCheckResourceAttr(resource, "name", testutils.ResourceName(testDataSpec.ID)),
					tftest.TestCheckResourceAttr(resource, "description", "description-test"),
					tftest.TestCheckResourceAttr(resource, "start_ip", "10.0.0.10"),
					tftest.TestCheckResourceAttr(resource, "end_ip", "10.0.0.250"),
					tftest.TestCheckResourceAttr(resource, "netmask", "255.255.255.0"),
					tftest.TestCheckResourceAttr(resource, "labels.%", "1"),
					tftest.TestCheckResourceAttr(resource, "labels.A", "B"),

					// test datasource
					tftest.TestCheckResourceAttrPair(resource, "name", datasourceByID, "name"),
					tftest.TestCheckResourceAttrPair(resource, "description", datasourceByID, "description"),
					tftest.TestCheckResourceAttrPair(resource, "start_ip", datasourceByID, "start_ip"),
					tftest.TestCheckResourceAttrPair(resource, "end_ip", datasourceByID, "end_ip"),
					tftest.TestCheckResourceAttrPair(resource, "netmask", datasourceByID, "netmask"),
					tftest.TestCheckResourceAttrPair(resource, "labels.A", datasourceByID, "labels.A"),

					tftest.TestCheckResourceAttrPair(resource, "name", datasourceByName, "name"),
					tftest.TestCheckResourceAttrPair(resource, "description", datasourceByName, "description"),
					tftest.TestCheckResourceAttrPair(resource, "start_ip", datasourceByName, "start_ip"),
					tftest.TestCheckResourceAttrPair(resource, "end_ip", datasourceByName, "end_ip"),
					tftest.TestCheckResourceAttrPair(resource, "netmask", datasourceByName, "netmask"),
					tftest.TestCheckResourceAttrPair(resource, "labels.A", datasourceByName, "labels.A"),
				),
			},

			// test update (without datasource)
			{
				Config: testutils.ParseTestdataConfig("./testdata/002.pn_update.tf.tmpl", &testDataSpec),
				Check: tftest.ComposeAggregateTestCheckFunc(
					// test resource
					tftest.TestCheckResourceAttr(resource, "name", testutils.ResourceName(testDataSpec.ID)),
					tftest.TestCheckResourceAttr(resource, "description", "description-test-updated"),
					tftest.TestCheckResourceAttr(resource, "start_ip", "10.0.0.5"),
					tftest.TestCheckResourceAttr(resource, "end_ip", "10.0.0.254"),
					tftest.TestCheckResourceAttr(resource, "netmask", "255.255.0.0"),
					tftest.TestCheckResourceAttr(resource, "labels.%", "1"),
					tftest.TestCheckResourceAttr(resource, "labels.A", "C"),
				),
			},

			// Import private network
			{
				ResourceName: resource,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf("%s@%s", s.RootModule().Resources[resource].Primary.ID, testDataSpec.Zone), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
