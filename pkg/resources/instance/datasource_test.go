package instance_test

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

	var (
		testdataSpec = testutils.TestdataSpec{
			ID:   time.Now().UnixNano(),
			Zone: testutils.TestZoneName,
		}
		byID   = "data.exoscale_compute_instance.by_id"
		byName = "data.exoscale_compute_instance.by_name"
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1 Neither id nor name.
			{
				Config:      testutils.ParseTestdataConfig("./testdata/008.datasource_missing_lookup.tf.tmpl", &testdataSpec),
				ExpectError: regexp.MustCompile("either name or id must be specified"),
			},

			// 2 Match by id and by name.
			{
				Config: testutils.ParseTestdataConfig("./testdata/009.datasource.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(byID, "id", instanceResource, "id"),
					resource.TestCheckResourceAttr(byID, "name", testutils.ResourceName(testdataSpec.ID)),
					resource.TestCheckResourceAttr(byID, "type", "standard.tiny"),
					resource.TestCheckResourceAttr(byID, "disk_size", "10"),
					resource.TestCheckResourceAttr(byID, "state", "running"),
					resource.TestCheckResourceAttr(byID, "ipv6", "true"),
					resource.TestCheckResourceAttrPair(byID, "ipv6_address", instanceResource, "ipv6_address"),
					resource.TestCheckResourceAttrPair(byID, "public_ip_address", instanceResource, "public_ip_address"),
					resource.TestCheckResourceAttrPair(byID, "template_id", instanceResource, "template_id"),
					resource.TestCheckResourceAttr(byID, "user_data", fmt.Sprintf("user-data-%d", testdataSpec.ID)),
					resource.TestCheckResourceAttr(byID, "labels.test", fmt.Sprintf("label-%d", testdataSpec.ID)),
					resource.TestCheckResourceAttr(byID, "anti_affinity_group_ids.#", "1"),
					resource.TestCheckResourceAttr(byID, "elastic_ip_ids.#", "1"),
					resource.TestCheckResourceAttr(byID, "private_network_ids.#", "1"),
					resource.TestCheckResourceAttr(byID, "security_group_ids.#", "1"),
					resource.TestCheckResourceAttrPair(byID, "ssh_key", "exoscale_ssh_key.test_key", "name"),
					resource.TestCheckResourceAttr(byID, "reverse_dns", "tf-provider-rdns-test.exoscale.com"),

					resource.TestCheckResourceAttrPair(byName, "id", instanceResource, "id"),
					resource.TestCheckResourceAttr(byName, "name", testutils.ResourceName(testdataSpec.ID)),
					resource.TestCheckResourceAttr(byName, "reverse_dns", "tf-provider-rdns-test.exoscale.com"),
				),
			},
		},
	})
}
