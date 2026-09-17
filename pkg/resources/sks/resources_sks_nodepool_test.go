package sks_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	egoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const (
	sksNodepoolAddonStorageLVM = "storage-lvm"
)

func TestAccResourceSKSNodepool(t *testing.T) {
	t.Parallel()

	if os.Getenv(resource.EnvTfAcc) == "" {
		t.Skipf("Acceptance tests skipped unless env '%s' set", resource.EnvTfAcc)
		return
	}

	var (
		r           = "exoscale_sks_nodepool.test"
		sksCluster  egoscale.SKSCluster
		sksNodepool egoscale.SKSNodepool
	)

	td := sksTestdata{Zone: testZoneName, ID: time.Now().UnixNano()}
	name := testutils.ResourceName(td.ID)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckResourceSKSNodepoolDestroy(r),
		Steps: []resource.TestStep{
			{
				// Create
				Config: parseSKSConfig(t, "./testdata/011.sksnp_create.tf.tmpl", td),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSClusterExists("exoscale_sks_cluster.test", &sksCluster),
					testAccCheckResourceSKSNodepoolExists(r, &sksNodepool),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal("description-test", sksNodepool.Description)
						a.Equal(int64(100), sksNodepool.DiskSize)
						a.Equal("acctest", sksNodepool.Labels["test"])
						a.Equal(name, sksNodepool.Name)
						a.Equal("test", sksNodepool.InstancePrefix)
						a.Equal(int64(2), sksNodepool.Size)
						a.True(slices.Contains(sksNodepool.Addons, sksNodepoolAddonStorageLVM))
						a.Equal(egoscale.SKSNodepoolTaint{Value: "test", Effect: "NoSchedule"}, sksNodepool.Taints["test"])
						if a.NotNil(sksNodepool.KubeletImageGC) {
							a.Equal("1m", sksNodepool.KubeletImageGC.MinAge)
							a.Equal(int64(13), sksNodepool.KubeletImageGC.HighThreshold)
							a.Equal(int64(12), sksNodepool.KubeletImageGC.LowThreshold)
						}
						if a.NotNil(sksNodepool.KubeletMaxPods) {
							a.Equal(int64(210), *sksNodepool.KubeletMaxPods)
						}
						return nil
					},
					testutils.CheckResourceState(r, testutils.CheckResourceStateValidateAttributes(testutils.TestAttrs{
						"created_at":                      validation.ToDiagFunc(validation.NoZeroValues),
						"description":                     testutils.ValidateString("description-test"),
						"disk_size":                       testutils.ValidateString("100"),
						"instance_pool_id":                validation.ToDiagFunc(validation.IsUUID),
						"instance_prefix":                 testutils.ValidateString("test"),
						"instance_type":                   testutils.ValidateString("standard.small"),
						"ipv6":                            testutils.ValidateString("false"),
						"kubelet_image_gc.min_age":        testutils.ValidateString("1m"),
						"kubelet_image_gc.high_threshold": testutils.ValidateString("13"),
						"kubelet_image_gc.low_threshold":  testutils.ValidateString("12"),
						"kubelet_max_pods":                testutils.ValidateString("210"),
						"labels.test":                     testutils.ValidateString("acctest"),
						"name":                            testutils.ValidateString(name),
						"size":                            testutils.ValidateString("2"),
						"state":                           validation.ToDiagFunc(validation.NoZeroValues),
						"storage_lvm":                     testutils.ValidateString("true"),
						"taints.test":                     testutils.ValidateString("test:NoSchedule"),
						"template_id":                     validation.ToDiagFunc(validation.IsUUID),
						"version":                         validation.ToDiagFunc(validation.NoZeroValues),
					})),
				),
			},
			{
				// Update
				Config: parseSKSConfig(t, "./testdata/012.sksnp_update.tf.tmpl", td),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSNodepoolExists(r, &sksNodepool),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Len(sksNodepool.AntiAffinityGroups, 1)
						a.Equal("description-test-updated", sksNodepool.Description)
						a.Equal(int64(110), sksNodepool.DiskSize)
						a.Equal("acctest-updated", sksNodepool.Labels["test"])
						a.Equal(name+"-updated", sksNodepool.Name)
						a.Equal("pool", sksNodepool.InstancePrefix)
						a.Len(sksNodepool.PrivateNetworks, 1)
						a.Len(sksNodepool.SecurityGroups, 1)
						a.Equal(int64(1), sksNodepool.Size)
						a.Equal(egoscale.SKSNodepoolTaint{Value: "test-updated", Effect: "NoSchedule"}, sksNodepool.Taints["test"])
						a.Equal(egoscale.SKSNodepoolPublicIPAssignmentDual, sksNodepool.PublicIPAssignment)
						return nil
					},
					testutils.CheckResourceState(r, testutils.CheckResourceStateValidateAttributes(testutils.TestAttrs{
						"anti_affinity_group_ids.#": testutils.ValidateString("1"),
						"created_at":                validation.ToDiagFunc(validation.NoZeroValues),
						"description":               testutils.ValidateString("description-test-updated"),
						"disk_size":                 testutils.ValidateString("110"),
						"instance_pool_id":          validation.ToDiagFunc(validation.IsUUID),
						"instance_prefix":           testutils.ValidateString("pool"),
						"instance_type":             testutils.ValidateString("standard.medium"),
						"ipv6":                      testutils.ValidateString("true"),
						"labels.test":               testutils.ValidateString("acctest-updated"),
						"name":                      testutils.ValidateString(name + "-updated"),
						"private_network_ids.#":     testutils.ValidateString("1"),
						"security_group_ids.#":      testutils.ValidateString("1"),
						"size":                      testutils.ValidateString("1"),
						"state":                     validation.ToDiagFunc(validation.NoZeroValues),
						"taints.test":               testutils.ValidateString("test-updated:NoSchedule"),
						"template_id":               validation.ToDiagFunc(validation.IsUUID),
						"version":                   validation.ToDiagFunc(validation.NoZeroValues),
					})),
				),
			},
			{
				// Import
				ResourceName: r,
				ImportStateIdFunc: func(sksCluster *egoscale.SKSCluster, sksNodepool *egoscale.SKSNodepool) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s/%s@%s", sksCluster.ID, sksNodepool.ID, testZoneName), nil
					}
				}(&sksCluster, &sksNodepool),
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return testutils.CheckResourceAttributes(
						testutils.TestAttrs{
							"anti_affinity_group_ids.#": testutils.ValidateString("1"),
							"cluster_id":                validation.ToDiagFunc(validation.IsUUID),
							"created_at":                validation.ToDiagFunc(validation.NoZeroValues),
							"description":               testutils.ValidateString("description-test-updated"),
							"disk_size":                 testutils.ValidateString("110"),
							"instance_pool_id":          validation.ToDiagFunc(validation.IsUUID),
							"instance_prefix":           testutils.ValidateString("pool"),
							"instance_type":             testutils.ValidateString("standard.medium"),
							"name":                      testutils.ValidateString(name + "-updated"),
							"private_network_ids.#":     testutils.ValidateString("1"),
							"security_group_ids.#":      testutils.ValidateString("1"),
							"size":                      testutils.ValidateString("1"),
							"state":                     validation.ToDiagFunc(validation.NoZeroValues),
							"taints.test":               testutils.ValidateString("test-updated:NoSchedule"),
							"template_id":               validation.ToDiagFunc(validation.IsUUID),
							"version":                   validation.ToDiagFunc(validation.NoZeroValues),
						},
						func(s []*terraform.InstanceState) map[string]string {
							for _, state := range s {
								if state.ID == sksNodepool.ID.String() {
									return state.Attributes
								}
							}
							return nil
						}(s),
					)
				},
			},
			{
				// Update kubelet image GC
				Config: parseSKSConfig(t, "./testdata/013.sksnp_update_kubelet_gc.tf.tmpl", td),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSKSNodepoolExists(r, &sksNodepool),
					func(s *terraform.State) error {
						a := assert.New(t)

						if a.NotNil(sksNodepool.KubeletImageGC) {
							a.Equal("5m", sksNodepool.KubeletImageGC.MinAge)
							a.Equal(int64(85), sksNodepool.KubeletImageGC.HighThreshold)
							a.Equal(int64(75), sksNodepool.KubeletImageGC.LowThreshold)
						}
						return nil
					},
					testutils.CheckResourceState(r, testutils.CheckResourceStateValidateAttributes(testutils.TestAttrs{
						"kubelet_image_gc.min_age":        testutils.ValidateString("5m"),
						"kubelet_image_gc.high_threshold": testutils.ValidateString("85"),
						"kubelet_image_gc.low_threshold":  testutils.ValidateString("75"),
					})),
				),
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

		clusterID, ok := rs.Primary.Attributes["cluster_id"]
		if !ok {
			return errors.New(`resource attribute "cluster_id" not set`)
		}

		defaultClient, err := testutils.APIClientV3()
		if err != nil {
			return fmt.Errorf("unable to initialize Exoscale client: %s", err)
		}
		ctx := context.Background()
		client, err := utils.SwitchClientZone(
			ctx,
			defaultClient,
			egoscale.ZoneName(testZoneName),
		)
		if err != nil {
			return fmt.Errorf("unable to initialize Exoscale client: %s", err)
		}

		cluster, err := client.GetSKSCluster(ctx, egoscale.UUID(clusterID))
		if err != nil {
			return err
		}

		for i := range cluster.Nodepools {
			if cluster.Nodepools[i].ID.String() == rs.Primary.ID {
				*sksNodepool = cluster.Nodepools[i]
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

		clusterID, ok := rs.Primary.Attributes["cluster_id"]
		if !ok {
			return errors.New(`resource attribute "cluster_id" not set`)
		}

		defaultClient, err := testutils.APIClientV3()
		if err != nil {
			return fmt.Errorf("unable to initialize Exoscale client: %s", err)
		}
		ctx := context.Background()
		client, err := utils.SwitchClientZone(
			ctx,
			defaultClient,
			egoscale.ZoneName(testZoneName),
		)
		if err != nil {
			return fmt.Errorf("unable to initialize Exoscale client: %s", err)
		}

		cluster, err := client.GetSKSCluster(ctx, egoscale.UUID(clusterID))
		if err != nil {
			if errors.Is(err, egoscale.ErrNotFound) {
				return nil
			}
			return err
		}

		for i := range cluster.Nodepools {
			if cluster.Nodepools[i].ID.String() == rs.Primary.ID {
				return errors.New("SKS Nodepool still exists")
			}
		}

		return nil
	}
}
