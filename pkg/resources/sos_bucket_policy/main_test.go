package sos_bucket_policy_test

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func TestSOSBucketPolicy(t *testing.T) {
	policyResourceName := "exoscale_sos_bucket_policy.test_policy"
	// policyDataSourceName := "data." + policyResourceName

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}

	confVars := config.Variables{
		"exoscale_api_key":    config.StringVariable(os.Getenv("EXOSCALE_API_KEY")),
		"exoscale_api_secret": config.StringVariable(os.Getenv("EXOSCALE_API_SECRET")),
	}

	resource.Test(t, resource.TestCase{
		ExternalProviders: map[string]resource.ExternalProvider{
			"aws": resource.ExternalProvider{
				Source: "hashicorp/aws",
			},
		},
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1 Create policy
			{
				ConfigVariables: confVars,
				Config:          testutils.ParseTestdataConfig("./testdata/001.policy_create.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						policyResourceName,
						"bucket",
						fmt.Sprintf("terraform-provider-test-%d", testdataSpec.ID),
					),
					resource.TestCheckResourceAttr(
						policyResourceName,
						"policy",
						fmt.Sprintf("{\n  \"default-service-strategy\": \"allow\",\n  \"services\": {\n    \"sos\": {\n      \"type\": \"allow\"\n    }\n  }\n}\n"),
					),
				),
			},
		},
	})
}
