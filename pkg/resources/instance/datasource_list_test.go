package instance_test

import (
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func testListDataSource(t *testing.T) {
	t.Parallel()

	var (
		testdataSpec = testutils.TestdataSpec{
			ID:   time.Now().UnixNano(),
			Zone: "at-vie-2",
		}
		name         = testutils.ResourceName(testdataSpec.ID)
		reverseDNS   = "tf-provider-rdns-list-test.exoscale.com."
		byName       = "data.exoscale_compute_instance_list.by_name"
		byReverseDNS = "data.exoscale_compute_instance_list.by_reverse_dns"
		byIDAndState = "data.exoscale_compute_instance_list.by_id_and_state"
		byNameRegex  = "data.exoscale_compute_instance_list.by_name_regex"
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1 Create the instance. The API attaches the default Security Group
			// to it, which the configuration does not list.
			{
				Config:             testutils.ParseTestdataConfig("./testdata/010.datasource_list_create.tf.tmpl", &testdataSpec),
				ExpectNonEmptyPlan: true,
			},

			// 2 The zone is required.
			{
				Config:      testutils.ParseTestdataConfig("./testdata/011.datasource_list_missing_zone.tf.tmpl", &testdataSpec),
				ExpectError: regexp.MustCompile("Missing required argument"),
			},

			// 3 Filter by name, reverse DNS + labels, id + state, name regex + disk size.
			{
				Config: testutils.ParseTestdataConfig("./testdata/012.datasource_list.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(byName, "instances.#", "1"),
					resource.TestCheckResourceAttrPair(byName, "instances.0.id", instanceResource, "id"),
					resource.TestCheckResourceAttr(byName, "instances.0.name", name),
					resource.TestCheckResourceAttr(byName, "instances.0.type", "standard.tiny"),
					resource.TestCheckResourceAttrPair(byName, "instances.0.ssh_key", "exoscale_ssh_key.test_key", "name"),
					resource.TestCheckResourceAttr(byName, "instances.0.disk_size", "10"),
					resource.TestCheckResourceAttr(byName, "instances.0.reverse_dns", reverseDNS),

					resource.TestCheckResourceAttr(byReverseDNS, "instances.#", "1"),
					resource.TestCheckResourceAttr(byReverseDNS, "instances.0.name", name),

					resource.TestCheckResourceAttr(byIDAndState, "instances.#", "1"),
					resource.TestCheckResourceAttrPair(byIDAndState, "id", instanceResource, "id"),

					resource.TestCheckResourceAttr(byNameRegex, "instances.#", "1"),
					resource.TestCheckResourceAttr(byNameRegex, "instances.0.name", name),
				),
			},
		},
	})
}
