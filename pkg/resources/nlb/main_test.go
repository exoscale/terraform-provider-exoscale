package nlb_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func TestNLB(t *testing.T) {
	t.Parallel()

	var (
		nlbResource       = "exoscale_nlb.test_nlb"
		serviceResource   = "exoscale_nlb_service.test_service"
		nlbByID           = "data.exoscale_nlb.test_nlb_by_id"
		nlbByName         = "data.exoscale_nlb.test_nlb_by_name"
		serviceListByID   = "data.exoscale_nlb_service_list.test_service_list_by_id"
		serviceListByName = "data.exoscale_nlb_service_list.test_service_list_by_name"
	)

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}

	resourceName := testutils.ResourceName(testdataSpec.ID)
	updatedName := fmt.Sprintf("terraform-provider-test-updated-%d", testdataSpec.ID)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1 Create the NLB, one service, and read them back through both
			// data sources (by id and by name) plus the service list.
			{
				Config: testutils.ParseTestdataConfig("./testdata/001.nlb_create.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(nlbResource, "name", resourceName),
					resource.TestCheckResourceAttr(nlbResource, "description", "description-test"),
					resource.TestCheckResourceAttr(nlbResource, "labels.%", "1"),
					resource.TestCheckResourceAttr(nlbResource, "labels.A", "B"),
					resource.TestCheckResourceAttr(nlbResource, "state", "running"),
					resource.TestCheckResourceAttrSet(nlbResource, "ip_address"),
					resource.TestCheckResourceAttrSet(nlbResource, "created_at"),
					// `services` is intentionally not asserted here: the service
					// is created after the NLB within the same apply, so the NLB
					// state still predates it. It is checked in step 2, once a
					// refresh has happened.

					resource.TestCheckResourceAttr(serviceResource, "name", resourceName),
					resource.TestCheckResourceAttr(serviceResource, "description", "service-description-test"),
					resource.TestCheckResourceAttr(serviceResource, "port", "80"),
					resource.TestCheckResourceAttr(serviceResource, "target_port", "8080"),
					// Defaulted because the configuration omits them.
					resource.TestCheckResourceAttr(serviceResource, "protocol", "tcp"),
					resource.TestCheckResourceAttr(serviceResource, "strategy", "round-robin"),
					resource.TestCheckResourceAttr(serviceResource, "healthcheck.#", "1"),
					resource.TestCheckResourceAttr(serviceResource, "healthcheck.0.mode", "https"),
					resource.TestCheckResourceAttr(serviceResource, "healthcheck.0.port", "8080"),
					resource.TestCheckResourceAttr(serviceResource, "healthcheck.0.uri", "/health"),
					resource.TestCheckResourceAttr(serviceResource, "healthcheck.0.tls_sni", "example.net"),

					resource.TestCheckResourceAttrPair(nlbResource, "name", nlbByID, "name"),
					resource.TestCheckResourceAttrPair(nlbResource, "description", nlbByID, "description"),
					resource.TestCheckResourceAttrPair(nlbResource, "ip_address", nlbByID, "ip_address"),
					resource.TestCheckResourceAttrPair(nlbResource, "labels.A", nlbByID, "labels.A"),
					resource.TestCheckResourceAttrPair(nlbResource, "name", nlbByName, "name"),
					resource.TestCheckResourceAttrPair(nlbResource, "id", nlbByName, "id"),

					// Service list data source, matched by nlb_id and by nlb_name.
					resource.TestCheckResourceAttr(serviceListByID, "services.#", "1"),
					resource.TestCheckResourceAttrPair(serviceResource, "id", serviceListByID, "services.0.id"),
					resource.TestCheckResourceAttr(serviceListByID, "services.0.port", "80"),
					resource.TestCheckResourceAttr(serviceListByID, "services.0.target_port", "8080"),
					resource.TestCheckResourceAttr(serviceListByID, "services.0.healthcheck.port", "8080"),
					resource.TestCheckResourceAttrPair(serviceResource, "name", serviceListByID, "services.0.name"),

					resource.TestCheckResourceAttr(serviceListByName, "services.#", "1"),
					resource.TestCheckResourceAttrPair(serviceResource, "id", serviceListByName, "services.0.id"),
					resource.TestCheckResourceAttrPair(nlbResource, "id", serviceListByName, "nlb_id"),
				),
			},

			// 2 Update both the NLB and the service. The service switches its
			// healthcheck away from https while dropping uri/tls_sni, which
			// must clear them rather than leave the previous values in place.
			{
				Config: testutils.ParseTestdataConfig("./testdata/002.nlb_update.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(nlbResource, "name", updatedName),
					resource.TestCheckResourceAttr(nlbResource, "description", "description-test-updated"),
					resource.TestCheckResourceAttr(nlbResource, "labels.%", "1"),
					resource.TestCheckResourceAttr(nlbResource, "labels.A", "C"),
					// By now the NLB has been refreshed and knows its service.
					resource.TestCheckResourceAttr(nlbResource, "services.#", "1"),
					resource.TestCheckResourceAttrPair(serviceResource, "id", nlbResource, "services.0"),

					resource.TestCheckResourceAttr(serviceResource, "name", updatedName),
					resource.TestCheckResourceAttr(serviceResource, "description", "service-description-test-updated"),
					resource.TestCheckResourceAttr(serviceResource, "port", "443"),
					resource.TestCheckResourceAttr(serviceResource, "target_port", "8443"),
					resource.TestCheckResourceAttr(serviceResource, "protocol", "udp"),
					resource.TestCheckResourceAttr(serviceResource, "strategy", "source-hash"),
					resource.TestCheckResourceAttr(serviceResource, "healthcheck.0.mode", "tcp"),
					resource.TestCheckResourceAttr(serviceResource, "healthcheck.0.port", "8443"),
					resource.TestCheckResourceAttr(serviceResource, "healthcheck.0.interval", "5"),
					resource.TestCheckResourceAttr(serviceResource, "healthcheck.0.timeout", "3"),
					resource.TestCheckResourceAttr(serviceResource, "healthcheck.0.retries", "2"),
					resource.TestCheckNoResourceAttr(serviceResource, "healthcheck.0.uri"),
					resource.TestCheckNoResourceAttr(serviceResource, "healthcheck.0.tls_sni"),
				),
			},

			// 3 Drop `labels` on the NLB. Every v3 update field is `omitempty`,
			// so clearing it only works through ResetLoadBalancerField.
			// `description` is deliberately kept: egoscale v3 cannot express an
			// empty one, so dropping it would leave the API value in place and
			// surface as drift on the next plan.
			{
				Config: testutils.ParseTestdataConfig("./testdata/003.nlb_clear.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(nlbResource, "labels.%", "0"),

					// Optional healthcheck attributes fall back to their defaults.
					resource.TestCheckResourceAttr(serviceResource, "healthcheck.0.mode", "tcp"),
					resource.TestCheckResourceAttr(serviceResource, "healthcheck.0.interval", "10"),
					resource.TestCheckResourceAttr(serviceResource, "healthcheck.0.timeout", "5"),
					resource.TestCheckResourceAttr(serviceResource, "healthcheck.0.retries", "1"),
				),
			},

			// 4 Import the NLB: <ID>@<ZONE>.
			{
				ResourceName: nlbResource,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf(
						"%s@%s",
						s.RootModule().Resources[nlbResource].Primary.ID,
						testdataSpec.Zone,
					), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},

			// 5 Import the service: <NLB-ID>/<SERVICE-ID>@<ZONE>.
			{
				ResourceName: serviceResource,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf(
						"%s/%s@%s",
						s.RootModule().Resources[nlbResource].Primary.ID,
						s.RootModule().Resources[serviceResource].Primary.ID,
						testdataSpec.Zone,
					), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
