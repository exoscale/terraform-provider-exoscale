package exoscale

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
)

var (
	testAccResourceNLBServiceDescription        = acctest.RandomWithPrefix(testPrefix)
	testAccResourceNLBServiceDescriptionUpdated = testAccResourceNLBServiceDescription + "-updated"
	testAccResourceNLBServiceInstancePoolName   = acctest.RandomWithPrefix(testPrefix)
	testAccResourceNLBServiceName               = acctest.RandomWithPrefix(testPrefix)
	testAccResourceNLBServiceNameUpdated        = testAccResourceNLBServiceName + "-updated"
	testAccResourceNLBServiceNLBName            = acctest.RandomWithPrefix(testPrefix)
	testAccResourceNLBServicePort               = "80"
	testAccResourceNLBServicePortUpdated        = "443"
	testAccResourceNLBServiceProtocol           = defaultNLBServiceProtocol
	testAccResourceNLBServiceProtocolUpdated    = "udp"
	testAccResourceNLBServiceStrategy           = defaulNLBServiceStrategy
	testAccResourceNLBServiceStrategyUpdated    = "source-hash"
	testAccResourceNLBServiceTargetPort         = "8080"
	testAccResourceNLBServiceTargetPortUpdated  = "8443"
	testAccResourceNLBServiceTemplateName       = testInstanceTemplateName

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
locals {
  zone = "%s"
}

data "exoscale_compute_template" "template" {
  zone = local.zone
  name = "%s"
}

resource "exoscale_instance_pool" "test" {
  zone             = local.zone
  name             = "%s"
  template_id      = data.exoscale_compute_template.template.id
  service_offering = "small"
  size             = 1
  disk_size        = 10

  timeouts {
    delete = "10m"
  }
}

resource "exoscale_nlb" "test" {
  name = "%s"
  zone = local.zone

  timeouts {
    delete = "10m"
  }
}

resource "exoscale_nlb_service" "test" {
  zone             = local.zone
  name             = "%s"
  description      = "%s"
  nlb_id           = exoscale_nlb.test.id
  instance_pool_id = exoscale_instance_pool.test.id
  protocol         = "%s"
  port             = %s
  target_port      = %s
  strategy         = "%s"

  healthcheck {
    mode     = "%s"
    port     = %s
    interval = %s
    timeout  = %s
    retries  = %s
  }

  timeouts {
    delete = "10m"
  }
}
`,
		testZoneName,
		testAccResourceNLBServiceTemplateName,
		testAccResourceNLBServiceInstancePoolName,
		testAccResourceNLBServiceNLBName,
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
locals {
  zone = "%s"
}

data "exoscale_compute_template" "template" {
  zone = local.zone
  name = "%s"
}

resource "exoscale_instance_pool" "test" {
  zone             = local.zone
  name             = "%s"
  template_id      = data.exoscale_compute_template.template.id
  service_offering = "small"
  size             = 2
  disk_size        = 10

  timeouts {
    delete = "10m"
  }
}

resource "exoscale_nlb" "test" {
  name = "%s"
  zone = local.zone

  timeouts {
    delete = "10m"
  }
}

resource "exoscale_nlb_service" "test" {
  zone             = local.zone
  name             = "%s"
  description      = "%s"
  nlb_id           = exoscale_nlb.test.id
  instance_pool_id = exoscale_instance_pool.test.id
  protocol         = "%s"
  port             = %s
  target_port      = %s
  strategy         = "%s"

  healthcheck {
    mode     = "%s"
    port     = %s
    uri      = "%s"
    tls_sni  = "%s"
    interval = %s
    timeout  = %s
    retries  = %s
  }

  timeouts {
    delete = "10m"
  }
}
	  `,
		testZoneName,
		testAccResourceNLBServiceTemplateName,
		testAccResourceNLBServiceInstancePoolName,
		testAccResourceNLBServiceNLBName,
		testAccResourceNLBServiceNameUpdated,
		testAccResourceNLBServiceDescriptionUpdated,
		testAccResourceNLBServiceProtocolUpdated,
		testAccResourceNLBServicePortUpdated,
		testAccResourceNLBServiceTargetPortUpdated,
		testAccResourceNLBServiceStrategyUpdated,
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
	var (
		r          = "exoscale_nlb_service.test"
		nlb        egoscale.NetworkLoadBalancer
		nlbService egoscale.NetworkLoadBalancerService
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceNLBServiceDestroy(r),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceNLBServiceConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceNLBExists("exoscale_nlb.test", &nlb),
					testAccCheckResourceNLBServiceExists(r, &nlbService),
					func(s *terraform.State) error {
						a := require.New(t)

						instancePoolID, err := attrFromState(s, "exoscale_instance_pool.test", "id")
						a.NoError(err, "unable to retrieve Instance Pool ID from state")

						a.Equal(testAccResourceNLBServiceDescription, *nlbService.Description)
						a.Equal(testAccResourceNLBServiceHealthcheckInterval, fmt.Sprint(int(nlbService.Healthcheck.Interval.Seconds())))
						a.Equal(testAccResourceNLBServiceHealthcheckMode, *nlbService.Healthcheck.Mode)
						a.Equal(testAccResourceNLBServiceHealthcheckPort, fmt.Sprint(*nlbService.Healthcheck.Port))
						a.Equal(testAccResourceNLBServiceHealthcheckRetries, fmt.Sprint(*nlbService.Healthcheck.Retries))
						a.Equal(testAccResourceNLBServiceHealthcheckTimeout, fmt.Sprint(int(nlbService.Healthcheck.Timeout.Seconds())))
						a.Equal(instancePoolID, *nlbService.InstancePoolID)
						a.Equal(testAccResourceNLBServiceName, *nlbService.Name)
						a.Equal(testAccResourceNLBServicePort, fmt.Sprint(*nlbService.Port))
						a.Equal(testAccResourceNLBServiceProtocol, *nlbService.Protocol)
						a.Equal(testAccResourceNLBServiceStrategy, *nlbService.Strategy)
						a.Equal(testAccResourceNLBServiceTargetPort, fmt.Sprint(*nlbService.TargetPort))

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resNLBServiceAttrDescription:                                                validateString(testAccResourceNLBServiceDescription),
						resNLBServiceAttrHealthcheck + ".#":                                         validateString("1"),
						resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckInterval: validateString(testAccResourceNLBServiceHealthcheckInterval),
						resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckMode:     validateString(testAccResourceNLBServiceHealthcheckMode),
						resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckPort:     validateString(testAccResourceNLBServiceHealthcheckPort),
						resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckRetries:  validateString(testAccResourceNLBServiceHealthcheckRetries),
						resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckTimeout:  validateString(testAccResourceNLBServiceHealthcheckTimeout),
						resNLBServiceAttrName:       validateString(testAccResourceNLBServiceName),
						resNLBServiceAttrPort:       validateString(testAccResourceNLBServicePort),
						resNLBServiceAttrProtocol:   validateString(testAccResourceNLBServiceProtocol),
						resNLBServiceAttrState:      validation.ToDiagFunc(validation.NoZeroValues),
						resNLBServiceAttrStrategy:   validateString(testAccResourceNLBServiceStrategy),
						resNLBServiceAttrTargetPort: validateString(testAccResourceNLBServiceTargetPort),
					})),
				),
			},
			{
				// Update
				Config: testAccResourceNLBServiceConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceNLBServiceExists(r, &nlbService),
					func(s *terraform.State) error {
						a := require.New(t)

						instancePoolID, err := attrFromState(s, "exoscale_instance_pool.test", "id")
						a.NoError(err, "unable to retrieve Instance Pool ID from state")

						a.Equal(testAccResourceNLBServiceDescriptionUpdated, *nlbService.Description)
						a.Equal(testAccResourceNLBServiceHealthcheckIntervalUpdated, fmt.Sprint(int(nlbService.Healthcheck.Interval.Seconds())))
						a.Equal(testAccResourceNLBServiceHealthcheckModeUpdated, *nlbService.Healthcheck.Mode)
						a.Equal(testAccResourceNLBServiceHealthcheckPortUpdated, fmt.Sprint(*nlbService.Healthcheck.Port))
						a.Equal(testAccResourceNLBServiceHealthcheckRetriesUpdated, fmt.Sprint(*nlbService.Healthcheck.Retries))
						a.Equal(testAccResourceNLBServiceHealthcheckTLSSNI, *nlbService.Healthcheck.TLSSNI)
						a.Equal(testAccResourceNLBServiceHealthcheckTimeoutUpdated, fmt.Sprint(int(nlbService.Healthcheck.Timeout.Seconds())))
						a.Equal(testAccResourceNLBServiceHealthcheckURI, *nlbService.Healthcheck.URI)
						a.Equal(instancePoolID, *nlbService.InstancePoolID)
						a.Equal(testAccResourceNLBServiceNameUpdated, *nlbService.Name)
						a.Equal(testAccResourceNLBServicePortUpdated, fmt.Sprint(*nlbService.Port))
						a.Equal(testAccResourceNLBServiceProtocolUpdated, *nlbService.Protocol)
						a.Equal(testAccResourceNLBServiceStrategyUpdated, *nlbService.Strategy)
						a.Equal(testAccResourceNLBServiceTargetPortUpdated, fmt.Sprint(*nlbService.TargetPort))

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resNLBServiceAttrDescription:                                                validateString(testAccResourceNLBServiceDescriptionUpdated),
						resNLBServiceAttrHealthcheck + ".#":                                         validateString("1"),
						resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckInterval: validateString(testAccResourceNLBServiceHealthcheckIntervalUpdated),
						resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckMode:     validateString(testAccResourceNLBServiceHealthcheckModeUpdated),
						resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckPort:     validateString(testAccResourceNLBServiceHealthcheckPortUpdated),
						resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckRetries:  validateString(testAccResourceNLBServiceHealthcheckRetriesUpdated),
						resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckTimeout:  validateString(testAccResourceNLBServiceHealthcheckTimeoutUpdated),
						resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckTLSSNI:   validateString(testAccResourceNLBServiceHealthcheckTLSSNI),
						resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckURI:      validateString(testAccResourceNLBServiceHealthcheckURI),
						resNLBServiceAttrName:       validateString(testAccResourceNLBServiceNameUpdated),
						resNLBServiceAttrPort:       validateString(testAccResourceNLBServicePortUpdated),
						resNLBServiceAttrProtocol:   validateString(testAccResourceNLBServiceProtocolUpdated),
						resNLBServiceAttrState:      validation.ToDiagFunc(validation.NoZeroValues),
						resNLBServiceAttrStrategy:   validateString(testAccResourceNLBServiceStrategyUpdated),
						resNLBServiceAttrTargetPort: validateString(testAccResourceNLBServiceTargetPortUpdated),
					})),
				),
			},
			{
				// Import
				ResourceName: r,
				ImportStateIdFunc: func(
					nlb *egoscale.NetworkLoadBalancer,
					nlbService *egoscale.NetworkLoadBalancerService,
				) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s/%s@%s", *nlb.ID, *nlbService.ID, testZoneName), nil
					}
				}(&nlb, &nlbService),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resNLBServiceAttrDescription:                                                validateString(testAccResourceNLBServiceDescriptionUpdated),
							resNLBServiceAttrHealthcheck + ".#":                                         validateString("1"),
							resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckInterval: validateString(testAccResourceNLBServiceHealthcheckIntervalUpdated),
							resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckMode:     validateString(testAccResourceNLBServiceHealthcheckModeUpdated),
							resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckPort:     validateString(testAccResourceNLBServiceHealthcheckPortUpdated),
							resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckRetries:  validateString(testAccResourceNLBServiceHealthcheckRetriesUpdated),
							resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckTimeout:  validateString(testAccResourceNLBServiceHealthcheckTimeoutUpdated),
							resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckTLSSNI:   validateString(testAccResourceNLBServiceHealthcheckTLSSNI),
							resNLBServiceAttrHealthcheck + ".0." + resNLBServiceAttrHealthcheckURI:      validateString(testAccResourceNLBServiceHealthcheckURI),
							resNLBServiceAttrNLBID:      validation.ToDiagFunc(validation.IsUUID),
							resNLBServiceAttrName:       validateString(testAccResourceNLBServiceNameUpdated),
							resNLBServiceAttrPort:       validateString(testAccResourceNLBServicePortUpdated),
							resNLBServiceAttrProtocol:   validateString(testAccResourceNLBServiceProtocolUpdated),
							resNLBServiceAttrState:      validation.ToDiagFunc(validation.NoZeroValues),
							resNLBServiceAttrStrategy:   validateString(testAccResourceNLBServiceStrategyUpdated),
							resNLBServiceAttrTargetPort: validateString(testAccResourceNLBServiceTargetPortUpdated),
						},
						func(s []*terraform.InstanceState) map[string]string {
							for _, state := range s {
								if state.ID == *nlbService.ID {
									return state.Attributes
								}
							}
							return nil
						}(s),
					)
				},
			},
		},
	})
}

func testAccCheckResourceNLBServiceExists(r string, nlbService *egoscale.NetworkLoadBalancerService) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		nlbID, ok := rs.Primary.Attributes[resNLBServiceAttrNLBID]
		if !ok {
			return fmt.Errorf("resource attribute %q not set", resNLBServiceAttrNLBID)
		}

		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testZoneName),
		)

		nlb, err := client.Client.GetNetworkLoadBalancer(ctx, testZoneName, nlbID)
		if err != nil {
			return err
		}

		for _, s := range nlb.Services {
			if *s.ID == rs.Primary.ID {
				*nlbService = *s
				return nil
			}
		}

		return fmt.Errorf("resource Network Load Balancer Service %q not found", rs.Primary.ID)
	}
}

func testAccCheckResourceNLBServiceDestroy(r string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		nlbID, ok := rs.Primary.Attributes[resNLBServiceAttrNLBID]
		if !ok {
			return fmt.Errorf("resource attribute %q not set", resNLBServiceAttrNLBID)
		}

		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testZoneName),
		)

		nlb, err := client.GetNetworkLoadBalancer(ctx, testZoneName, nlbID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		for _, s := range nlb.Services {
			if *s.ID == rs.Primary.ID {
				return errors.New("Network Load Balancer Service still exists")
			}
		}

		return nil
	}
}
