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
	testAccResourceSKSNodepoolAntiAffinityGroupName = testPrefix + "-" + testRandomString()
	testAccResourceSKSNodepoolDescription           = testPrefix + "-" + testRandomString()
	testAccResourceSKSNodepoolDiskSize              = defaultSKSNodepoolDiskSize
	testAccResourceSKSNodepoolDiskSizeUpdated       = defaultSKSNodepoolDiskSize * 2
	testAccResourceSKSNodepoolInstanceType          = "small"
	testAccResourceSKSNodepoolInstanceTypeUpdated   = "medium"
	testAccResourceSKSNodepoolName                  = testPrefix + "-" + testRandomString()
	testAccResourceSKSNodepoolNameUpdated           = testAccResourceSKSNodepoolName + "-updated"
	testAccResourceSKSNodepoolSize                  = 1
	testAccResourceSKSNodepoolSizeUpdated           = 2

	testAccResourceSKSNodepoolConfigCreate = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_sks_cluster" "test" {
  zone = local.zone
  name = "%s"

  timeouts {
    delete = "10m"
  }
}

resource "exoscale_sks_nodepool" "test" {
  zone = local.zone
  cluster_id = exoscale_sks_cluster.test.id
  name = "%s"
  description = "%s"
  instance_type = "%s"
  disk_size = %d
  size = %d

  timeouts {
    delete = "10m"
  }
}
`,
		testZoneName,
		testAccResourceSKSClusterName,
		testAccResourceSKSNodepoolName,
		testAccResourceSKSNodepoolDescription,
		testAccResourceSKSNodepoolInstanceType,
		testAccResourceSKSNodepoolDiskSize,
		testAccResourceSKSNodepoolSize,
	)

	testAccResourceSKSNodepoolConfigUpdate = fmt.Sprintf(`
locals {
  zone = "%s"
}

data "exoscale_security_group" "default" {
  name = "default"
}

resource "exoscale_affinity" "test" {
  name = "%s"
}

resource "exoscale_sks_cluster" "test" {
  zone = local.zone
  name = "%s"

  timeouts {
    delete = "10m"
  }
}

resource "exoscale_sks_nodepool" "test" {
  zone = local.zone
  cluster_id = exoscale_sks_cluster.test.id
  name = "%s"
  instance_type = "%s"
  disk_size = %d
  size = %d
  anti_affinity_group_ids = [exoscale_affinity.test.id]
  security_group_ids = [data.exoscale_security_group.default.id]

  timeouts {
    delete = "10m"
  }
}
	  `,
		testZoneName,
		testAccResourceSKSNodepoolAntiAffinityGroupName,
		testAccResourceSKSClusterName,
		testAccResourceSKSNodepoolNameUpdated,
		testAccResourceSKSNodepoolInstanceTypeUpdated,
		testAccResourceSKSNodepoolDiskSizeUpdated,
		testAccResourceSKSNodepoolSizeUpdated,
	)
)

func TestAccResourceSKSNodepool(t *testing.T) {
	nodepool := new(exov2.SKSNodepool)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceSKSNodepoolDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceSKSNodepoolConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSNodepoolExists("exoscale_sks_nodepool.test", nodepool),
					testAccCheckResourceSKSNodepool(nodepool),
					testAccCheckResourceSKSNodepoolAttributes(testAttrs{
						"created_at":       validation.NoZeroValues,
						"description":      ValidateString(testAccResourceSKSNodepoolDescription),
						"disk_size":        ValidateString(fmt.Sprint(testAccResourceSKSNodepoolDiskSize)),
						"id":               validation.IsUUID,
						"instance_pool_id": validation.IsUUID,
						"instance_type":    ValidateString(testAccResourceSKSNodepoolInstanceType),
						"name":             ValidateString(testAccResourceSKSNodepoolName),
						"size":             ValidateString(fmt.Sprint(testAccResourceSKSNodepoolSize)),
						"state":            validation.NoZeroValues,
						"template_id":      validation.IsUUID,
						"version":          validation.NoZeroValues,
					}),
				),
			},
			{
				Config: testAccResourceSKSNodepoolConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSNodepoolExists("exoscale_sks_nodepool.test", nodepool),
					testAccCheckResourceSKSNodepool(nodepool),
					testAccCheckResourceSKSNodepoolAttributes(testAttrs{
						"created_at":                validation.NoZeroValues,
						"description":               ValidateString(""),
						"disk_size":                 ValidateString(fmt.Sprint(testAccResourceSKSNodepoolDiskSizeUpdated)),
						"id":                        validation.IsUUID,
						"instance_pool_id":          validation.IsUUID,
						"instance_type":             ValidateString(testAccResourceSKSNodepoolInstanceTypeUpdated),
						"name":                      ValidateString(testAccResourceSKSNodepoolNameUpdated),
						"anti_affinity_group_ids.#": ValidateString("1"),
						"security_group_ids.#":      ValidateString("1"),
						"size":                      ValidateString(fmt.Sprint(testAccResourceSKSNodepoolSizeUpdated)),
						"state":                     validation.NoZeroValues,
						"template_id":               validation.IsUUID,
						"version":                   validation.NoZeroValues,
					}),
				),
			},
			{
				ResourceName:            "exoscale_sks_nodepool.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"state"},
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							"created_at":                validation.NoZeroValues,
							"description":               ValidateString(""),
							"disk_size":                 ValidateString(fmt.Sprint(testAccResourceSKSNodepoolDiskSizeUpdated)),
							"id":                        validation.IsUUID,
							"instance_pool_id":          validation.IsUUID,
							"instance_type":             ValidateString(testAccResourceSKSNodepoolInstanceTypeUpdated),
							"name":                      ValidateString(testAccResourceSKSNodepoolNameUpdated),
							"anti_affinity_group_ids.#": ValidateString("1"),
							"security_group_ids.#":      ValidateString("1"),
							"size":                      ValidateString(fmt.Sprint(testAccResourceSKSNodepoolSizeUpdated)),
							"state":                     validation.NoZeroValues,
							"template_id":               validation.IsUUID,
							"version":                   validation.NoZeroValues,
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceSKSNodepoolExists(n string, nodepool *exov2.SKSNodepool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		clusterID, ok := rs.Primary.Attributes["cluster_id"]
		if !ok {
			return errors.New("resource cluster_id not set")
		}

		client := GetComputeClient(testAccProvider.Meta())

		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testZoneName),
		)
		cluster, err := client.GetSKSCluster(ctx, testZoneName, clusterID)
		if err != nil {
			return err
		}

		for _, np := range cluster.Nodepools {
			if np.ID == rs.Primary.ID {
				return Copy(nodepool, np)
			}
		}

		return fmt.Errorf("resource SKS Nodepool %q not found", rs.Primary.ID)
	}
}

func testAccCheckResourceSKSNodepool(nodepool *exov2.SKSNodepool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if nodepool.ID == "" {
			return errors.New("SKS Nodepool ID is empty")
		}

		return nil
	}
}

func testAccCheckResourceSKSNodepoolAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_sks_nodepool" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("resource not found in the state")
	}
}

func testAccCheckResourceSKSNodepoolDestroy(s *terraform.State) error {
	client := GetComputeClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_sks_nodepool" {
			continue
		}

		clusterID, ok := rs.Primary.Attributes["cluster_id"]
		if !ok {
			return errors.New("resource cluster_id not set")
		}

		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testZoneName),
		)
		cluster, err := client.GetSKSCluster(ctx, testZoneName, clusterID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		for _, np := range cluster.Nodepools {
			if np.ID == rs.Primary.ID {
				return errors.New("SKS Nodepool still exists")
			}
		}
	}

	return nil
}
