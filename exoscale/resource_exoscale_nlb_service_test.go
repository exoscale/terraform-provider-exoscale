package exoscale

import (
	"context"
	"errors"
	"fmt"
	"testing"

	exov2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var (
	testAccResourceNLBServiceZoneName                   = testZoneName
	testAccResourceNLBServiceInstancePoolName           = testPrefix + "-" + testRandomString()
	testAccResourceNLBServiceInstancePoolTemplateID     = testInstanceTemplateID
	testAccResourceNLBServiceNlbName                    = testPrefix + "-" + testRandomString()
	testAccResourceNLBServiceName                       = testPrefix + "-" + testRandomString()
	testAccResourceNLBServiceNameUpdated                = testAccResourceNLBServiceName + "-updated"
	testAccResourceNLBServiceDescription                = testPrefix + "-" + testRandomString()
	testAccResourceNLBServiceDescriptionUpdated         = testAccResourceNLBServiceDescription + "-updated"
	testAccResourceNLBServicePort                       = "80"
	testAccResourceNLBServicePortUpdated                = "443"
	testAccResourceNLBServiceTargetPort                 = "8080"
	testAccResourceNLBServiceTargetPortUpdated          = "8443"
	testAccResourceNLBServiceProtocol                   = "tcp"
	testAccResourceNLBServiceStrategy                   = "round-robin"
	testAccResourceNLBServiceHealthcheckMode            = "tcp"
	testAccResourceNLBServiceHealthcheckModeUpdated     = "https"
	testAccResourceNLBServiceHealthcheckURI             = "/healthz"
	testAccResourceNLBServiceHealthcheckTLSSNI          = "example.net"
	testAccResourceNLBServiceHealthcheckInterval        = "10"
	testAccResourceNLBServiceHealthcheckIntervalUpdated = "5"
	testAccResourceNLBServiceHealthcheckTimeout         = "5"
	testAccResourceNLBServiceHealthcheckTimeoutUpdated  = "3"
	testAccResourceNLBServiceHealthcheckRetries         = "1"
	testAccResourceNLBServiceHealthcheckRetriesUpdated  = "2"
	testAccResourceNLBServiceHealthcheckPort            = "8080"
	testAccResourceNLBServiceHealthcheckPortUpdated     = "8443"

	testAccResourceNLBServiceConfigCreate = fmt.Sprintf(`
variable "zone" {
  default = "%s"
}

resource "exoscale_instance_pool" "pool" {
  zone = var.zone
  name = "%s"
  template_id = "%s"
  service_offering = "medium"
  size = 1
  disk_size = 10

  timeouts {
    delete = "10m"
  }
}

resource "exoscale_nlb" "nlb" {
  name = "%s"
  zone = var.zone

  timeouts {
    delete = "10m"
  }
}

resource "exoscale_nlb_service" "service" {
  zone = var.zone
  name = "%s"
  description = "%s"
  nlb_id = exoscale_nlb.nlb.id
  instance_pool_id = exoscale_instance_pool.pool.id
  protocol = "%s"
  port = %s
  target_port = %s
  strategy = "%s"

  healthcheck {
    mode = "%s"
	port = %s
	interval = %s
	timeout = %s
	retries = %s
  }

  timeouts {
    delete = "10m"
  }
}
`,
		testAccResourceNLBServiceZoneName,
		testAccResourceNLBServiceInstancePoolName,
		testAccResourceNLBServiceInstancePoolTemplateID,
		testAccResourceNLBServiceNlbName,
		testAccResourceNLBServiceName,
		testAccResourceNLBServiceDescription,
		testAccResourceNLBServiceProtocol,
		testAccResourceNLBServicePort,
		testAccResourceNLBServiceTargetPort,
		testAccResourceNLBServiceStrategy,
		testAccResourceNLBServiceHealthcheckMode,
		testAccResourceNLBServiceHealthcheckPort,
		testAccResourceNLBServiceHealthcheckInterval,
		testAccResourceNLBServiceHealthcheckTimeout,
		testAccResourceNLBServiceHealthcheckRetries,
	)

	testAccResourceNLBServiceConfigUpdate = fmt.Sprintf(`
variable "zone" {
  default = "%s"
}

resource "exoscale_instance_pool" "pool" {
  zone = var.zone
  name = "%s"
  template_id = "%s"
  service_offering = "medium"
  size = 2
  disk_size = 10

  timeouts {
    delete = "10m"
  }
}

resource "exoscale_nlb" "nlb" {
  name = "%s"
  zone = var.zone

  timeouts {
    delete = "10m"
  }
}

resource "exoscale_nlb_service" "service" {
  zone = var.zone
  name = "%s"
  description = "%s"
  nlb_id = exoscale_nlb.nlb.id
  instance_pool_id = exoscale_instance_pool.pool.id
  protocol = "%s"
  port = %s
  target_port = %s
  strategy = "%s"

  healthcheck {
    mode = "%s"
    port = %s
    uri = "%s"
    tls_sni = "%s"
    interval = %s
    timeout = %s
    retries = %s
  }

  timeouts {
    delete = "10m"
  }
}
	  `,
		testAccResourceNLBServiceZoneName,
		testAccResourceNLBServiceInstancePoolName,
		testAccResourceNLBServiceInstancePoolTemplateID,
		testAccResourceNLBServiceNlbName,
		testAccResourceNLBServiceNameUpdated,
		testAccResourceNLBServiceDescriptionUpdated,
		testAccResourceNLBServiceProtocol,
		testAccResourceNLBServicePortUpdated,
		testAccResourceNLBServiceTargetPortUpdated,
		testAccResourceNLBServiceStrategy,
		testAccResourceNLBServiceHealthcheckModeUpdated,
		testAccResourceNLBServiceHealthcheckPortUpdated,
		testAccResourceNLBServiceHealthcheckURI,
		testAccResourceNLBServiceHealthcheckTLSSNI,
		testAccResourceNLBServiceHealthcheckIntervalUpdated,
		testAccResourceNLBServiceHealthcheckTimeoutUpdated,
		testAccResourceNLBServiceHealthcheckRetriesUpdated,
	)
)

func TestAccResourceNLBService(t *testing.T) {
	service := new(exov2.NetworkLoadBalancerService)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceNLBServiceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceNLBServiceConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceNLBServiceExists("exoscale_nlb_service.service", service),
					testAccCheckResourceNLBService(service),
					testAccCheckResourceNLBServiceAttributes(testAttrs{
						"name":                            ValidateString(testAccResourceNLBServiceName),
						"description":                     ValidateString(testAccResourceNLBServiceDescription),
						"protocol":                        ValidateString(testAccResourceNLBServiceProtocol),
						"port":                            ValidateString(testAccResourceNLBServicePort),
						"target_port":                     ValidateString(testAccResourceNLBServiceTargetPort),
						"strategy":                        ValidateString(testAccResourceNLBServiceStrategy),
						"healthcheck.#":                   ValidateString("1"),
						"healthcheck.1514854563.mode":     ValidateString(testAccResourceNLBServiceHealthcheckMode),
						"healthcheck.1514854563.port":     ValidateString(testAccResourceNLBServiceHealthcheckPort),
						"healthcheck.1514854563.interval": ValidateString(testAccResourceNLBServiceHealthcheckInterval),
						"healthcheck.1514854563.timeout":  ValidateString(testAccResourceNLBServiceHealthcheckTimeout),
						"healthcheck.1514854563.retries":  ValidateString(testAccResourceNLBServiceHealthcheckRetries),
					}),
				),
			},
			{
				Config: testAccResourceNLBServiceConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceNLBServiceExists("exoscale_nlb_service.service", service),
					testAccCheckResourceNLBService(service),
					testAccCheckResourceNLBServiceAttributes(testAttrs{
						"name":                            ValidateString(testAccResourceNLBServiceNameUpdated),
						"description":                     ValidateString(testAccResourceNLBServiceDescriptionUpdated),
						"protocol":                        ValidateString(testAccResourceNLBServiceProtocol),
						"port":                            ValidateString(testAccResourceNLBServicePortUpdated),
						"target_port":                     ValidateString(testAccResourceNLBServiceTargetPortUpdated),
						"strategy":                        ValidateString(testAccResourceNLBServiceStrategy),
						"healthcheck.#":                   ValidateString("1"),
						"healthcheck.4214407825.mode":     ValidateString(testAccResourceNLBServiceHealthcheckModeUpdated),
						"healthcheck.4214407825.port":     ValidateString(testAccResourceNLBServiceHealthcheckPortUpdated),
						"healthcheck.4214407825.uri":      ValidateString(testAccResourceNLBServiceHealthcheckURI),
						"healthcheck.4214407825.tls_sni":  ValidateString(testAccResourceNLBServiceHealthcheckTLSSNI),
						"healthcheck.4214407825.interval": ValidateString(testAccResourceNLBServiceHealthcheckIntervalUpdated),
						"healthcheck.4214407825.timeout":  ValidateString(testAccResourceNLBServiceHealthcheckTimeoutUpdated),
						"healthcheck.4214407825.retries":  ValidateString(testAccResourceNLBServiceHealthcheckRetriesUpdated),
					}),
				),
			},
			{
				ResourceName:            "exoscale_nlb_service.service",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"state"},
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							"name":                            ValidateString(testAccResourceNLBServiceNameUpdated),
							"description":                     ValidateString(testAccResourceNLBServiceDescriptionUpdated),
							"protocol":                        ValidateString(testAccResourceNLBServiceProtocol),
							"port":                            ValidateString(testAccResourceNLBServicePortUpdated),
							"target_port":                     ValidateString(testAccResourceNLBServiceTargetPortUpdated),
							"strategy":                        ValidateString(testAccResourceNLBServiceStrategy),
							"healthcheck.#":                   ValidateString("1"),
							"healthcheck.4214407825.mode":     ValidateString(testAccResourceNLBServiceHealthcheckModeUpdated),
							"healthcheck.4214407825.port":     ValidateString(testAccResourceNLBServiceHealthcheckPortUpdated),
							"healthcheck.4214407825.uri":      ValidateString(testAccResourceNLBServiceHealthcheckURI),
							"healthcheck.4214407825.tls_sni":  ValidateString(testAccResourceNLBServiceHealthcheckTLSSNI),
							"healthcheck.4214407825.interval": ValidateString(testAccResourceNLBServiceHealthcheckIntervalUpdated),
							"healthcheck.4214407825.timeout":  ValidateString(testAccResourceNLBServiceHealthcheckTimeoutUpdated),
							"healthcheck.4214407825.retries":  ValidateString(testAccResourceNLBServiceHealthcheckRetriesUpdated),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceNLBServiceExists(n string, service *exov2.NetworkLoadBalancerService) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		nlbID, ok := rs.Primary.Attributes["nlb_id"]
		if !ok {
			return errors.New("resource nlb_id not set")
		}

		client := GetComputeClient(testAccProvider.Meta())

		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testAccResourceNLBServiceZoneName),
		)
		r, err := client.GetNetworkLoadBalancer(ctx, testAccResourceNLBServiceZoneName, nlbID)
		if err != nil {
			return err
		}

		for _, s := range r.Services {
			if s.ID == rs.Primary.ID {
				return Copy(service, s)
			}
		}

		return fmt.Errorf("resource Network Load Balancer Service %q not found", rs.Primary.ID)
	}
}

func testAccCheckResourceNLBService(service *exov2.NetworkLoadBalancerService) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if service.ID == "" {
			return errors.New("Network Load Balancer Service ID is empty")
		}

		return nil
	}
}

func testAccCheckResourceNLBServiceAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_nlb_service" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("resource not found in the state")
	}
}

func testAccCheckResourceNLBServiceDestroy(s *terraform.State) error {
	client := GetComputeClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_nlb_service" {
			continue
		}

		nlbID, ok := rs.Primary.Attributes["nlb_id"]
		if !ok {
			return errors.New("resource nlb_id not set")
		}

		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testAccResourceNLBServiceZoneName),
		)
		nlb, err := client.GetNetworkLoadBalancer(
			ctx,
			testAccResourceNLBZoneName,
			nlbID,
		)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		for _, s := range nlb.Services {
			if s.ID == rs.Primary.ID {
				return errors.New("Network Load Balancer Service still exists")
			}
		}
	}

	return nil
}
