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
)

var (
	testAccResourceNLBZoneName = testZoneName

	testAccResourceNLBInstancePoolName       = acctest.RandomWithPrefix(testPrefix)
	testAccResourceNLBInstancePoolTemplateID = testInstanceTemplateID

	testAccResourceNLBName               = acctest.RandomWithPrefix(testPrefix)
	testAccResourceNLBNameUpdated        = testAccResourceNLBName + "-updated"
	testAccResourceNLBDescription        = testDescription
	testAccResourceNLBDescriptionUpdated = testDescription + "-updated"

	testAccResourceNLBConfigCreate = fmt.Sprintf(`
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
  description = "%s"
  zone = var.zone

  timeouts {
    delete = "10m"
  }
}

resource "exoscale_nlb_service" "service" {
  zone = var.zone
  name = "%s"
  description = "test"
  nlb_id = exoscale_nlb.nlb.id
  instance_pool_id = exoscale_instance_pool.pool.id
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
		testAccResourceNLBZoneName,
		testAccResourceNLBInstancePoolName,
		testAccResourceNLBInstancePoolTemplateID,
		testAccResourceNLBName,
		testAccResourceNLBDescription,
		testAccResourceNLBName,
	)

	testAccResourceNLBConfigUpdate = fmt.Sprintf(`
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
  description = "%s"
  zone = var.zone

  timeouts {
    delete = "10m"
  }
}

resource "exoscale_nlb_service" "service" {
  zone = var.zone
  name = "%s"
  description = "test"
  nlb_id = exoscale_nlb.nlb.id
  instance_pool_id = exoscale_instance_pool.pool.id
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
		testAccResourceNLBZoneName,
		testAccResourceNLBInstancePoolName,
		testAccResourceNLBInstancePoolTemplateID,
		testAccResourceNLBNameUpdated,
		testAccResourceNLBDescriptionUpdated,
		testAccResourceNLBName,
	)
)

func TestAccResourceNLB(t *testing.T) {
	nlb := new(exov2.NetworkLoadBalancer)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceNLBDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceNLBConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceNLBExists("exoscale_nlb.nlb", nlb),
					testAccCheckResourceNLB(nlb),
					testAccCheckResourceNLBAttributes(testAttrs{
						"zone":        ValidateString(testAccResourceNLBZoneName),
						"name":        ValidateString(testAccResourceNLBName),
						"description": ValidateString(testAccResourceNLBDescription),
						"created_at":  validation.ToDiagFunc(validation.NoZeroValues),
						"ip_address":  validation.ToDiagFunc(validation.IsIPv4Address),
						"state":       validation.ToDiagFunc(validation.NoZeroValues),
					}),
				),
			},
			{
				Config: testAccResourceNLBConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceNLBExists("exoscale_nlb.nlb", nlb),
					testAccCheckResourceNLB(nlb),
					testAccCheckResourceNLBAttributes(testAttrs{
						"zone":        ValidateString(testAccResourceNLBZoneName),
						"name":        ValidateString(testAccResourceNLBNameUpdated),
						"description": ValidateString(testAccResourceNLBDescriptionUpdated),
						"created_at":  validation.ToDiagFunc(validation.NoZeroValues),
						"ip_address":  validation.ToDiagFunc(validation.IsIPv4Address),
						"state":       validation.ToDiagFunc(validation.NoZeroValues),
					}),
				),
			},
			{
				ResourceName:            "exoscale_nlb.nlb",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"state"},
				ImportStateCheck: composeImportStateCheckFunc(
					testAccCheckResourceImportedAttributes(
						"exoscale_nlb",
						testAttrs{
							"zone":        ValidateString(testAccResourceNLBZoneName),
							"name":        ValidateString(testAccResourceNLBNameUpdated),
							"description": ValidateString(testAccResourceNLBDescriptionUpdated),
							"created_at":  validation.ToDiagFunc(validation.NoZeroValues),
							"ip_address":  validation.ToDiagFunc(validation.IsIPv4Address),
							"state":       validation.ToDiagFunc(validation.NoZeroValues),
						},
					),
					testAccCheckResourceImportedAttributes(
						"exoscale_nlb_service",
						testAttrs{
							"name": ValidateString(testAccResourceNLBName),
						},
					),
				),
			},
		},
	})
}

func testAccCheckResourceNLBExists(n string, nlb *exov2.NetworkLoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client := GetComputeClient(testAccProvider.Meta())

		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testAccResourceNLBZoneName),
		)
		r, err := client.GetNetworkLoadBalancer(ctx, testAccResourceNLBZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		return Copy(nlb, r)
	}
}

func testAccCheckResourceNLB(nlb *exov2.NetworkLoadBalancer) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if nlb.ID == "" {
			return errors.New("network load balancer ID is empty")
		}

		return nil
	}
}

func testAccCheckResourceNLBAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_nlb" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("resource not found in the state")
	}
}

func testAccCheckResourceNLBDestroy(s *terraform.State) error {
	client := GetComputeClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_nlb" {
			continue
		}

		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testAccResourceNLBZoneName),
		)
		_, err := client.GetNetworkLoadBalancer(
			ctx,
			testAccResourceNLBZoneName,
			rs.Primary.ID,
		)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}
	}

	return errors.New("network load balancer still exists")
}
