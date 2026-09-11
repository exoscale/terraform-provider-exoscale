package sks_cluster_test

import (
	"testing"
	"time"

	tftest "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func TestAccDataSourceSKSCluster(t *testing.T) {
	t.Parallel()

	var (
		r        = "exoscale_sks_cluster.test"
		dsByID   = "data.exoscale_sks_cluster.by_id"
		dsByName = "data.exoscale_sks_cluster.by_name"
	)

	td := sksTestdata{Zone: testZoneName, ID: time.Now().UnixNano()}
	name := testutils.ResourceName(td.ID)

	tftest.Test(t, tftest.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []tftest.TestStep{
			{
				// Create the cluster + nodepool only.
				Config: parseSKSConfig(t, "./testdata/009.sks_datasource_create.tf.tmpl", td),
				Check:  tftest.TestCheckResourceAttr(r, "name", name),
			},
			{
				// Add the data source lookups; the implicit refresh before this
				// step's plan picks up the nodepool created above.
				Config: parseSKSConfig(t, "./testdata/010.sks_datasource_read.tf.tmpl", td),
				Check: tftest.ComposeAggregateTestCheckFunc(
					tftest.TestCheckResourceAttr(r, "name", name),

					// data source by id
					tftest.TestCheckResourceAttrPair(r, "id", dsByID, "id"),
					tftest.TestCheckResourceAttrPair(r, "name", dsByID, "name"),
					tftest.TestCheckResourceAttrPair(r, "addons.#", dsByID, "addons.#"),
					tftest.TestCheckResourceAttrPair(r, "auto_upgrade", dsByID, "auto_upgrade"),
					tftest.TestCheckResourceAttrPair(r, "cni", dsByID, "cni"),
					tftest.TestCheckResourceAttrPair(r, "created_at", dsByID, "created_at"),
					tftest.TestCheckResourceAttrPair(r, "default_security_group_id", dsByID, "default_security_group_id"),
					tftest.TestCheckResourceAttrPair(r, "description", dsByID, "description"),
					tftest.TestCheckResourceAttrPair(r, "enable_kube_proxy", dsByID, "enable_kube_proxy"),
					tftest.TestCheckResourceAttrPair(r, "endpoint", dsByID, "endpoint"),
					tftest.TestCheckResourceAttrPair(r, "feature_gates.#", dsByID, "feature_gates.#"),
					tftest.TestCheckResourceAttrPair(r, "labels.test", dsByID, "labels.test"),
					tftest.TestCheckResourceAttrPair(r, "nodepools.#", dsByID, "nodepools.#"),
					tftest.TestCheckResourceAttrPair(r, "service_level", dsByID, "service_level"),
					tftest.TestCheckResourceAttrPair(r, "state", dsByID, "state"),
					tftest.TestCheckResourceAttrPair(r, "version", dsByID, "version"),

					// data source by name
					tftest.TestCheckResourceAttrPair(r, "id", dsByName, "id"),
					tftest.TestCheckResourceAttrPair(r, "name", dsByName, "name"),
					tftest.TestCheckResourceAttrPair(r, "addons.#", dsByName, "addons.#"),
					tftest.TestCheckResourceAttrPair(r, "auto_upgrade", dsByName, "auto_upgrade"),
					tftest.TestCheckResourceAttrPair(r, "cni", dsByName, "cni"),
					tftest.TestCheckResourceAttrPair(r, "created_at", dsByName, "created_at"),
					tftest.TestCheckResourceAttrPair(r, "default_security_group_id", dsByName, "default_security_group_id"),
					tftest.TestCheckResourceAttrPair(r, "description", dsByName, "description"),
					tftest.TestCheckResourceAttrPair(r, "enable_kube_proxy", dsByName, "enable_kube_proxy"),
					tftest.TestCheckResourceAttrPair(r, "endpoint", dsByName, "endpoint"),
					tftest.TestCheckResourceAttrPair(r, "feature_gates.#", dsByName, "feature_gates.#"),
					tftest.TestCheckResourceAttrPair(r, "labels.test", dsByName, "labels.test"),
					tftest.TestCheckResourceAttrPair(r, "nodepools.#", dsByName, "nodepools.#"),
					tftest.TestCheckResourceAttrPair(r, "service_level", dsByName, "service_level"),
					tftest.TestCheckResourceAttrPair(r, "state", dsByName, "state"),
					tftest.TestCheckResourceAttrPair(r, "version", dsByName, "version"),
				),
			},
		},
	})
}
