package sks_test

import (
	"testing"
	"time"

	tftest "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func TestAccDataSourceSKSNodepool(t *testing.T) {
	t.Parallel()

	var (
		r        = "exoscale_sks_nodepool.test"
		dsByID   = "data.exoscale_sks_nodepool.by_id"
		dsByName = "data.exoscale_sks_nodepool.by_name"
	)

	td := sksTestdata{Zone: testZoneName, ID: time.Now().UnixNano()}
	name := testutils.ResourceName(td.ID)

	tftest.Test(t, tftest.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []tftest.TestStep{
			{
				// Create the cluster + nodepool only.
				Config: parseSKSConfig(t, "./testdata/014.sksnp_datasource_create.tf.tmpl", td),
				Check:  tftest.TestCheckResourceAttr(r, "name", name),
			},
			{
				// Add the data source lookups; the implicit refresh before this
				// step's plan picks up the nodepool created above.
				Config: parseSKSConfig(t, "./testdata/015.sksnp_datasource_read.tf.tmpl", td),
				Check: tftest.ComposeAggregateTestCheckFunc(
					tftest.TestCheckResourceAttr(r, "name", name),

					// data source by id
					tftest.TestCheckResourceAttrPair(r, "id", dsByID, "id"),
					tftest.TestCheckResourceAttrPair(r, "name", dsByID, "name"),
					tftest.TestCheckResourceAttrPair(r, "cluster_id", dsByID, "cluster_id"),
					tftest.TestCheckResourceAttrPair(r, "created_at", dsByID, "created_at"),
					tftest.TestCheckResourceAttrPair(r, "description", dsByID, "description"),
					tftest.TestCheckResourceAttrPair(r, "disk_size", dsByID, "disk_size"),
					tftest.TestCheckResourceAttrPair(r, "instance_pool_id", dsByID, "instance_pool_id"),
					tftest.TestCheckResourceAttrPair(r, "instance_prefix", dsByID, "instance_prefix"),
					tftest.TestCheckResourceAttrPair(r, "instance_type", dsByID, "instance_type"),
					tftest.TestCheckResourceAttrPair(r, "labels.test", dsByID, "labels.test"),
					tftest.TestCheckResourceAttrPair(r, "size", dsByID, "size"),
					tftest.TestCheckResourceAttrPair(r, "state", dsByID, "state"),
					tftest.TestCheckResourceAttrPair(r, "template_id", dsByID, "template_id"),
					tftest.TestCheckResourceAttrPair(r, "version", dsByID, "version"),

					// data source by name
					tftest.TestCheckResourceAttrPair(r, "id", dsByName, "id"),
					tftest.TestCheckResourceAttrPair(r, "name", dsByName, "name"),
					tftest.TestCheckResourceAttrPair(r, "cluster_id", dsByName, "cluster_id"),
					tftest.TestCheckResourceAttrPair(r, "created_at", dsByName, "created_at"),
					tftest.TestCheckResourceAttrPair(r, "description", dsByName, "description"),
					tftest.TestCheckResourceAttrPair(r, "disk_size", dsByName, "disk_size"),
					tftest.TestCheckResourceAttrPair(r, "instance_pool_id", dsByName, "instance_pool_id"),
					tftest.TestCheckResourceAttrPair(r, "instance_prefix", dsByName, "instance_prefix"),
					tftest.TestCheckResourceAttrPair(r, "instance_type", dsByName, "instance_type"),
					tftest.TestCheckResourceAttrPair(r, "labels.test", dsByName, "labels.test"),
					tftest.TestCheckResourceAttrPair(r, "size", dsByName, "size"),
					tftest.TestCheckResourceAttrPair(r, "state", dsByName, "state"),
					tftest.TestCheckResourceAttrPair(r, "template_id", dsByName, "template_id"),
					tftest.TestCheckResourceAttrPair(r, "version", dsByName, "version"),
				),
			},
		},
	})
}
