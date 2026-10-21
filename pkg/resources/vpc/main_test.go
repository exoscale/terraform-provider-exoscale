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
					tftest.TestCheckResourceAttr(resourceName, "dns_servers.0", "8.8.8.8"),
					tftest.TestCheckResourceAttr(resourceName, "dns_servers.1", "1.1.1.1"),
					tftest.TestCheckResourceAttr(resourceName, "ntp_servers.0", "42.42.42.42"),
					tftest.TestCheckResourceAttr(resourceName, "ntp_servers.1", "43.43.43.43"),
					tftest.TestCheckResourceAttr(resourceName, "domain_search.0", "my.domain"),
					tftest.TestCheckResourceAttr(resourceName, "domain_search.1", "their.domain"),
					tftest.TestCheckResourceAttr(resourceName, "labels.%", "1"),
					tftest.TestCheckResourceAttr(resourceName, "labels.A", "B"),

					tftest.TestCheckResourceAttrPair(resourceName, "name", datasourceByID, "name"),
					tftest.TestCheckResourceAttrPair(resourceName, "description", datasourceByID, "description"),
					tftest.TestCheckResourceAttrPair(resourceName, "labels.A", datasourceByID, "labels.A"),
					tftest.TestCheckResourceAttrPair(resourceName, "dns_servers.0", datasourceByID, "dns_servers.0"),
					tftest.TestCheckResourceAttrPair(resourceName, "dns_servers.1", datasourceByID, "dns_servers.1"),
					tftest.TestCheckResourceAttrPair(resourceName, "ntp_servers.0", datasourceByID, "ntp_servers.0"),
					tftest.TestCheckResourceAttrPair(resourceName, "ntp_servers.1", datasourceByID, "ntp_servers.1"),
					tftest.TestCheckResourceAttrPair(resourceName, "domain_search.0", datasourceByID, "domain_search.0"),
					tftest.TestCheckResourceAttrPair(resourceName, "domain_search.1", datasourceByID, "domain_search.1"),

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
					tftest.TestCheckResourceAttr(resourceName, "dns_servers.#", "1"),
					tftest.TestCheckResourceAttr(resourceName, "ntp_servers.#", "1"),
					tftest.TestCheckResourceAttr(resourceName, "domain_search.#", "1"),
					tftest.TestCheckResourceAttr(resourceName, "dns_servers.0", "8.8.8.8"),
					tftest.TestCheckResourceAttr(resourceName, "ntp_servers.0", "42.42.42.42"),
					tftest.TestCheckResourceAttr(resourceName, "domain_search.0", "my.domain"),
				),
			},

			// Update resource to delete some dhcp options
			{
				Config: testutils.ParseTestdataConfig("./testdata/003.vpc_update_delete_some_dhcp_options.tf.tmpl", &testDataSpec),
				Check: tftest.ComposeAggregateTestCheckFunc(
					tftest.TestCheckResourceAttr(resourceName, "dns_servers.#", "1"),
					tftest.TestCheckResourceAttr(resourceName, "ntp_servers.#", "0"),
					tftest.TestCheckResourceAttr(resourceName, "domain_search.#", "0"),
					tftest.TestCheckResourceAttr(resourceName, "dns_servers.0", "8.8.8.8"),
				),
			},

			// Update resource to delete all dhcp options
			{
				Config: testutils.ParseTestdataConfig("./testdata/004.vpc_update_delete_all_dhcp_options.tf.tmpl", &testDataSpec),
				Check: tftest.ComposeAggregateTestCheckFunc(
					tftest.TestCheckResourceAttr(resourceName, "dns_servers.#", "0"),
					tftest.TestCheckResourceAttr(resourceName, "ntp_servers.#", "0"),
					tftest.TestCheckResourceAttr(resourceName, "domain_search.#", "0"),
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
					tftest.TestCheckResourceAttr(resourceName, "ipv4_block", "10.20.0.0/24"),
					tftest.TestCheckNoResourceAttr(resourceName, "labels.%"),
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

func Test_Resource_VPC_Route(t *testing.T) {
	t.Parallel()

	vpcResourceName := "exoscale_vpc.test_vpc"
	subnetResourceName := "exoscale_vpc_subnet.test_subnet"
	resourceName := "exoscale_vpc_route.test_route"

	testDataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}

	tftest.Test(t, tftest.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []tftest.TestStep{
			// Create VPC, Subnet and route
			{
				Config: testutils.ParseTestdataConfig("./testdata/005.route_create.tf.tmpl", &testDataSpec),
				Check: tftest.ComposeAggregateTestCheckFunc(
					tftest.TestCheckResourceAttr(resourceName, "destination", "10.99.0.0/24"),
					tftest.TestCheckResourceAttr(resourceName, "target", "ip=10.21.0.5"),
					tftest.TestCheckResourceAttr(resourceName, "description", "test route"),
				),
			},

			// Import
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					vpcID := s.RootModule().Resources[vpcResourceName].Primary.ID
					subnetID := s.RootModule().Resources[subnetResourceName].Primary.ID
					routeID := s.RootModule().Resources[resourceName].Primary.ID
					return fmt.Sprintf("%s@%s@%s@%s", vpcID, subnetID, routeID, testDataSpec.Zone), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
