package instance_pool_test

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func testListDataSource(t *testing.T) {
	t.Parallel()

	var (
		testdataSpec = testutils.TestdataSpec{
			ID:   time.Now().UnixNano(),
			Zone: testutils.TestZoneName,
		}
		name = testutils.ResourceName(testdataSpec.ID)
		ds   = "data.exoscale_instance_pool_list.test"
	)

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1 Create the pools.
			{
				Config: testutils.ParseTestdataConfig("./testdata/006.datasource_list_create.tf.tmpl", &testdataSpec),
			},

			// 2 The zone is required.
			{
				Config:      testutils.ParseTestdataConfig("./testdata/007.datasource_list_missing_zone.tf.tmpl", &testdataSpec),
				ExpectError: regexp.MustCompile("Missing required argument"),
			},

			// 3 List the pools. The values a pool does not have are empty
			// rather than null, as with the SDKv2.
			{
				Config: testutils.ParseTestdataConfig("./testdata/008.datasource_list.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(ds, "id"),
					checkListedPool(ds, "exoscale_instance_pool.minimal", map[string]string{
						"name":                      name + "-minimal",
						"zone":                      testdataSpec.Zone,
						"description":               "",
						"deploy_target_id":          "",
						"disk_size":                 "10",
						"instance_prefix":           "pool",
						"instance_type":             "standard.tiny",
						"ipv6":                      "false",
						"key_pair":                  "",
						"size":                      "1",
						"user_data":                 "",
						"labels.%":                  "0",
						"affinity_group_ids.#":      "0",
						"anti_affinity_group_ids.#": "0",
						"elastic_ip_ids.#":          "0",
						"network_ids.#":             "0",
						"security_group_ids.#":      "1",
						"instances.#":               "1",
					}),
					checkListedPool(ds, "exoscale_instance_pool.labeled", map[string]string{
						"name":        name + "-labeled",
						"description": fmt.Sprintf("description-%d", testdataSpec.ID),
						"user_data":   fmt.Sprintf("user-data-%d", testdataSpec.ID),
						"labels.%":    "1",
						"labels.test": fmt.Sprintf("label-%d", testdataSpec.ID),
						"instances.#": "1",
					}),
				),
			},
		},
	})
}

// checkListedPool checks the attributes of the element of `pools` matching the
// given resource. The zone holds the pools of other tests too.
func checkListedPool(ds, res string, expected map[string]string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		id, err := testutils.AttrFromState(s, res, "id")
		if err != nil {
			return err
		}

		list, ok := s.RootModule().Resources[ds]
		if !ok {
			return fmt.Errorf("%s not found in the state", ds)
		}
		attrs := list.Primary.Attributes

		count, err := strconv.Atoi(attrs["pools.#"])
		if err != nil {
			return fmt.Errorf("unable to read the number of pools: %w", err)
		}

		for i := range count {
			prefix := fmt.Sprintf("pools.%d.", i)
			if attrs[prefix+"id"] != id {
				continue
			}

			for k, want := range expected {
				got, ok := attrs[prefix+k]
				if !ok {
					return fmt.Errorf("%s: %s%s is not set, expected %q", ds, prefix, k, want)
				}
				if got != want {
					return fmt.Errorf("%s: %s%s is %q, expected %q", ds, prefix, k, got, want)
				}
			}

			return nil
		}

		return fmt.Errorf("%s: instance pool %s not found", ds, id)
	}
}
