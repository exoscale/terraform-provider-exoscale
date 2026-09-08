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

func Test_Resource_VPC_Subnet(t *testing.T) {
	t.Parallel()

	vpcResourceName := "exoscale_vpc.test_vpc"
	resourceName := "exoscale_vpc_subnet.test_subnet"
	datasourceByID := "data.exoscale_vpc_subnet.test_subnet_by_id"
	datasourceByName := "data.exoscale_vpc_subnet.test_subnet_by_name"

	testDataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}

	tftest.Test(t, tftest.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []tftest.TestStep{
			// Create VPC, Subnet and data sources (match by id and name)
			{
				Config: testutils.ParseTestdataConfig("./testdata/003.subnet_create.tf.tmpl", &testDataSpec),
				Check: tftest.ComposeAggregateTestCheckFunc(
					tftest.TestCheckResourceAttr(resourceName, "name", fmt.Sprintf("terraform-provider-test-subnet-%d", testDataSpec.ID)),
					tftest.TestCheckResourceAttr(resourceName, "description", "description-test"),
					tftest.TestCheckResourceAttr(resourceName, "ipv4_block", "10.20.0.0/24"),
					tftest.TestCheckResourceAttr(resourceName, "address_family", "inet4"),
					tftest.TestCheckResourceAttr(resourceName, "address_space", "private"),
					tftest.TestCheckResourceAttr(resourceName, "labels.%", "1"),
					tftest.TestCheckResourceAttr(resourceName, "labels.A", "B"),

					tftest.TestCheckResourceAttrPair(resourceName, "name", datasourceByID, "name"),
					tftest.TestCheckResourceAttrPair(resourceName, "ipv4_block", datasourceByID, "ipv4_block"),
					tftest.TestCheckResourceAttrPair(resourceName, "name", datasourceByName, "name"),
					tftest.TestCheckResourceAttrPair(resourceName, "ipv4_block", datasourceByName, "ipv4_block"),
				),
			},

			// Update (without data sources)
			{
				Config: testutils.ParseTestdataConfig("./testdata/004.subnet_update.tf.tmpl", &testDataSpec),
				Check: tftest.ComposeAggregateTestCheckFunc(
					tftest.TestCheckResourceAttr(resourceName, "name", fmt.Sprintf("terraform-provider-test-subnet-%d-updated", testDataSpec.ID)),
					tftest.TestCheckResourceAttr(resourceName, "description", "description-test-updated"),
					tftest.TestCheckResourceAttr(resourceName, "ipv4_block", "10.20.0.0/23"),
					tftest.TestCheckResourceAttr(resourceName, "labels.%", "1"),
					tftest.TestCheckResourceAttr(resourceName, "labels.A", "C"),
				),
			},

			// Import
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					vpcID := s.RootModule().Resources[vpcResourceName].Primary.ID
					subnetID := s.RootModule().Resources[resourceName].Primary.ID
					return fmt.Sprintf("%s@%s@%s", vpcID, subnetID, testDataSpec.Zone), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func Test_Resource_VPC_Subnet_Attachment(t *testing.T) {
	t.Parallel()

	instanceResourceName := "exoscale_compute_instance.test_instance"
	vpcResourceName := "exoscale_vpc.test_vpc"
	subnetResourceName := "exoscale_vpc_subnet.test_subnet"
	resourceName := "exoscale_vpc_subnet_attachment.test_attachment"

	testDataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}

	tftest.Test(t, tftest.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []tftest.TestStep{
			// Create instance, VPC, Subnet and attachment
			{
				Config: testutils.ParseTestdataConfig("./testdata/005.instance_attachment_create.tf.tmpl", &testDataSpec),
				Check: tftest.ComposeAggregateTestCheckFunc(
					tftest.TestCheckResourceAttr(resourceName, "ipv4_address", "10.22.0.5"),
					tftest.TestCheckResourceAttrPair(resourceName, "instance_id", instanceResourceName, "id"),
					tftest.TestCheckResourceAttrPair(resourceName, "vpc_id", vpcResourceName, "id"),
					tftest.TestCheckResourceAttrPair(resourceName, "subnet_id", subnetResourceName, "id"),
				),
			},

			// Import
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					instanceID := s.RootModule().Resources[instanceResourceName].Primary.ID
					vpcID := s.RootModule().Resources[vpcResourceName].Primary.ID
					subnetID := s.RootModule().Resources[subnetResourceName].Primary.ID
					return fmt.Sprintf("%s@%s@%s@%s", instanceID, vpcID, subnetID, testDataSpec.Zone), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
