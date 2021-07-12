package exoscale

import (
	"context"
	"errors"
	"fmt"
	"testing"

	exov2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

var (
	testAccResourceNLBInstancePoolName       = acctest.RandomWithPrefix(testPrefix)
	testAccResourceNLBInstancePoolTemplateID = testInstanceTemplateID

	testAccResourceNLBName        = acctest.RandomWithPrefix(testPrefix)
	testAccResourceNLBNameUpdated = testAccResourceNLBName + "-updated"
	testAccResourceNLBDescription = acctest.RandString(10)

	testAccResourceNLBConfigCreate = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_instance_pool" "test" {
  zone = local.zone
  name = "%s"
  template_id = "%s"
  service_offering = "medium"
  size = 1
  disk_size = 10

  timeouts {
	delete = "10m"
  }
}

resource "exoscale_nlb" "test" {
  name = "%s"
  description = "%s"
  zone = local.zone

  timeouts {
    delete = "10m"
  }
}

resource "exoscale_nlb_service" "test" {
  zone = local.zone
  name = "%s"
  nlb_id = exoscale_nlb.test.id
  instance_pool_id = exoscale_instance_pool.test.id
  protocol = "tcp"
  port = 80
  target_port = 80
  strategy = "round-robin"

  healthcheck {
    mode = "http"
	port = 80
	interval = 5
	timeout = 3
	retries = 1
	uri = "/healthz"
  }

  timeouts {
	delete = "10m"
  }
}
`,
		testZoneName,
		testAccResourceNLBInstancePoolName,
		testAccResourceNLBInstancePoolTemplateID,
		testAccResourceNLBName,
		testAccResourceNLBDescription,
		testAccResourceNLBName,
	)

	testAccResourceNLBConfigUpdate = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_instance_pool" "test" {
  zone = local.zone
  name = "%s"
  template_id = "%s"
  service_offering = "medium"
  size = 1
  disk_size = 10

  timeouts {
	delete = "10m"
  }
}

resource "exoscale_nlb" "test" {
  name = "%s"
  description = ""
  zone = local.zone

  timeouts {
    delete = "10m"
  }
}

resource "exoscale_nlb_service" "test" {
  zone = local.zone
  name = "%s"
  nlb_id = exoscale_nlb.test.id
  instance_pool_id = exoscale_instance_pool.test.id
  protocol = "tcp"
  port = 80
  target_port = 80
  strategy = "round-robin"

  healthcheck {
	mode = "http"
	port = 80
	interval = 5
	timeout = 3
	retries = 1
	uri = "/healthz"
  }

  timeouts {
	delete = "10m"
  }
}
`,
		testZoneName,
		testAccResourceNLBInstancePoolName,
		testAccResourceNLBInstancePoolTemplateID,
		testAccResourceNLBNameUpdated,
		testAccResourceNLBName,
	)
)

func TestAccResourceNLB(t *testing.T) {
	var (
		r   = "exoscale_nlb.test"
		nlb exov2.NetworkLoadBalancer
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
						a.Equal(testAccResourceNLBName, *nlb.Name)
						a.Len(nlb.Services, 1)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resNLBAttrCreatedAt:   validation.ToDiagFunc(validation.NoZeroValues),
						resNLBAttrDescription: ValidateString(testAccResourceNLBDescription),
						resNLBAttrIPAddress:   validation.ToDiagFunc(validation.IsIPv4Address),
						resNLBAttrName:        ValidateString(testAccResourceNLBName),
						resNLBAttrState:       validation.ToDiagFunc(validation.NoZeroValues),
						resNLBAttrZone:        ValidateString(testZoneName),

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

						a.Empty(defaultString(nlb.Description, ""))
						a.Equal(testAccResourceNLBNameUpdated, *nlb.Name)
						a.Len(nlb.Services, 1)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resNLBAttrCreatedAt:       validation.ToDiagFunc(validation.NoZeroValues),
						resNLBAttrDescription:     validation.ToDiagFunc(validation.StringIsEmpty),
						resNLBAttrIPAddress:       validation.ToDiagFunc(validation.IsIPv4Address),
						resNLBAttrName:            ValidateString(testAccResourceNLBNameUpdated),
						resNLBAttrServices + ".#": ValidateString("1"),
						resNLBAttrState:           validation.ToDiagFunc(validation.NoZeroValues),
						resNLBAttrZone:            ValidateString(testZoneName),
					})),
				),
			},
			{
				// Import
				ResourceName: r,
				ImportStateIdFunc: func(nlb *exov2.NetworkLoadBalancer) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", *nlb.ID, testZoneName), nil
					}
				}(&nlb),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resNLBAttrCreatedAt:       validation.ToDiagFunc(validation.NoZeroValues),
							resNLBAttrDescription:     validation.ToDiagFunc(validation.StringIsEmpty),
							resNLBAttrIPAddress:       validation.ToDiagFunc(validation.IsIPv4Address),
							resNLBAttrName:            ValidateString(testAccResourceNLBNameUpdated),
							resNLBAttrServices + ".#": ValidateString("1"),
							resNLBAttrState:           validation.ToDiagFunc(validation.NoZeroValues),
							resNLBAttrZone:            ValidateString(testZoneName),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceNLBExists(r string, nlb *exov2.NetworkLoadBalancer) resource.TestCheckFunc {
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

func testAccCheckResourceNLBDestroy(nlb *exov2.NetworkLoadBalancer) resource.TestCheckFunc {
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
