package exoscale

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	exov2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
)

var (
	testAccResourceSKSClusterName        = acctest.RandomWithPrefix(testPrefix)
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
  exoscale_ccm = true
  metrics_server = false

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
  exoscale_ccm = true
  metrics_server = false

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
	var sksCluster exov2.SKSCluster

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceSKSClusterDestroy,
		Steps: []resource.TestStep{
			{ // Create
				Config: testAccResourceSKSClusterConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists("exoscale_sks_cluster.test", &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						// Retrieve the latest SKS version available to test the
						// exoscale_sks_cluster.version attribute default value.
						client, err := exov2.NewClient(
							os.Getenv("EXOSCALE_API_KEY"),
							os.Getenv("EXOSCALE_API_SECRET"),
							exov2.ClientOptCond(func() bool {
								if v := os.Getenv("EXOSCALE_TRACE"); v != "" {
									return true
								}
								return false
							}, exov2.ClientOptWithTrace()))
						if err != nil {
							return fmt.Errorf("unable to initialize Exoscale client: %s", err)
						}

						versions, err := client.ListSKSClusterVersions(
							exoapi.WithEndpoint(
								context.Background(),
								exoapi.NewReqEndpoint(os.Getenv("EXOSCALE_API_ENVIRONMENT"), testZoneName)),
						)
						if err != nil || len(versions) == 0 {
							if len(versions) == 0 {
								err = errors.New("no version returned by the API")
							}
							return fmt.Errorf("unable to retrieve SKS versions: %s", err)
						}
						latestVersion := versions[0]

						a.Equal(latestVersion, sksCluster.Version)

						return nil
					},
					testAccCheckResourceSKSClusterAttributes(testAttrs{
						"cni":            ValidateString(defaultSKSClusterCNI),
						"created_at":     validation.ToDiagFunc(validation.NoZeroValues),
						"description":    ValidateString(testAccResourceSKSClusterDescription),
						"endpoint":       validation.ToDiagFunc(validation.IsURLWithHTTPS),
						"exoscale_ccm":   ValidateString("true"),
						"id":             validation.ToDiagFunc(validation.IsUUID),
						"metrics_server": ValidateString("false"),
						"name":           ValidateString(testAccResourceSKSClusterName),
						"service_level":  ValidateString(defaultSKSClusterServiceLevel),
						"state":          validation.ToDiagFunc(validation.NoZeroValues),
						"version":        validation.ToDiagFunc(validation.NoZeroValues),

						"zone": ValidateString(testZoneName),
					}),
				),
			},
			{ // Update
				Config: testAccResourceSKSClusterConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists("exoscale_sks_cluster.test", &sksCluster),
					testAccCheckResourceSKSClusterAttributes(testAttrs{
						"cni":            ValidateString(defaultSKSClusterCNI),
						"created_at":     validation.ToDiagFunc(validation.NoZeroValues),
						"description":    ValidateString(""),
						"endpoint":       validation.ToDiagFunc(validation.IsURLWithHTTPS),
						"exoscale_ccm":   ValidateString("true"),
						"id":             validation.ToDiagFunc(validation.IsUUID),
						"metrics_server": ValidateString("false"),
						"name":           ValidateString(testAccResourceSKSClusterNameUpdated),
						"nodepools.#":    ValidateString("1"),
						"service_level":  ValidateString(defaultSKSClusterServiceLevel),
						"state":          validation.ToDiagFunc(validation.NoZeroValues),
						"version":        validation.ToDiagFunc(validation.NoZeroValues),
						"zone":           ValidateString(testZoneName),
					}),
				),
			},
			{ // Import
				ResourceName:            "exoscale_sks_cluster.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"state"},
				ImportStateCheck: composeImportStateCheckFunc(
					testAccCheckResourceImportedAttributes(
						"exoscale_sks_cluster",
						testAttrs{
							"cni":            ValidateString(defaultSKSClusterCNI),
							"created_at":     validation.ToDiagFunc(validation.NoZeroValues),
							"description":    ValidateString(""),
							"endpoint":       validation.ToDiagFunc(validation.IsURLWithHTTPS),
							"exoscale_ccm":   ValidateString("true"),
							"id":             validation.ToDiagFunc(validation.IsUUID),
							"metrics_server": ValidateString("false"),
							"name":           ValidateString(testAccResourceSKSClusterNameUpdated),
							"nodepools.#":    ValidateString("1"),
							"service_level":  ValidateString(defaultSKSClusterServiceLevel),
							"state":          validation.ToDiagFunc(validation.NoZeroValues),
							"version":        validation.ToDiagFunc(validation.NoZeroValues),
							"zone":           ValidateString(testZoneName),
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

		*sksCluster = *r
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
