package block_storage_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testVolume(t *testing.T) {
	resourceName := "exoscale_block_storage_volume.test_volume"
	dataSourceName := "data." + resourceName

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
						resourceName,
						"name",
						fmt.Sprintf("test_volume_%d", testdataSpec.ID),
					),
					resource.TestCheckResourceAttr(resourceName, "size", "120"),
					resource.TestCheckResourceAttr(resourceName, "labels.%", "1"),
					resource.TestCheckResourceAttr(resourceName, "labels.foo", "bar"),
					resource.TestCheckResourceAttrSet(resourceName, "created_at"),
					resource.TestCheckResourceAttrSet(resourceName, "blocksize"),
					resource.TestCheckResourceAttrSet(resourceName, "state"),
					resource.TestCheckResourceAttr(dataSourceName, "size", "120"),
					resource.TestCheckResourceAttr(dataSourceName, "labels.%", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "labels.foo", "bar"),
					resource.TestCheckResourceAttrSet(dataSourceName, "created_at"),
					resource.TestCheckResourceAttrSet(dataSourceName, "blocksize"),
					resource.TestCheckResourceAttr(dataSourceName, "state", "detached"),
					resource.TestCheckResourceAttr(dataSourceName, "snapshots.#", "0"),
					resource.TestCheckNoResourceAttr(dataSourceName, "instance"),
				),
			},
			// Update volume lables
			{
				Config: testutils.ParseTestdataConfig("./testdata/002.volume_update.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						resourceName,
						"name",
						fmt.Sprintf("test_volume_%d", testdataSpec.ID),
					),
					resource.TestCheckResourceAttr(resourceName, "labels.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "labels.foo1", "bar1"),
					resource.TestCheckResourceAttr(resourceName, "labels.foo2", "bar2"),
					resource.TestCheckResourceAttr(dataSourceName, "labels.%", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "labels.foo1", "bar1"),
					resource.TestCheckResourceAttr(dataSourceName, "labels.foo2", "bar2"),
				),
			},
			// Resize volume
			{
				Config: testutils.ParseTestdataConfig("./testdata/003.volume_resize.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						resourceName,
						"name",
						fmt.Sprintf("test_volume_%d", testdataSpec.ID),
					),
					resource.TestCheckResourceAttr(resourceName, "size", "130"),
					resource.TestCheckResourceAttr(dataSourceName, "size", "130"),
				),
			},
			// Create instance & attach volume
			{
				Config: testutils.ParseTestdataConfig("./testdata/004.volume_attach.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						resourceName,
						"name",
						fmt.Sprintf("test_volume_%d", testdataSpec.ID),
					),
					resource.TestCheckResourceAttr("exoscale_compute_instance.test_instance", "block_storage_volume_ids.#", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "instance.%", "1"),
					resource.TestCheckResourceAttr(dataSourceName, "state", "attached"),
				),
			},

			// Detach volume from instance
			{
				Config: testutils.ParseTestdataConfig("./testdata/005.volume_detach.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						resourceName,
						"name",
						fmt.Sprintf("test_volume_%d", testdataSpec.ID),
					),
					resource.TestCheckResourceAttr("exoscale_compute_instance.test_instance", "block_storage_volume_ids.#", "0"),
					resource.TestCheckResourceAttr(dataSourceName, "instance.%", "0"),
					resource.TestCheckResourceAttr(dataSourceName, "state", "detached"),
				),
			},
			// Import
			{
				ResourceName: resourceName,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(s *terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", s.RootModule().Resources[resourceName].Primary.ID, "ch-gva-2"), nil
					}
				}(),
				ImportState: true,
			},
		},
	})
}
