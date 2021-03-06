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
	testAccResourceSKSClusterDescription = acctest.RandString(10)

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
  auto_upgrade = true

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
  description = ""
  exoscale_ccm = true
  metrics_server = false
  auto_upgrade = false

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
	var (
		r          = "exoscale_sks_cluster.test"
		sksCluster exov2.SKSCluster
	)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},

		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceSKSClusterDestroy(&sksCluster),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceSKSClusterConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
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

						a.Equal([]string{sksClusterAddonExoscaleCCM}, *sksCluster.AddOns)
						a.True(defaultBool(sksCluster.AutoUpgrade, false))
						a.Equal(defaultSKSClusterCNI, *sksCluster.CNI)
						a.Equal(testAccResourceSKSClusterDescription, *sksCluster.Description)
						a.Equal(testAccResourceSKSClusterName, *sksCluster.Name)
						a.Equal(defaultSKSClusterServiceLevel, *sksCluster.ServiceLevel)
						a.Equal(latestVersion, *sksCluster.Version)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSKSClusterAttrAutoUpgrade:   ValidateString("true"),
						resSKSClusterAttrCNI:           ValidateString(defaultSKSClusterCNI),
						resSKSClusterAttrCreatedAt:     validation.ToDiagFunc(validation.NoZeroValues),
						resSKSClusterAttrDescription:   ValidateString(testAccResourceSKSClusterDescription),
						resSKSClusterAttrEndpoint:      validation.ToDiagFunc(validation.IsURLWithHTTPS),
						resSKSClusterAttrExoscaleCCM:   ValidateString("true"),
						resSKSClusterAttrMetricsServer: ValidateString("false"),
						resSKSClusterAttrName:          ValidateString(testAccResourceSKSClusterName),
						resSKSClusterAttrServiceLevel:  ValidateString(defaultSKSClusterServiceLevel),
						resSKSClusterAttrState:         validation.ToDiagFunc(validation.NoZeroValues),
						resSKSClusterAttrVersion:       validation.ToDiagFunc(validation.NoZeroValues),
					})),
				),
			},
			{
				// Update
				Config: testAccResourceSKSClusterConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists(r, &sksCluster),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.False(defaultBool(sksCluster.AutoUpgrade, false))
						a.Empty(defaultString(sksCluster.Description, ""))
						a.Equal(testAccResourceSKSClusterNameUpdated, *sksCluster.Name)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSKSClusterAttrAutoUpgrade:   ValidateString("false"),
						resSKSClusterAttrCNI:           ValidateString(defaultSKSClusterCNI),
						resSKSClusterAttrCreatedAt:     validation.ToDiagFunc(validation.NoZeroValues),
						resSKSClusterAttrDescription:   validation.ToDiagFunc(validation.StringIsEmpty),
						resSKSClusterAttrEndpoint:      validation.ToDiagFunc(validation.IsURLWithHTTPS),
						resSKSClusterAttrExoscaleCCM:   ValidateString("true"),
						resSKSClusterAttrMetricsServer: ValidateString("false"),
						resSKSClusterAttrName:          ValidateString(testAccResourceSKSClusterNameUpdated),
						resSKSClusterAttrServiceLevel:  ValidateString(defaultSKSClusterServiceLevel),
						resSKSClusterAttrState:         validation.ToDiagFunc(validation.NoZeroValues),
						resSKSClusterAttrVersion:       validation.ToDiagFunc(validation.NoZeroValues),
					})),
				),
			},
			{
				// Import
				ResourceName: r,
				ImportStateIdFunc: func(sksCluster *exov2.SKSCluster) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", *sksCluster.ID, testZoneName), nil
					}
				}(&sksCluster),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resSKSClusterAttrAutoUpgrade:   ValidateString("false"),
							resSKSClusterAttrCNI:           ValidateString(defaultSKSClusterCNI),
							resSKSClusterAttrCreatedAt:     validation.ToDiagFunc(validation.NoZeroValues),
							resSKSClusterAttrDescription:   validation.ToDiagFunc(validation.StringIsEmpty),
							resSKSClusterAttrEndpoint:      validation.ToDiagFunc(validation.IsURLWithHTTPS),
							resSKSClusterAttrExoscaleCCM:   ValidateString("true"),
							resSKSClusterAttrMetricsServer: ValidateString("false"),
							resSKSClusterAttrName:          ValidateString(testAccResourceSKSClusterNameUpdated),
							resSKSClusterAttrServiceLevel:  ValidateString(defaultSKSClusterServiceLevel),
							resSKSClusterAttrState:         validation.ToDiagFunc(validation.NoZeroValues),
							resSKSClusterAttrVersion:       validation.ToDiagFunc(validation.NoZeroValues),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceSKSClusterExists(r string, sksCluster *exov2.SKSCluster) resource.TestCheckFunc {
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
		res, err := client.GetSKSCluster(ctx, testZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*sksCluster = *res
		return nil
	}
}

func testAccCheckResourceSKSClusterDestroy(sksCluster *exov2.SKSCluster) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testZoneName),
		)

		_, err := client.GetSKSCluster(ctx, testZoneName, *sksCluster.ID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("SKS cluster still exists")
	}
}
