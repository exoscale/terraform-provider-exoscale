package exoscale

import (
	"context"
	"errors"
	"fmt"
	"testing"

	exov2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var (
	testAccResourceSKSClusterName        = testPrefix + "-" + testRandomString()
	testAccResourceSKSClusterNameUpdated = testAccResourceSKSClusterName + "-updated"
	testAccResourceSKSClusterDescription = testDescription

	testAccResourceSKSClusterConfigCreate = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_sks_cluster" "test" {
  zone = local.zone
  name = "%s"
  description = "%s"

  timeouts {
    create = "10m"
  }
}

resource "exoscale_sks_nodepool" "test" {
  zone = local.zone
  cluster_id = exoscale_sks_cluster.test.id
  name = "test"
  instance_type = "small"
  disk_size = 20
  size = 1

  timeouts {
    delete = "10m"
  }
}
`,
		testZoneName,
		testAccResourceSKSClusterName,
		testAccResourceSKSClusterDescription,
	)

	testAccResourceSKSClusterConfigUpdate = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_sks_cluster" "test" {
  zone = local.zone
  name = "%s"

  timeouts {
    create = "10m"
  }
}

resource "exoscale_sks_nodepool" "test" {
  zone = local.zone
  cluster_id = exoscale_sks_cluster.test.id
  name = "test"
  instance_type = "small"
  disk_size = 20
  size = 1

  timeouts {
    create = "10m"
  }
}
`,
		testZoneName,
		testAccResourceSKSClusterNameUpdated,
	)
)

func TestAccResourceSKSCluster(t *testing.T) {
	sksCluster := new(exov2.SKSCluster)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceSKSClusterDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceSKSClusterConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists("exoscale_sks_cluster.test", sksCluster),
					testAccCheckResourceSKSCluster(sksCluster),
					testAccCheckResourceSKSClusterAttributes(testAttrs{
						"addons.791607250": ValidateString(defaultSKSClusterAddOns[0]),
						"cni":              ValidateString(defaultSKSClusterCNI),
						"created_at":       validation.NoZeroValues,
						"description":      ValidateString(testAccResourceSKSClusterDescription),
						"endpoint":         validation.IsURLWithHTTPS,
						"id":               validation.IsUUID,
						"name":             ValidateString(testAccResourceSKSClusterName),
						"service_level":    ValidateString(defaultSKSClusterServiceLevel),
						"state":            validation.NoZeroValues,
						"version":          ValidateString(defaultSKSClusterVersion),
						"zone":             ValidateString(testZoneName),
					}),
				),
			},
			{
				Config: testAccResourceSKSClusterConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists("exoscale_sks_cluster.test", sksCluster),
					testAccCheckResourceSKSCluster(sksCluster),
					testAccCheckResourceSKSClusterAttributes(testAttrs{
						"addons.791607250": ValidateString(defaultSKSClusterAddOns[0]),
						"cni":              ValidateString(defaultSKSClusterCNI),
						"created_at":       validation.NoZeroValues,
						"description":      ValidateString(""),
						"endpoint":         validation.IsURLWithHTTPS,
						"id":               validation.IsUUID,
						"name":             ValidateString(testAccResourceSKSClusterNameUpdated),
						"nodepools.#":      ValidateString("1"),
						"service_level":    ValidateString(defaultSKSClusterServiceLevel),
						"state":            validation.NoZeroValues,
						"version":          ValidateString(defaultSKSClusterVersion),
						"zone":             ValidateString(testZoneName),
					}),
				),
			},
			{
				ResourceName:            "exoscale_sks_cluster.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"state"},
				ImportStateCheck: composeImportStateCheckFunc(
					testAccCheckResourceImportedAttributes(
						"exoscale_sks_cluster",
						testAttrs{
							"addons.791607250": ValidateString(defaultSKSClusterAddOns[0]),
							"cni":              ValidateString(defaultSKSClusterCNI),
							"created_at":       validation.NoZeroValues,
							"description":      ValidateString(""),
							"endpoint":         validation.IsURLWithHTTPS,
							"id":               validation.IsUUID,
							"name":             ValidateString(testAccResourceSKSClusterNameUpdated),
							"nodepools.#":      ValidateString("1"),
							"service_level":    ValidateString(defaultSKSClusterServiceLevel),
							"state":            validation.NoZeroValues,
							"version":          ValidateString(defaultSKSClusterVersion),
							"zone":             ValidateString(testZoneName),
						},
					),
				),
			},
		},
	})
}

func testAccCheckResourceSKSClusterExists(n string, sksCluster *exov2.SKSCluster) resource.TestCheckFunc {
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
			exoapi.NewReqEndpoint(testEnvironment, testZoneName),
		)
		r, err := client.GetSKSCluster(ctx, testZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		return Copy(sksCluster, r)
	}
}

func testAccCheckResourceSKSCluster(sksCluster *exov2.SKSCluster) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if sksCluster.ID == "" {
			return errors.New("SKS cluster ID is empty")
		}

		return nil
	}
}

func testAccCheckResourceSKSClusterAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_sks_cluster" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("resource not found in the state")
	}
}

func testAccCheckResourceSKSClusterDestroy(s *terraform.State) error {
	client := GetComputeClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_sks_cluster" {
			continue
		}

		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testZoneName),
		)

		if _, err := client.GetSKSCluster(ctx, testZoneName, rs.Primary.ID); err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}
	}

	return errors.New("SKS cluster still exists")
}
