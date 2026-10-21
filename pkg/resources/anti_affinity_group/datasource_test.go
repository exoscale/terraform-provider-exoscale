package anti_affinity_group_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func testDataSource(t *testing.T) {
	t.Parallel()

	r := "exoscale_anti_affinity_group.test"
	dsByID := "data.exoscale_anti_affinity_group.by_id"
	dsByName := "data.exoscale_anti_affinity_group.by_name"

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testutils.ParseTestdataConfig("./testdata/003.datasource_missing_lookup.tf.tmpl", &testdataSpec),
				ExpectError: regexp.MustCompile(`Invalid Attribute Combination`),
			},
			{
				Config: testutils.ParseTestdataConfig("./testdata/004.datasource.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dsByID, "id", r, "id"),
					resource.TestCheckResourceAttrPair(dsByID, "name", r, "name"),
					resource.TestCheckResourceAttr(dsByID, "instances.#", "0"),
					resource.TestCheckResourceAttrPair(dsByName, "id", r, "id"),
					resource.TestCheckResourceAttrPair(dsByName, "name", r, "name"),
				),
			},
			{
				Config: testutils.ParseTestdataConfig("./testdata/005.datasource_instance.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(dsByID, "id", r, "id"),
					resource.TestCheckResourceAttr(dsByID, "instances.#", "1"),
					resource.TestCheckTypeSetElemAttrPair(dsByID, "instances.*", "exoscale_compute_instance.test", "id"),
					resource.TestCheckResourceAttrPair(dsByName, "id", r, "id"),
					resource.TestCheckResourceAttr(dsByName, "instances.#", "1"),
				),
			},
		},
	})
}
