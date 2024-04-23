package block_storage_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func TestBlockStorage(t *testing.T) {
	volumeResourceName := "exoscale_block_storage_volume.test_volume"
	volumeDataSourceName := "data." + volumeResourceName
	snapshotResourceName := "exoscale_block_storage_volume_snapshot.test_snapshot"
	snapshotDataSourceName := "data." + snapshotResourceName
	volumeFromSnapshotResourceName := "exoscale_block_storage_volume.test_volume_from_snapshot"

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: "ch-gva-2", // to be replaced by global testutils.TestZoneName when BS reaches GA.
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create volume
			{
				Config: testutils.ParseTestdataConfig("./testdata/001.volume_create.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						volumeResourceName,
						"name",
						fmt.Sprintf("terraform-provider-test-%d", testdataSpec.ID),
					),
					resource.TestCheckResourceAttr(volumeResourceName, "size", "10"),
					resource.TestCheckResourceAttr(volumeResourceName, "labels.%", "1"),
					resource.TestCheckResourceAttr(volumeResourceName, "labels.foo", "bar"),
					resource.TestCheckResourceAttrSet(volumeResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(volumeResourceName, "blocksize"),
					resource.TestCheckResourceAttr(volumeResourceName, "state", "detached"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "size", "10"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "labels.%", "1"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "labels.foo", "bar"),
					resource.TestCheckResourceAttrSet(volumeDataSourceName, "created_at"),
					resource.TestCheckResourceAttrSet(volumeDataSourceName, "blocksize"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "state", "detached"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "snapshots.#", "0"),
					resource.TestCheckNoResourceAttr(volumeDataSourceName, "instance"),
				),
			},
			// Update volume lables
			{
				Config: testutils.ParseTestdataConfig("./testdata/002.volume_update.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(volumeResourceName, "labels.%", "2"),
					resource.TestCheckResourceAttr(volumeResourceName, "labels.foo1", "bar1"),
					resource.TestCheckResourceAttr(volumeResourceName, "labels.foo2", "bar2"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "labels.%", "2"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "labels.foo1", "bar1"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "labels.foo2", "bar2"),
				),
			},
			// Resize volume
			{
				Config: testutils.ParseTestdataConfig("./testdata/003.volume_resize.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(volumeResourceName, "size", "20"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "size", "20"),
				),
			},
			// Create instance & attach volume
			{
				Config: testutils.ParseTestdataConfig("./testdata/004.volume_attach.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("exoscale_compute_instance.test_instance", "block_storage_volume_ids.#", "1"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "instance.%", "1"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "state", "attached"),
				),
			},
			// Detach volume from instance
			{
				Config: testutils.ParseTestdataConfig("./testdata/005.volume_detach.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("exoscale_compute_instance.test_instance", "block_storage_volume_ids.#", "0"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "instance.%", "0"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "state", "detached"),
				),
			},
			// Create snapshot
			{
				Config: testutils.ParseTestdataConfig("./testdata/006.create_snapshot.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(snapshotResourceName, "name"),
					resource.TestCheckResourceAttrSet(snapshotResourceName, "size"),
					resource.TestCheckResourceAttr(snapshotResourceName, "labels.%", "1"),
					resource.TestCheckResourceAttr(snapshotResourceName, "labels.foo", "bar"),
					resource.TestCheckResourceAttrSet(snapshotResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(snapshotResourceName, "state"),
					resource.TestCheckResourceAttrSet(snapshotDataSourceName, "size"),
					resource.TestCheckResourceAttr(snapshotDataSourceName, "labels.%", "1"),
					resource.TestCheckResourceAttr(snapshotDataSourceName, "labels.foo", "bar"),
					resource.TestCheckResourceAttrSet(snapshotDataSourceName, "created_at"),
					resource.TestCheckResourceAttr(snapshotDataSourceName, "state", "created"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "snapshots.#", "1"),
				),
			},
			// Create volume from snapshot
			{
				Config: testutils.ParseTestdataConfig("./testdata/007.volume_from_snapshot.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						volumeFromSnapshotResourceName,
						"name",
						fmt.Sprintf("terraform-provider-test-%d", testdataSpec.ID),
					),
					resource.TestCheckResourceAttr(volumeFromSnapshotResourceName, "labels.%", "0"),
					resource.TestCheckResourceAttrSet(volumeFromSnapshotResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(volumeFromSnapshotResourceName, "blocksize"),
					resource.TestCheckResourceAttr(volumeFromSnapshotResourceName, "state", "detached"),
				),
			},
			// Import
			{
				ResourceName: volumeResourceName,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(s *terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", s.RootModule().Resources[volumeResourceName].Primary.ID, "ch-gva-2"), nil
					}
				}(),
				ImportState: true,
			},
		},
	})
}
