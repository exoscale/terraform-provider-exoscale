package sks_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

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
					resource.TestCheckResourceAttrSet(r, "created_at"),
					resource.TestCheckResourceAttr(r, "description", "description-test"),
					resource.TestCheckResourceAttr(r, "disk_size", "100"),
					resource.TestMatchResourceAttr(r, "instance_pool_id", testUUIDRegex),
					resource.TestCheckResourceAttr(r, "instance_prefix", "test"),
					resource.TestCheckResourceAttr(r, "instance_type", "standard.small"),
					resource.TestCheckResourceAttr(r, "ipv6", "false"),
					resource.TestCheckResourceAttr(r, "kubelet_image_gc.min_age", "1m"),
					resource.TestCheckResourceAttr(r, "kubelet_image_gc.high_threshold", "13"),
					resource.TestCheckResourceAttr(r, "kubelet_image_gc.low_threshold", "12"),
					resource.TestCheckResourceAttr(r, "kubelet_max_pods", "210"),
					resource.TestCheckResourceAttr(r, "labels.test", "acctest"),
					resource.TestCheckResourceAttr(r, "name", name),
					resource.TestCheckResourceAttr(r, "size", "2"),
					resource.TestCheckResourceAttrSet(r, "state"),
					resource.TestCheckResourceAttr(r, "storage_lvm", "true"),
					resource.TestCheckResourceAttr(r, "taints.test", "test:NoSchedule"),
					resource.TestMatchResourceAttr(r, "template_id", testUUIDRegex),
					resource.TestCheckResourceAttrSet(r, "version"),
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
					resource.TestCheckResourceAttr(r, "anti_affinity_group_ids.#", "1"),
					resource.TestMatchResourceAttr(r, "cluster_id", testUUIDRegex),
					resource.TestCheckResourceAttrSet(r, "created_at"),
					resource.TestCheckResourceAttr(r, "description", "description-test-updated"),
					resource.TestCheckResourceAttr(r, "disk_size", "110"),
					resource.TestMatchResourceAttr(r, "instance_pool_id", testUUIDRegex),
					resource.TestCheckResourceAttr(r, "instance_prefix", "pool"),
					resource.TestCheckResourceAttr(r, "instance_type", "standard.medium"),
					resource.TestCheckResourceAttr(r, "ipv6", "true"),
					resource.TestCheckResourceAttr(r, "labels.test", "acctest-updated"),
					resource.TestCheckResourceAttr(r, "name", name+"-updated"),
					resource.TestCheckResourceAttr(r, "private_network_ids.#", "1"),
					resource.TestCheckResourceAttr(r, "security_group_ids.#", "1"),
					resource.TestCheckResourceAttr(r, "size", "1"),
					resource.TestCheckResourceAttrSet(r, "state"),
					resource.TestCheckResourceAttr(r, "taints.test", "test-updated:NoSchedule"),
					resource.TestMatchResourceAttr(r, "template_id", testUUIDRegex),
					resource.TestCheckResourceAttrSet(r, "version"),
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
					resource.TestCheckResourceAttr(r, "kubelet_image_gc.min_age", "5m"),
					resource.TestCheckResourceAttr(r, "kubelet_image_gc.high_threshold", "85"),
					resource.TestCheckResourceAttr(r, "kubelet_image_gc.low_threshold", "75"),
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
