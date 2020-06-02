package exoscale

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/exoscale/egoscale"
	apiv2 "github.com/exoscale/egoscale/api/v2"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var (
	testAccResourceNLBServiceZoneName = testZoneName

	testAccResourceNLBServiceInstancePoolName       = testPrefix + "-" + testRandomString()
	testAccResourceNLBServiceInstancePoolTemplateID = testInstanceTemplateID

	testAccResourceNLBServiceNlbName            = testPrefix + "-" + testRandomString()
	testAccResourceNLBServiceName               = testPrefix + "-" + testRandomString()
	testAccResourceNLBServiceNameUpdated        = testAccResourceNLBServiceName + "-updated"
	testAccResourceNLBServiceDescription        = testPrefix + "-" + testRandomString()
	testAccResourceNLBServiceDescriptionUpdated = testAccResourceNLBServiceDescription + "-updated"
	testAccResourceNLBServicePort               = "8080"
	testAccResourceNLBServicePortUpdated        = "80"
	testAccResourceNLBServiceTargetPort         = "8080"
	testAccResourceNLBServiceTargetPortUpdated  = "80"
	testAccResourceNLBServiceProtocol           = "tcp"
	testAccResourceNLBServiceStrategy           = "round-robin"

	testAccResourceNLBServiceHealthcheckMode        = "tcp"
	testAccResourceNLBServiceHealthcheckModeUpdated = "http"
	testAccResourceNLBServiceHealthcheckInterval    = "10"
	testAccResourceNLBServiceHealthcheckTimeout     = "5"
	testAccResourceNLBServiceHealthcheckRetries     = "1"
	testAccResourceNLBServiceHealthcheckPort        = "8080"
	testAccResourceNLBServiceHealthcheckPortUpdated = "80"
	testAccResourceNLBServiceHealthcheckURI         = "/healthz"

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
	uri = ""
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
    interval = %s
    timeout = %s
    retries = %s
    uri = "%s"
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
		testAccResourceNLBServiceHealthcheckInterval,
		testAccResourceNLBServiceHealthcheckTimeout,
		testAccResourceNLBServiceHealthcheckRetries,
		testAccResourceNLBServiceHealthcheckURI,
	)
)

func TestAccResourceNLBService(t *testing.T) {
	service := new(egoscale.NetworkLoadBalancerService)

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
						"target_port":                     ValidateString(testAccResourceNLBServiceTargetPort),
						"port":                            ValidateString(testAccResourceNLBServicePort),
						"strategy":                        ValidateString(testAccResourceNLBServiceStrategy),
						"healthcheck.#":                   ValidateString("1"),
						"healthcheck.1241295229.mode":     ValidateString(testAccResourceNLBServiceHealthcheckMode),
						"healthcheck.1241295229.interval": ValidateString(testAccResourceNLBServiceHealthcheckInterval),
						"healthcheck.1241295229.timeout":  ValidateString(testAccResourceNLBServiceHealthcheckTimeout),
						"healthcheck.1241295229.retries":  ValidateString(testAccResourceNLBServiceHealthcheckRetries),
						"healthcheck.1241295229.port":     ValidateString(testAccResourceNLBServiceHealthcheckPort),
						"healthcheck.1241295229.uri":      ValidateString(""),
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
						"target_port":                     ValidateString(testAccResourceNLBServiceTargetPortUpdated),
						"port":                            ValidateString(testAccResourceNLBServicePortUpdated),
						"strategy":                        ValidateString(testAccResourceNLBServiceStrategy),
						"healthcheck.#":                   ValidateString("1"),
						"healthcheck.1564274940.mode":     ValidateString(testAccResourceNLBServiceHealthcheckModeUpdated),
						"healthcheck.1564274940.interval": ValidateString(testAccResourceNLBServiceHealthcheckInterval),
						"healthcheck.1564274940.timeout":  ValidateString(testAccResourceNLBServiceHealthcheckTimeout),
						"healthcheck.1564274940.retries":  ValidateString(testAccResourceNLBServiceHealthcheckRetries),
						"healthcheck.1564274940.port":     ValidateString(testAccResourceNLBServiceHealthcheckPortUpdated),
						"healthcheck.1564274940.uri":      ValidateString(testAccResourceNLBServiceHealthcheckURI),
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
							"target_port":                     ValidateString(testAccResourceNLBServiceTargetPortUpdated),
							"port":                            ValidateString(testAccResourceNLBServicePortUpdated),
							"strategy":                        ValidateString(testAccResourceNLBServiceStrategy),
							"healthcheck.#":                   ValidateString("1"),
							"healthcheck.1564274940.mode":     ValidateString(testAccResourceNLBServiceHealthcheckModeUpdated),
							"healthcheck.1564274940.interval": ValidateString(testAccResourceNLBServiceHealthcheckInterval),
							"healthcheck.1564274940.timeout":  ValidateString(testAccResourceNLBServiceHealthcheckTimeout),
							"healthcheck.1564274940.retries":  ValidateString(testAccResourceNLBServiceHealthcheckRetries),
							"healthcheck.1564274940.port":     ValidateString(testAccResourceNLBServiceHealthcheckPortUpdated),
							"healthcheck.1564274940.uri":      ValidateString(testAccResourceNLBServiceHealthcheckURI),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceNLBServiceExists(n string, service *egoscale.NetworkLoadBalancerService) resource.TestCheckFunc {
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

		ctx := apiv2.WithEndpoint(
			context.Background(),
			apiv2.NewReqEndpoint(testEnvironment, testAccResourceNLBServiceZoneName),
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

func testAccCheckResourceNLBService(service *egoscale.NetworkLoadBalancerService) resource.TestCheckFunc {
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

		ctx := apiv2.WithEndpoint(
			context.Background(),
			apiv2.NewReqEndpoint(testEnvironment, testAccResourceNLBServiceZoneName),
		)
		nlb, err := client.GetNetworkLoadBalancer(
			ctx,
			testAccResourceNLBZoneName,
			nlbID,
		)
		if err != nil {
			if err == egoscale.ErrNotFound {
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
