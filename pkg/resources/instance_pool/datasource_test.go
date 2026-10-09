package instance_pool_test

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func testDataSource(t *testing.T) {
	t.Parallel()

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}

	checks := func(ds string) resource.TestCheckFunc {
		return resource.ComposeAggregateTestCheckFunc(
			resource.TestCheckResourceAttrPair(ds, "id", poolResource, "id"),
			resource.TestCheckResourceAttr(ds, "name", testutils.ResourceName(testdataSpec.ID)),
			resource.TestCheckResourceAttr(ds, "zone", testdataSpec.Zone),
			resource.TestCheckResourceAttr(ds, "description", fmt.Sprintf("description-%d", testdataSpec.ID)),
			resource.TestCheckResourceAttr(ds, "anti_affinity_group_ids.#", "1"),
			resource.TestCheckTypeSetElemAttrPair(ds, "anti_affinity_group_ids.*", "exoscale_anti_affinity_group.test", "id"),
			resource.TestCheckResourceAttr(ds, "affinity_group_ids.#", "1"),
			resource.TestCheckResourceAttr(ds, "disk_size", "10"),
			resource.TestCheckResourceAttr(ds, "instance_prefix", "test"),
			resource.TestCheckResourceAttr(ds, "instance_type", "standard.tiny"),
			resource.TestCheckResourceAttr(ds, "ipv6", "false"),
			resource.TestCheckResourceAttrPair(ds, "key_pair", "exoscale_ssh_key.test", "name"),
			resource.TestCheckResourceAttr(ds, "labels.%", "1"),
			resource.TestCheckResourceAttr(ds, "labels.test", fmt.Sprintf("label-%d", testdataSpec.ID)),
			resource.TestCheckResourceAttr(ds, "network_ids.#", "1"),
			resource.TestCheckTypeSetElemAttrPair(ds, "network_ids.*", "exoscale_private_network.test", "id"),
			resource.TestCheckResourceAttr(ds, "size", "1"),
			resource.TestCheckResourceAttrPair(ds, "template_id", "data.exoscale_template.ubuntu", "id"),
			resource.TestCheckResourceAttr(ds, "user_data", fmt.Sprintf("user-data-%d", testdataSpec.ID)),
			resource.TestCheckResourceAttr(ds, "instances.#", "1"),
			resource.TestCheckResourceAttrPair(ds, "instances.0.id", poolResource, "instances.0.id"),
			resource.TestCheckResourceAttrPair(ds, "instances.0.name", poolResource, "instances.0.name"),
		)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testutils.ParseTestdataConfig("./testdata/004.datasource_missing_lookup.tf.tmpl", &testdataSpec),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
			{
				Config: testutils.ParseTestdataConfig("./testdata/005.datasource.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					checks("data.exoscale_instance_pool.by_id"),
					checks("data.exoscale_instance_pool.by_name"),
				),
			},
		},
	})
}
