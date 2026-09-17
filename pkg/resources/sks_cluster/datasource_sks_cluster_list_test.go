package sks_cluster_test

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	tftest "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func TestAccDataSourceSKSClusterList(t *testing.T) {
	t.Parallel()

	var (
		r  = "exoscale_sks_cluster.test"
		ds = "data.exoscale_sks_cluster_list.list"
	)

	td := sksTestdata{Zone: testZoneName, ID: time.Now().UnixNano()}
	name := testutils.ResourceName(td.ID)

	tftest.Test(t, tftest.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []tftest.TestStep{
			{
				// Create the cluster only.
				Config: parseSKSConfig(t, "./testdata/017.sks_cluster_list_create.tf.tmpl", td),
				Check:  tftest.TestCheckResourceAttr(r, "name", name),
			},
			{
				// Add the list data source lookup.
				Config: parseSKSConfig(t, "./testdata/018.sks_cluster_list_read.tf.tmpl", td),
				Check: tftest.ComposeAggregateTestCheckFunc(
					tftest.TestCheckResourceAttr(r, "name", name),
					testAccCheckSKSClusterInList(ds, r),
				),
			},
		},
	})
}

// testAccCheckSKSClusterInList asserts that the "clusters" list exposed by
// the exoscale_sks_cluster_list data source at dsAddr contains an entry
// matching the exoscale_sks_cluster resource at resAddr. The zone used by
// acceptance tests is shared across parallel test runs, so the list may
// contain more than just the cluster created by this test.
func testAccCheckSKSClusterInList(dsAddr, resAddr string) tftest.TestCheckFunc {
	return func(s *terraform.State) error {
		ds, ok := s.RootModule().Resources[dsAddr]
		if !ok {
			return fmt.Errorf("data source not found: %s", dsAddr)
		}

		res, ok := s.RootModule().Resources[resAddr]
		if !ok {
			return fmt.Errorf("resource not found: %s", resAddr)
		}
		wantID := res.Primary.ID

		count, err := strconv.Atoi(ds.Primary.Attributes["clusters.#"])
		if err != nil {
			return fmt.Errorf("invalid clusters.# attribute: %s", err)
		}

		for i := 0; i < count; i++ {
			if ds.Primary.Attributes[fmt.Sprintf("clusters.%d.id", i)] == wantID {
				return nil
			}
		}

		return fmt.Errorf("cluster %q not found in %s.clusters", wantID, dsAddr)
	}
}
