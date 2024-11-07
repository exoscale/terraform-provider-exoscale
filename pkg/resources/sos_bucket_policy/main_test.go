package sos_bucket_policy_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func TestSOSBucketPolicy(t *testing.T) {
	policyResourceName := "exoscale_sos_bucket_policy.test_policy"
	// policyDataSourceName := "data." + policyResourceName

	testdataSpec := testutils.TestdataSpec{
		ID: time.Now().UnixNano(),
		// Zone: testutils.TestZoneName,
		Zone: "ch-gva-2",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1 Create policy
			{
				Config: testutils.ParseTestdataConfig("./testdata/001.policy_create.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						policyResourceName,
						"bucket",
						fmt.Sprintf("terraform-provider-test-%d", testdataSpec.ID),
					),
				),
			},
		},
	})
}
