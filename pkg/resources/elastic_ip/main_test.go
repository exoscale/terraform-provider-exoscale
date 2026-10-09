package elasticip_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	tftest "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestElasticIP(t *testing.T) {
	t.Parallel()

	resource4 := "exoscale_elastic_ip.test4"
	resource6 := "exoscale_elastic_ip.test6"
	datasourceByID := "data.exoscale_elastic_ip.by_id"
	datasourceByIPAddress := "data.exoscale_elastic_ip.by_ip_address"
	datasourceByLabels := "data.exoscale_elastic_ip.by_labels"

	testDataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}
	name := testutils.ResourceName(testDataSpec.ID)

	importStateIDFunc := func(r string) tftest.ImportStateIdFunc {
		return func(s *terraform.State) (string, error) {
			return fmt.Sprintf("%s@%s", s.RootModule().Resources[r].Primary.ID, testDataSpec.Zone), nil
		}
	}

	tftest.Test(t, tftest.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []tftest.TestStep{
			// Create a managed EIPv4, an unmanaged EIPv6 and the data sources
			{
				Config: testutils.ParseTestdataConfig("./testdata/001.eip_create.tf.tmpl", &testDataSpec),
				Check: tftest.ComposeAggregateTestCheckFunc(
					tftest.TestCheckResourceAttrSet(resource4, "id"),
					tftest.TestCheckResourceAttrSet(resource4, "ip_address"),
					tftest.TestCheckResourceAttrSet(resource4, "cidr"),
					tftest.TestCheckResourceAttr(resource4, "address_family", "inet4"),
					tftest.TestCheckResourceAttr(resource4, "description", name),
					tftest.TestCheckResourceAttr(resource4, "healthcheck.mode", "http"),
					tftest.TestCheckResourceAttr(resource4, "healthcheck.port", "80"),
					tftest.TestCheckResourceAttr(resource4, "healthcheck.uri", "/health"),
					tftest.TestCheckResourceAttr(resource4, "healthcheck.interval", "5"),
					tftest.TestCheckResourceAttr(resource4, "healthcheck.timeout", "3"),
					tftest.TestCheckResourceAttr(resource4, "healthcheck.strikes_ok", "2"),
					tftest.TestCheckResourceAttr(resource4, "healthcheck.strikes_fail", "1"),
					tftest.TestCheckResourceAttr(resource4, "healthcheck.tls_sni", ""),
					tftest.TestCheckResourceAttr(resource4, "healthcheck.tls_skip_verify", "false"),
					tftest.TestCheckResourceAttr(resource4, "labels.test", name),
					tftest.TestCheckNoResourceAttr(resource4, "reverse_dns"),

					tftest.TestCheckResourceAttrSet(resource6, "ip_address"),
					tftest.TestCheckResourceAttr(resource6, "address_family", "inet6"),
					tftest.TestCheckResourceAttr(resource6, "description", name),
					tftest.TestCheckResourceAttr(resource6, "reverse_dns", "tf-provider-test.exoscale.com"),
					tftest.TestCheckNoResourceAttr(resource6, "healthcheck.mode"),
					tftest.TestCheckResourceAttr(resource6, "labels.test", name+"-6"),

					tftest.TestCheckResourceAttrPair(resource4, "id", datasourceByID, "id"),
					tftest.TestCheckResourceAttrPair(resource4, "ip_address", datasourceByID, "ip_address"),
					tftest.TestCheckResourceAttrPair(resource4, "cidr", datasourceByID, "cidr"),
					tftest.TestCheckResourceAttrPair(resource4, "address_family", datasourceByID, "address_family"),
					tftest.TestCheckResourceAttrPair(resource4, "description", datasourceByID, "description"),
					tftest.TestCheckResourceAttrPair(resource4, "healthcheck.mode", datasourceByID, "healthcheck.mode"),
					tftest.TestCheckResourceAttrPair(resource4, "healthcheck.port", datasourceByID, "healthcheck.port"),
					tftest.TestCheckResourceAttrPair(resource4, "healthcheck.uri", datasourceByID, "healthcheck.uri"),
					tftest.TestCheckResourceAttrPair(resource4, "healthcheck.interval", datasourceByID, "healthcheck.interval"),
					tftest.TestCheckResourceAttrPair(resource4, "healthcheck.timeout", datasourceByID, "healthcheck.timeout"),
					tftest.TestCheckResourceAttrPair(resource4, "healthcheck.strikes_ok", datasourceByID, "healthcheck.strikes_ok"),
					tftest.TestCheckResourceAttrPair(resource4, "healthcheck.strikes_fail", datasourceByID, "healthcheck.strikes_fail"),
					tftest.TestCheckResourceAttrPair(resource4, "labels.test", datasourceByID, "labels.test"),
					tftest.TestCheckResourceAttr(datasourceByID, "reverse_dns", ""),

					tftest.TestCheckResourceAttrPair(resource6, "id", datasourceByIPAddress, "id"),
					tftest.TestCheckResourceAttrPair(resource6, "address_family", datasourceByIPAddress, "address_family"),
					tftest.TestCheckResourceAttrPair(resource6, "reverse_dns", datasourceByIPAddress, "reverse_dns"),
					tftest.TestCheckResourceAttrPair(resource6, "labels.test", datasourceByIPAddress, "labels.test"),
					tftest.TestCheckNoResourceAttr(datasourceByIPAddress, "healthcheck.mode"),

					tftest.TestCheckResourceAttrPair(resource4, "id", datasourceByLabels, "id"),
					tftest.TestCheckResourceAttrPair(resource4, "ip_address", datasourceByLabels, "ip_address"),
				),
			},

			// Update the healthcheck, set a reverse DNS on the EIPv4, remove
			// the description and the reverse DNS of the EIPv6
			{
				Config: testutils.ParseTestdataConfig("./testdata/002.eip_update.tf.tmpl", &testDataSpec),
				Check: tftest.ComposeAggregateTestCheckFunc(
					tftest.TestCheckResourceAttr(resource4, "description", name+"-updated"),
					tftest.TestCheckResourceAttr(resource4, "reverse_dns", "tf-provider-test.exoscale.com"),
					tftest.TestCheckResourceAttr(resource4, "healthcheck.mode", "https"),
					tftest.TestCheckResourceAttr(resource4, "healthcheck.port", "443"),
					tftest.TestCheckResourceAttr(resource4, "healthcheck.uri", "/health-updated"),
					tftest.TestCheckResourceAttr(resource4, "healthcheck.interval", "6"),
					tftest.TestCheckResourceAttr(resource4, "healthcheck.timeout", "4"),
					tftest.TestCheckResourceAttr(resource4, "healthcheck.strikes_ok", "3"),
					tftest.TestCheckResourceAttr(resource4, "healthcheck.strikes_fail", "2"),
					tftest.TestCheckResourceAttr(resource4, "healthcheck.tls_sni", "example.net"),
					tftest.TestCheckResourceAttr(resource4, "healthcheck.tls_skip_verify", "true"),
					tftest.TestCheckResourceAttr(resource4, "labels.test", name+"-updated"),

					tftest.TestCheckNoResourceAttr(resource6, "description"),
					tftest.TestCheckNoResourceAttr(resource6, "reverse_dns"),
					tftest.TestCheckResourceAttr(resource6, "labels.test", name+"-6-updated"),
				),
			},

			// Import
			{
				ResourceName:            resource4,
				ImportStateIdFunc:       importStateIDFunc(resource4),
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
			{
				ResourceName:            resource6,
				ImportStateIdFunc:       importStateIDFunc(resource6),
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},

			// Removing the healthcheck replaces the managed EIP
			{
				Config: testutils.ParseTestdataConfig("./testdata/003.eip_healthcheck_removed.tf.tmpl", &testDataSpec),
				ConfigPlanChecks: tftest.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(resource4, plancheck.ResourceActionReplace),
						plancheck.ExpectResourceAction(resource6, plancheck.ResourceActionNoop),
					},
				},
				Check: tftest.ComposeAggregateTestCheckFunc(
					tftest.TestCheckNoResourceAttr(resource4, "healthcheck.mode"),
					tftest.TestCheckResourceAttr(resource4, "reverse_dns", "tf-provider-test.exoscale.com"),
					tftest.TestCheckResourceAttr(resource4, "labels.test", name+"-updated"),
				),
			},
		},
	})
}
