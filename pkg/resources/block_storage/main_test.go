package block_storage_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	egoscale "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

func TestBlockStorage(t *testing.T) {
	t.Parallel()

	volumeResourceName := "exoscale_block_storage_volume.test_volume"
	volumeDataSourceName := "data." + volumeResourceName
	snapshotResourceName := "exoscale_block_storage_volume_snapshot.test_snapshot"
	snapshotDataSourceName := "data." + snapshotResourceName
	volumeFromSnapshotResourceName := "exoscale_block_storage_volume.test_volume_from_snapshot"

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1 Create volume
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
			// 2 Update volume name only
			{
				Config: testutils.ParseTestdataConfig("./testdata/002.volume_rename.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						volumeResourceName,
						"name",
						fmt.Sprintf("terraform-provider-test-%d-renamed", testdataSpec.ID),
					),
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
			// 3 Clear volume labels by setting labels attribute to empty
			{
				Config: testutils.ParseTestdataConfig("./testdata/003.volume_empty_labels.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(volumeResourceName, "labels.%", "0"),
					resource.TestCheckResourceAttrSet(volumeResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(volumeResourceName, "blocksize"),
					resource.TestCheckResourceAttr(volumeResourceName, "state", "detached"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "size", "10"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "labels.%", "0"),
					resource.TestCheckResourceAttrSet(volumeDataSourceName, "created_at"),
					resource.TestCheckResourceAttrSet(volumeDataSourceName, "blocksize"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "state", "detached"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "snapshots.#", "0"),
					resource.TestCheckNoResourceAttr(volumeDataSourceName, "instance"),
				),
			},
			// 4 Update volume lables only
			{
				Config: testutils.ParseTestdataConfig("./testdata/004.volume_update.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(volumeResourceName, "labels.%", "2"),
					resource.TestCheckResourceAttr(volumeResourceName, "labels.foo1", "bar1"),
					resource.TestCheckResourceAttr(volumeResourceName, "labels.foo2", "bar2"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "labels.%", "2"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "labels.foo1", "bar1"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "labels.foo2", "bar2"),
				),
			},
			// 5 Unsetting volume labels should make the labels unmanaged.
			{
				Config: testutils.ParseTestdataConfig("./testdata/005.volume_unmanaged_labels.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckNoResourceAttr(volumeResourceName, "labels"),
					resource.TestCheckResourceAttrSet(volumeResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(volumeResourceName, "blocksize"),
					resource.TestCheckResourceAttr(volumeResourceName, "state", "detached"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "size", "10"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "labels.%", "2"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "labels.foo1", "bar1"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "labels.foo2", "bar2"),
					resource.TestCheckResourceAttrSet(volumeDataSourceName, "created_at"),
					resource.TestCheckResourceAttrSet(volumeDataSourceName, "blocksize"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "state", "detached"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "snapshots.#", "0"),
					resource.TestCheckNoResourceAttr(volumeDataSourceName, "instance"),
				),
			},
			// 6 Update volume labels and name
			{
				Config: testutils.ParseTestdataConfig("./testdata/006.volume_update_labels_and_name.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						volumeResourceName,
						"name",
						fmt.Sprintf("terraform-provider-test-%d-renamed-again", testdataSpec.ID),
					),
					resource.TestCheckResourceAttr(volumeResourceName, "labels.%", "3"),
					resource.TestCheckResourceAttr(volumeResourceName, "labels.foo1", "bar1"),
					resource.TestCheckResourceAttr(volumeResourceName, "labels.foo2", "bar2"),
					resource.TestCheckResourceAttr(volumeResourceName, "labels.foo3", "bar3"),
					resource.TestCheckResourceAttrSet(volumeResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(volumeResourceName, "blocksize"),
					resource.TestCheckResourceAttr(volumeResourceName, "state", "detached"),
					resource.TestCheckResourceAttr(
						volumeDataSourceName,
						"name",
						fmt.Sprintf("terraform-provider-test-%d-renamed-again", testdataSpec.ID),
					),
					resource.TestCheckResourceAttr(volumeDataSourceName, "size", "10"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "labels.%", "3"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "labels.foo1", "bar1"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "labels.foo2", "bar2"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "labels.foo3", "bar3"),
					resource.TestCheckResourceAttrSet(volumeDataSourceName, "created_at"),
					resource.TestCheckResourceAttrSet(volumeDataSourceName, "blocksize"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "state", "detached"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "snapshots.#", "0"),
					resource.TestCheckNoResourceAttr(volumeDataSourceName, "instance"),
				),
			},
			// 7 Resize volume
			{
				Config: testutils.ParseTestdataConfig("./testdata/007.volume_resize.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(volumeResourceName, "size", "20"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "size", "20"),
				),
			},
			// 8 Create instance & attach volume
			{
				Config: testutils.ParseTestdataConfig("./testdata/008.volume_attach.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("exoscale_compute_instance.test_instance", "block_storage_volume_ids.#", "1"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "instance.%", "1"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "state", "attached"),
				),
			},
			// 9 Detach volume from instance
			{
				Config: testutils.ParseTestdataConfig("./testdata/009.volume_detach.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("exoscale_compute_instance.test_instance", "block_storage_volume_ids.#", "0"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "instance.%", "0"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "state", "detached"),
				),
			},
			// 10 Create snapshot
			{
				Config: testutils.ParseTestdataConfig("./testdata/010.create_snapshot.tf.tmpl", &testdataSpec),
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
			// 11 Create volume from snapshot
			{
				Config: testutils.ParseTestdataConfig("./testdata/011.volume_from_snapshot.tf.tmpl", &testdataSpec),
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
			// 12 Unsetting snapshot labels should make the labels unmanaged.
			{
				Config: testutils.ParseTestdataConfig("./testdata/012.snapshot_unmanaged_labels.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(snapshotResourceName, "name"),
					resource.TestCheckResourceAttrSet(snapshotResourceName, "size"),
					resource.TestCheckNoResourceAttr(snapshotResourceName, "labels"),
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
			// 13 Update snapshot name and labels
			{
				Config: testutils.ParseTestdataConfig("./testdata/013.snapshot_update_name_and_labels.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						snapshotResourceName,
						"name",
						fmt.Sprintf("terraform-provider-test-%d-renamed", testdataSpec.ID),
					),
					resource.TestCheckResourceAttrSet(snapshotResourceName, "size"),
					resource.TestCheckResourceAttr(snapshotResourceName, "labels.%", "2"),
					resource.TestCheckResourceAttr(snapshotResourceName, "labels.l1", "v1"),
					resource.TestCheckResourceAttr(snapshotResourceName, "labels.l2", "v2"),
					resource.TestCheckResourceAttrSet(snapshotResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(snapshotResourceName, "state"),
					resource.TestCheckResourceAttr(
						snapshotDataSourceName,
						"name",
						fmt.Sprintf("terraform-provider-test-%d-renamed", testdataSpec.ID),
					),
					resource.TestCheckResourceAttrSet(snapshotDataSourceName, "size"),
					resource.TestCheckResourceAttr(snapshotDataSourceName, "labels.%", "2"),
					resource.TestCheckResourceAttr(snapshotDataSourceName, "labels.l1", "v1"),
					resource.TestCheckResourceAttr(snapshotDataSourceName, "labels.l2", "v2"),
					resource.TestCheckResourceAttrSet(snapshotDataSourceName, "created_at"),
					resource.TestCheckResourceAttr(snapshotDataSourceName, "state", "created"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "snapshots.#", "1"),
				),
			},
			// 14 Update snapshot name only
			{
				Config: testutils.ParseTestdataConfig("./testdata/014.snapshot_update_name.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						snapshotResourceName,
						"name",
						fmt.Sprintf("terraform-provider-test-%d-renamed-again", testdataSpec.ID),
					),
					resource.TestCheckResourceAttrSet(snapshotResourceName, "size"),
					resource.TestCheckResourceAttrSet(snapshotResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(snapshotResourceName, "state"),
					resource.TestCheckResourceAttr(
						snapshotDataSourceName,
						"name",
						fmt.Sprintf("terraform-provider-test-%d-renamed-again", testdataSpec.ID),
					),
					resource.TestCheckResourceAttrSet(snapshotDataSourceName, "size"),
					resource.TestCheckResourceAttr(snapshotDataSourceName, "labels.%", "2"),
					resource.TestCheckResourceAttr(snapshotDataSourceName, "labels.l1", "v1"),
					resource.TestCheckResourceAttr(snapshotDataSourceName, "labels.l2", "v2"),
					resource.TestCheckResourceAttrSet(snapshotDataSourceName, "created_at"),
					resource.TestCheckResourceAttr(snapshotDataSourceName, "state", "created"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "snapshots.#", "1"),
				),
			},
			// 15 Update snapshot labels only
			{
				Config: testutils.ParseTestdataConfig("./testdata/015.snapshot_update_labels.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(snapshotResourceName, "size"),
					resource.TestCheckResourceAttr(snapshotResourceName, "labels.%", "1"),
					resource.TestCheckResourceAttr(snapshotResourceName, "labels.l2", "v2"),
					resource.TestCheckResourceAttrSet(snapshotResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(snapshotResourceName, "state"),
					resource.TestCheckResourceAttr(
						snapshotDataSourceName,
						"name",
						fmt.Sprintf("terraform-provider-test-%d-renamed-again", testdataSpec.ID),
					),
					resource.TestCheckResourceAttrSet(snapshotDataSourceName, "size"),
					resource.TestCheckResourceAttr(snapshotDataSourceName, "labels.%", "1"),
					resource.TestCheckResourceAttr(snapshotDataSourceName, "labels.l2", "v2"),
					resource.TestCheckResourceAttrSet(snapshotDataSourceName, "created_at"),
					resource.TestCheckResourceAttr(snapshotDataSourceName, "state", "created"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "snapshots.#", "1"),
				),
			},
			// 16 Clear snapshot labels by setting empty labels attribute
			{
				Config: testutils.ParseTestdataConfig("./testdata/016.snapshot_empty_labels.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(snapshotResourceName, "size"),
					resource.TestCheckResourceAttr(snapshotResourceName, "labels.%", "0"),
					resource.TestCheckResourceAttrSet(snapshotResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(snapshotResourceName, "state"),
					resource.TestCheckResourceAttr(
						snapshotDataSourceName,
						"name",
						fmt.Sprintf("terraform-provider-test-%d-renamed-again", testdataSpec.ID),
					),
					resource.TestCheckResourceAttrSet(snapshotDataSourceName, "size"),
					resource.TestCheckResourceAttr(snapshotDataSourceName, "labels.%", "0"),
					resource.TestCheckResourceAttrSet(snapshotDataSourceName, "created_at"),
					resource.TestCheckResourceAttr(snapshotDataSourceName, "state", "created"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "snapshots.#", "1"),
				),
			},
			// Import
			{
				ResourceName: volumeResourceName,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(s *terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", s.RootModule().Resources[volumeResourceName].Primary.ID, testdataSpec.Zone), nil
					}
				}(),
				ImportState: true,
			},
		},
	})
}

func TestBlockStorageDeleteAndDetach(t *testing.T) {
	t.Parallel()

	volumeResourceName := "exoscale_block_storage_volume.test_volume"
	volumeDataSourceName := "data." + volumeResourceName
	var volumeID string

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testAccCheckBlockStorageVolumeDestroy(testdataSpec.Zone, &volumeID),
		Steps: []resource.TestStep{
			// 1 Create instance
			{
				Config: testutils.ParseTestdataConfig("./testdata/001.instance_create.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						"exoscale_compute_instance.test_instance",
						"name",
						fmt.Sprintf("terraform-provider-test-%d", testdataSpec.ID),
					),
				),
			},
			// 2 Create and attach volume
			{
				Config: testutils.ParseTestdataConfig("./testdata/002.volume_create_attach.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(volumeResourceName, "id", func(v string) error {
						volumeID = v
						return nil
					}),
					resource.TestCheckResourceAttr(
						volumeResourceName,
						"name",
						fmt.Sprintf("terraform-provider-test-%d", testdataSpec.ID),
					),
					resource.TestCheckResourceAttr("exoscale_compute_instance.test_instance", "block_storage_volume_ids.#", "1"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "instance.%", "1"),
					resource.TestCheckResourceAttrSet(volumeResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(volumeResourceName, "blocksize"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "size", "20"),
					resource.TestCheckResourceAttr(volumeResourceName, "labels.%", "1"),
					resource.TestCheckResourceAttr(volumeResourceName, "labels.foo1", "bar1"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "labels.%", "1"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "labels.foo1", "bar1"),
					resource.TestCheckResourceAttrSet(volumeDataSourceName, "created_at"),
					resource.TestCheckResourceAttrSet(volumeDataSourceName, "blocksize"),
					resource.TestCheckResourceAttr(volumeDataSourceName, "state", "attached"),
				),
			},
			// 3 Detach and delete volume, crucially, in the same step
			{
				Config: testutils.ParseTestdataConfig("./testdata/003.volume_detach_delete.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("exoscale_compute_instance.test_instance", "block_storage_volume_ids.#", "0"),
				),
			},
		},
	})
}

func testAccCheckBlockStorageVolumeDestroy(zone string, volumeID *string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		if volumeID == nil || *volumeID == "" {
			return errors.New("volume ID was not captured during the test")
		}

		defaultClient, err := testutils.APIClientV3()
		if err != nil {
			return fmt.Errorf("unable to initialize Exoscale client: %w", err)
		}

		ctx := context.Background()
		client, err := utils.SwitchClientZone(ctx, defaultClient, egoscale.ZoneName(zone))
		if err != nil {
			return fmt.Errorf("unable to initialize Exoscale client: %w", err)
		}

		_, err = client.GetBlockStorageVolume(ctx, egoscale.UUID(*volumeID))
		if err != nil {
			if errors.Is(err, egoscale.ErrNotFound) {
				return nil
			}
			return err
		}

		return errors.New("block storage volume still exists")
	}
}
