package vpc_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	tftest "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func Test_Resource_VPC(t *testing.T) {
	t.Parallel()

	resourceName := "exoscale_vpc.test_vpc"
	datasourceByID := "data.exoscale_vpc.test_vpc_by_id"
	datasourceByName := "data.exoscale_vpc.test_vpc_by_name"

	testDataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}

	tftest.Test(t, tftest.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []tftest.TestStep{
			// Create VPC and data sources (match by id and name)
			{
				Config: testutils.ParseTestdataConfig("./testdata/001.vpc_create.tf.tmpl", &testDataSpec),
				Check: tftest.ComposeAggregateTestCheckFunc(
					tftest.TestCheckResourceAttr(resourceName, "name", testutils.ResourceName(testDataSpec.ID)),
					tftest.TestCheckResourceAttr(resourceName, "description", "description-test"),
					tftest.TestCheckResourceAttr(resourceName, "labels.%", "1"),
					tftest.TestCheckResourceAttr(resourceName, "labels.A", "B"),

					tftest.TestCheckResourceAttrPair(resourceName, "name", datasourceByID, "name"),
					tftest.TestCheckResourceAttrPair(resourceName, "description", datasourceByID, "description"),
					tftest.TestCheckResourceAttrPair(resourceName, "labels.A", datasourceByID, "labels.A"),

					tftest.TestCheckResourceAttrPair(resourceName, "name", datasourceByName, "name"),
					tftest.TestCheckResourceAttrPair(resourceName, "description", datasourceByName, "description"),
				),
			},

			// Update (without data sources)
			{
				Config: testutils.ParseTestdataConfig("./testdata/002.vpc_update.tf.tmpl", &testDataSpec),
				Check: tftest.ComposeAggregateTestCheckFunc(
					tftest.TestCheckResourceAttr(resourceName, "name", fmt.Sprintf("terraform-provider-test-updated-%d", testDataSpec.ID)),
					tftest.TestCheckResourceAttr(resourceName, "description", "description-test-updated"),
					tftest.TestCheckResourceAttr(resourceName, "labels.%", "1"),
					tftest.TestCheckResourceAttr(resourceName, "labels.A", "C"),
				),
			},

			// Import
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf("%s@%s", s.RootModule().Resources[resourceName].Primary.ID, testDataSpec.Zone), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
