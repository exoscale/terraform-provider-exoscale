package iam_test

import (
	"testing"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testDataSourceOrgPolicy(t *testing.T) {
	fullResourceName := "data.exoscale_iam_org_policy.test"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Since IAM Organization policy has the power to lock us out of testing account,
			// we will only do basic check for this resource.
			{
				Config: `data "exoscale_iam_org_policy" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullResourceName, "default_service_strategy"),
				),
			},
		},
	})
}
