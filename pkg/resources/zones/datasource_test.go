package zones_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func TestGetZones(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "exoscale_zones" "example_zones" {
  id = "asdf"
}

# Outputs
output "zones_output" {
  value = data.exoscale_zones.example_zones.zones
}
`,
				// Check: resource.ComposeTestCheckFunc(
				// 	dsCheckAttrs("data.exoscale_instance_pool.by-id", testutils.TestAttrs{
				// 		"affinity_group_ids.#": testutils.ValidateString("1"),
				// 		"affinity_group_ids.0": validation.ToDiagFunc(validation.IsUUID),
				// 		"description":          testutils.ValidateString(dsDescription),
				// 		"disk_size":            testutils.ValidateString(dsDiskSize),
				// 		"instance_type":        utils.ValidateComputeInstanceType,
				// 		"instance_prefix":      testutils.ValidateString(dsInstancePrefix),
				// 		"key_pair":             testutils.ValidateString(dsKeyPair),
				// 		"labels.test":          testutils.ValidateString(dsLabelValue),
				// 		"id":                   validation.ToDiagFunc(validation.IsUUID),
				// 		"name":                 testutils.ValidateString(dsName),
				// 		"network_ids.#":        testutils.ValidateString("1"),
				// 		"network_ids.0":        validation.ToDiagFunc(validation.IsUUID),
				// 		"size":                 testutils.ValidateString(dsSize),
				// 		// NOTE: state is unreliable atm, improvement suggested in 54808
				// 		// "state":                testutils.ValidateString("running"),
				// 		"template_id":    validation.ToDiagFunc(validation.IsUUID),
				// 		"user_data":      testutils.ValidateString(dsUserData),
				// 		"instances.#":    testutils.ValidateString("2"),
				// 		"instances.0.id": validation.ToDiagFunc(validation.IsUUID),
				// 		"instances.1.id": validation.ToDiagFunc(validation.IsUUID),
				// 	}),
				// ),
			},
		},
	})
}
