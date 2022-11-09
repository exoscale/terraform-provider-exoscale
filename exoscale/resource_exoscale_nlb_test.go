package exoscale

import (
	"context"
	"errors"
	"fmt"
	"testing"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

var (
	testAccResourceNLBInstancePoolName  = acctest.RandomWithPrefix(testPrefix)
	testAccResourceNLBLabelValue        = acctest.RandomWithPrefix(testPrefix)
	testAccResourceNLBLabelValueUpdated = testAccResourceNLBLabelValue + "-updated"
	testAccResourceNLBTemplateName      = testInstanceTemplateName

	testAccResourceNLBName               = acctest.RandomWithPrefix(testPrefix)
	testAccResourceNLBNameUpdated        = testAccResourceNLBName + "-updated"
	testAccResourceNLBDescription        = acctest.RandString(10)
	testAccResourceNLBDescriptionUpdated = testAccResourceNLBDescription + "-updated"

	testAccResourceNLBConfigCreate = fmt.Sprintf(`
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
  service_offering = "medium"
  size             = 1
  disk_size        = 10

  timeouts {
	delete = "10m"
  }
}

resource "exoscale_nlb" "test" {
  name        = "%s"
  description = "%s"
  zone        = local.zone

  labels = {
  	test = "%s"
  }

  timeouts {
    delete = "10m"
  }
}

resource "exoscale_nlb_service" "test" {
  zone             = local.zone
  name             = "%s"
  nlb_id           = exoscale_nlb.test.id
  instance_pool_id = exoscale_instance_pool.test.id
  protocol         = "tcp"
  port             = 80
  target_port      = 80
  strategy         = "round-robin"

  healthcheck {
    mode     = "http"
    port     = 80
    interval = 5
    timeout  = 3
    retries  = 1
    uri      = "/healthz"
  }

  timeouts {
    delete = "10m"
  }
}
`,
		testZoneName,
		testAccResourceNLBTemplateName,
		testAccResourceNLBInstancePoolName,
		testAccResourceNLBName,
		testAccResourceNLBDescription,
		testAccResourceNLBLabelValue,
		testAccResourceNLBName,
	)

	testAccResourceNLBConfigUpdate = fmt.Sprintf(`
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
  service_offering = "medium"
  size             = 1
  disk_size        = 10

  timeouts {
    delete = "10m"
  }
}

resource "exoscale_nlb" "test" {
  name        = "%s"
  description = "%s"
  zone        = local.zone

  labels = {
    test = "%s"
  }

  timeouts {
    delete = "10m"
  }
}

resource "exoscale_nlb_service" "test" {
  zone             = local.zone
  name             = "%s"
  nlb_id           = exoscale_nlb.test.id
  instance_pool_id = exoscale_instance_pool.test.id
  protocol         = "tcp"
  port             = 80
  target_port      = 80
  strategy         = "round-robin"

  healthcheck {
    mode     = "http"
    port     = 80
    interval = 5
    timeout  = 3
    retries  = 1
    uri      = "/healthz"
  }

  timeouts {
    delete = "10m"
  }
}
`,
		testZoneName,
		testAccResourceNLBTemplateName,
		testAccResourceNLBInstancePoolName,
		testAccResourceNLBNameUpdated,
		testAccResourceNLBDescriptionUpdated,
		testAccResourceNLBLabelValueUpdated,
		testAccResourceNLBName,
	)
)

func TestAccResourceNLB(t *testing.T) {
	var (
		r   = "exoscale_nlb.test"
		nlb egoscale.NetworkLoadBalancer
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceNLBDestroy(&nlb),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceNLBConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceNLBExists(r, &nlb),
					func(s *terraform.State) error {
						a := require.New(t)

						a.Equal(testAccResourceNLBDescription, *nlb.Description)
						a.Equal(testAccResourceNLBLabelValue, (*nlb.Labels)["test"])
						a.Equal(testAccResourceNLBName, *nlb.Name)
						a.Len(nlb.Services, 1)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resNLBAttrCreatedAt:        validation.ToDiagFunc(validation.NoZeroValues),
						resNLBAttrDescription:      validateString(testAccResourceNLBDescription),
						resNLBAttrIPAddress:        validation.ToDiagFunc(validation.IsIPv4Address),
						resNLBAttrLabels + ".test": validateString(testAccResourceNLBLabelValue),
						resNLBAttrName:             validateString(testAccResourceNLBName),
						resNLBAttrState:            validation.ToDiagFunc(validation.NoZeroValues),
						resNLBAttrZone:             validateString(testZoneName),

						// Note: can't test the resNLBAttrServices attribute yet, as the
						// exoscale_nlb_service resource is created after the exoscale_nlb
						// being tested here: the return of resourceNLBRead() doesn't include
						// the up-to-date list of services.
					})),
				),
			},
			{
				// Update
				Config: testAccResourceNLBConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceNLBExists(r, &nlb),
					func(s *terraform.State) error {
						a := require.New(t)

						a.Equal(testAccResourceNLBDescriptionUpdated, *nlb.Description)
						a.Equal(testAccResourceNLBLabelValueUpdated, (*nlb.Labels)["test"])
						a.Equal(testAccResourceNLBNameUpdated, *nlb.Name)
						a.Len(nlb.Services, 1)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resNLBAttrCreatedAt:        validation.ToDiagFunc(validation.NoZeroValues),
						resNLBAttrDescription:      validateString(testAccResourceNLBDescriptionUpdated),
						resNLBAttrIPAddress:        validation.ToDiagFunc(validation.IsIPv4Address),
						resNLBAttrLabels + ".test": validateString(testAccResourceNLBLabelValueUpdated),
						resNLBAttrName:             validateString(testAccResourceNLBNameUpdated),
						resNLBAttrServices + ".#":  validateString("1"),
						resNLBAttrState:            validation.ToDiagFunc(validation.NoZeroValues),
						resNLBAttrZone:             validateString(testZoneName),
					})),
				),
			},
			{
				// Import
				ResourceName: r,
				ImportStateIdFunc: func(nlb *egoscale.NetworkLoadBalancer) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", *nlb.ID, testZoneName), nil
					}
				}(&nlb),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resNLBAttrCreatedAt:        validation.ToDiagFunc(validation.NoZeroValues),
							resNLBAttrDescription:      validateString(testAccResourceNLBDescriptionUpdated),
							resNLBAttrIPAddress:        validation.ToDiagFunc(validation.IsIPv4Address),
							resNLBAttrLabels + ".test": validateString(testAccResourceNLBLabelValueUpdated),
							resNLBAttrName:             validateString(testAccResourceNLBNameUpdated),
							resNLBAttrServices + ".#":  validateString("1"),
							resNLBAttrState:            validation.ToDiagFunc(validation.NoZeroValues),
							resNLBAttrZone:             validateString(testZoneName),
						},
						func(s []*terraform.InstanceState) map[string]string {
							for _, state := range s {
								if state.ID == *nlb.ID {
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

func testAccCheckResourceNLBExists(r string, nlb *egoscale.NetworkLoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testZoneName),
		)

		res, err := client.Client.GetNetworkLoadBalancer(ctx, testZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*nlb = *res
		return nil
	}
}

func testAccCheckResourceNLBDestroy(nlb *egoscale.NetworkLoadBalancer) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testZoneName),
		)

		_, err := client.GetNetworkLoadBalancer(ctx, testZoneName, *nlb.ID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("Network Load Balancer still exists")
	}
}
