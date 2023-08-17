package list_test

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func TestFilterableListDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testutils.AccPreCheck(t) },
		ProviderFactories: testutils.Providers(),
		Steps: []resource.TestStep{
			{
				Config: `
data "exoscale_sks_nodepool_list" "my_sks_nodepool_list" {
  # we omit the zone to trigger an error as the zone attribute must be mandatory.
}
`,
				ExpectError: regexp.MustCompile("Missing required argument"),
			},
			{
				Config: `
data "exoscale_sks_cluster_list" "my_sks_cluster_list" {
  # we omit the zone to trigger an error as the zone attribute must be mandatory.
}
`,
				ExpectError: regexp.MustCompile("Missing required argument"),
			},
		},
	})
}
