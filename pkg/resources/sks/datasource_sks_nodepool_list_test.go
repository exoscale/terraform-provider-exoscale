package sks_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	tftest "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func TestAccDataSourceSKSNodepoolList(t *testing.T) {
	t.Parallel()

	var (
		clusterRes  = "exoscale_sks_cluster.test"
		nodepoolRes = "exoscale_sks_nodepool.test"
		ds          = "data.exoscale_sks_nodepool_list.list"
	)

	td := sksTestdata{Zone: testZoneName, ID: time.Now().UnixNano()}
	name := testutils.ResourceName(td.ID)

	tftest.Test(t, tftest.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []tftest.TestStep{
			{
				// Create the cluster + nodepool only.
				Config: parseSKSConfig(t, "./testdata/019.sks_nodepool_list_create.tf.tmpl", td),
				Check:  tftest.TestCheckResourceAttr(nodepoolRes, "name", name),
			},
			{
				// Add the list data source lookup.
				Config: parseSKSConfig(t, "./testdata/020.sks_nodepool_list_read.tf.tmpl", td),
				Check: tftest.ComposeAggregateTestCheckFunc(
					tftest.TestCheckResourceAttr(nodepoolRes, "name", name),
					testAccCheckSKSNodepoolInList(ds, nodepoolRes, clusterRes),
				),
			},
		},
	})
}

// testAccCheckSKSNodepoolInList asserts that the "nodepools" list exposed by
// the exoscale_sks_nodepool_list data source at dsAddr contains an entry
// matching the exoscale_sks_nodepool resource at nodepoolAddr, with a
// "cluster_id" pointing back at the exoscale_sks_cluster resource at
// clusterAddr. The zone used by acceptance tests is shared across parallel
// test runs, so the list may contain more than just the nodepool created by
// this test.
func testAccCheckSKSNodepoolInList(dsAddr, nodepoolAddr, clusterAddr string) tftest.TestCheckFunc {
	return func(s *terraform.State) error {
		ds, ok := s.RootModule().Resources[dsAddr]
		if !ok {
			return fmt.Errorf("data source not found: %s", dsAddr)
		}

		nodepool, ok := s.RootModule().Resources[nodepoolAddr]
		if !ok {
			return fmt.Errorf("resource not found: %s", nodepoolAddr)
		}
		wantID := nodepool.Primary.ID

		cluster, ok := s.RootModule().Resources[clusterAddr]
		if !ok {
			return fmt.Errorf("resource not found: %s", clusterAddr)
		}
		wantClusterID := cluster.Primary.ID

		count, err := strconv.Atoi(ds.Primary.Attributes["nodepools.#"])
		if err != nil {
			return fmt.Errorf("invalid nodepools.# attribute: %s", err)
		}

		for i := 0; i < count; i++ {
			if ds.Primary.Attributes[fmt.Sprintf("nodepools.%d.id", i)] != wantID {
				continue
			}

			gotClusterID := ds.Primary.Attributes[fmt.Sprintf("nodepools.%d.cluster_id", i)]
			if gotClusterID != wantClusterID {
				return fmt.Errorf("nodepool %q has cluster_id %q, want %q", wantID, gotClusterID, wantClusterID)
			}

			return nil
		}

		return fmt.Errorf("nodepool %q not found in %s.nodepools", wantID, dsAddr)
	}
}
