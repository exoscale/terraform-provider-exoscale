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
	testAccResourceSKSNodepoolAntiAffinityGroupName       = acctest.RandomWithPrefix(testPrefix)
	testAccResourceSKSNodepoolDescription                 = acctest.RandomWithPrefix(testPrefix)
	testAccResourceSKSNodepoolDiskSize                    = defaultSKSNodepoolDiskSize
	testAccResourceSKSNodepoolDiskSizeUpdated             = defaultSKSNodepoolDiskSize * 2
	testAccResourceSKSNodepoolInstancePrefix              = "test"
	testAccResourceSKSNodepoolInstanceType                = "standard.small"
	testAccResourceSKSNodepoolInstanceTypeUpdated         = "standard.medium"
	testAccResourceSKSNodepoolName                        = acctest.RandomWithPrefix(testPrefix)
	testAccResourceSKSNodepoolNameUpdated                 = testAccResourceSKSNodepoolName + "-updated"
	testAccResourceSKSNodepoolPrivateNetworkName          = acctest.RandomWithPrefix(testPrefix)
	testAccResourceSKSNodepoolSize                  int64 = 2
	testAccResourceSKSNodepoolSizeUpdated           int64 = 1

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
  instance_prefix = "%s"

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
		testAccResourceSKSNodepoolInstancePrefix,
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

resource "exoscale_network" "test" {
  zone     = local.zone
  name     = "%s"
  start_ip = "10.0.0.20"
  end_ip   = "10.0.0.253"
  netmask  = "255.255.255.0"
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
  description = ""
  instance_type = "%s"
  disk_size = %d
  size = %d
  instance_prefix = "%s"
  anti_affinity_group_ids = [exoscale_affinity.test.id]
  security_group_ids = [data.exoscale_security_group.default.id]
  private_network_ids = [exoscale_network.test.id]

  timeouts {
    delete = "10m"
  }
}
	  `,
		testZoneName,
		testAccResourceSKSNodepoolAntiAffinityGroupName,
		testAccResourceSKSNodepoolPrivateNetworkName,
		testAccResourceSKSClusterName,
		testAccResourceSKSNodepoolNameUpdated,
		testAccResourceSKSNodepoolInstanceTypeUpdated,
		testAccResourceSKSNodepoolDiskSizeUpdated,
		testAccResourceSKSNodepoolSizeUpdated,
		defaultSKSNodepoolInstancePrefix,
	)
)

func TestAccResourceSKSNodepool(t *testing.T) {
	var (
		r           = "exoscale_sks_nodepool.test"
		sksCluster  egoscale.SKSCluster
		sksNodepool egoscale.SKSNodepool
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceSKSNodepoolDestroy(r),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceSKSNodepoolConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists("exoscale_sks_cluster.test", &sksCluster),
					testAccCheckResourceSKSNodepoolExists(r, &sksNodepool),
					func(s *terraform.State) error {
						a := require.New(t)

						a.Equal(testAccResourceSKSNodepoolDescription, *sksNodepool.Description)
						a.Equal(testAccResourceSKSNodepoolDiskSize, *sksNodepool.DiskSize)
						a.Equal(testAccResourceSKSNodepoolName, *sksNodepool.Name)
						a.Equal(testAccResourceSKSNodepoolInstancePrefix, *sksNodepool.InstancePrefix)
						a.Equal(testInstanceTypeIDSmall, *sksNodepool.InstanceTypeID)
						a.Equal(testAccResourceSKSNodepoolSize, *sksNodepool.Size)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSKSNodepoolAttrCreatedAt:      validation.ToDiagFunc(validation.NoZeroValues),
						resSKSNodepoolAttrDescription:    validateString(testAccResourceSKSNodepoolDescription),
						resSKSNodepoolAttrDiskSize:       validateString(fmt.Sprint(testAccResourceSKSNodepoolDiskSize)),
						resSKSNodepoolAttrInstancePoolID: validation.ToDiagFunc(validation.IsUUID),
						resSKSNodepoolAttrInstancePrefix: validateString(testAccResourceSKSNodepoolInstancePrefix),
						resSKSNodepoolAttrInstanceType:   validateString(testAccResourceSKSNodepoolInstanceType),
						resSKSNodepoolAttrName:           validateString(testAccResourceSKSNodepoolName),
						resSKSNodepoolAttrSize:           validateString(fmt.Sprint(testAccResourceSKSNodepoolSize)),
						resSKSNodepoolAttrState:          validation.ToDiagFunc(validation.NoZeroValues),
						resSKSNodepoolAttrTemplateID:     validation.ToDiagFunc(validation.IsUUID),
						resSKSNodepoolAttrVersion:        validation.ToDiagFunc(validation.NoZeroValues),
					})),
				),
			},
			{
				// Update
				Config: testAccResourceSKSNodepoolConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSNodepoolExists(r, &sksNodepool),
					func(s *terraform.State) error {
						a := require.New(t)

						a.Len(*sksNodepool.AntiAffinityGroupIDs, 1)
						a.Empty(defaultString(sksNodepool.Description, ""))
						a.Equal(testAccResourceSKSNodepoolDiskSizeUpdated, *sksNodepool.DiskSize)
						a.Equal(testAccResourceSKSNodepoolNameUpdated, *sksNodepool.Name)
						a.Equal(defaultSKSNodepoolInstancePrefix, *sksNodepool.InstancePrefix)
						a.Equal(testInstanceTypeIDMedium, *sksNodepool.InstanceTypeID)
						a.Len(*sksNodepool.PrivateNetworkIDs, 1)
						a.Len(*sksNodepool.SecurityGroupIDs, 1)
						a.Equal(testAccResourceSKSNodepoolSizeUpdated, *sksNodepool.Size)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSKSNodepoolAttrAntiAffinityGroupIDs + ".#": validateString("1"),
						resSKSNodepoolAttrCreatedAt:                   validation.ToDiagFunc(validation.NoZeroValues),
						resSKSNodepoolAttrDescription:                 validation.ToDiagFunc(validation.StringIsEmpty),
						resSKSNodepoolAttrDiskSize:                    validateString(fmt.Sprint(testAccResourceSKSNodepoolDiskSizeUpdated)),
						resSKSNodepoolAttrInstancePoolID:              validation.ToDiagFunc(validation.IsUUID),
						resSKSNodepoolAttrInstancePrefix:              validateString(defaultSKSNodepoolInstancePrefix),
						resSKSNodepoolAttrInstanceType:                validateString(testAccResourceSKSNodepoolInstanceTypeUpdated),
						resSKSNodepoolAttrName:                        validateString(testAccResourceSKSNodepoolNameUpdated),
						resSKSNodepoolAttrPrivateNetworkIDs + ".#":    validateString("1"),
						resSKSNodepoolAttrSecurityGroupIDs + ".#":     validateString("1"),
						resSKSNodepoolAttrSize:                        validateString(fmt.Sprint(testAccResourceSKSNodepoolSizeUpdated)),
						resSKSNodepoolAttrState:                       validation.ToDiagFunc(validation.NoZeroValues),
						resSKSNodepoolAttrTemplateID:                  validation.ToDiagFunc(validation.IsUUID),
						resSKSNodepoolAttrVersion:                     validation.ToDiagFunc(validation.NoZeroValues),
					})),
				),
			},
			{
				// Import
				ResourceName: r,
				ImportStateIdFunc: func(
					sksCluster *egoscale.SKSCluster,
					sksNodepool *egoscale.SKSNodepool,
				) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s/%s@%s", *sksCluster.ID, *sksNodepool.ID, testZoneName), nil
					}
				}(&sksCluster, &sksNodepool),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resSKSNodepoolAttrAntiAffinityGroupIDs + ".#": validateString("1"),
							resSKSNodepoolAttrCreatedAt:                   validation.ToDiagFunc(validation.NoZeroValues),
							resSKSNodepoolAttrClusterID:                   validation.ToDiagFunc(validation.IsUUID),
							resSKSNodepoolAttrDescription:                 validation.ToDiagFunc(validation.StringIsEmpty),
							resSKSNodepoolAttrDiskSize:                    validateString(fmt.Sprint(testAccResourceSKSNodepoolDiskSizeUpdated)),
							resSKSNodepoolAttrInstancePoolID:              validation.ToDiagFunc(validation.IsUUID),
							resSKSNodepoolAttrInstancePrefix:              validateString(defaultSKSNodepoolInstancePrefix),
							resSKSNodepoolAttrInstanceType:                validateString(testAccResourceSKSNodepoolInstanceTypeUpdated),
							resSKSNodepoolAttrName:                        validateString(testAccResourceSKSNodepoolNameUpdated),
							resSKSNodepoolAttrPrivateNetworkIDs + ".#":    validateString("1"),
							resSKSNodepoolAttrSecurityGroupIDs + ".#":     validateString("1"),
							resSKSNodepoolAttrSize:                        validateString(fmt.Sprint(testAccResourceSKSNodepoolSizeUpdated)),
							resSKSNodepoolAttrState:                       validation.ToDiagFunc(validation.NoZeroValues),
							resSKSNodepoolAttrTemplateID:                  validation.ToDiagFunc(validation.IsUUID),
							resSKSNodepoolAttrVersion:                     validation.ToDiagFunc(validation.NoZeroValues),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceSKSNodepoolExists(r string, sksNodepool *egoscale.SKSNodepool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		clusterID, ok := rs.Primary.Attributes[resSKSNodepoolAttrClusterID]
		if !ok {
			return fmt.Errorf("resource attribute %q not set", resSKSNodepoolAttrClusterID)
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
			if *np.ID == rs.Primary.ID {
				*sksNodepool = *np
				return nil
			}
		}

		return fmt.Errorf("resource SKS Nodepool %q not found", rs.Primary.ID)
	}
}

func testAccCheckResourceSKSNodepoolDestroy(r string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		clusterID, ok := rs.Primary.Attributes[resSKSNodepoolAttrClusterID]
		if !ok {
			return fmt.Errorf("resource attribute %q not set", resSKSNodepoolAttrClusterID)
		}

		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testZoneName),
		)

		sksCluster, err := client.GetSKSCluster(ctx, testZoneName, clusterID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		for _, np := range sksCluster.Nodepools {
			if *np.ID == rs.Primary.ID {
				return errors.New("SKS Nodepool still exists")
			}
		}

		return nil
	}
}
