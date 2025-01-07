package sos_bucket_policy_test

import (
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/exoscale/testutils"
)

func TestSOSBucketPolicy(t *testing.T) {
	policyResourceName := "exoscale_sos_bucket_policy.test_policy"
	policyDataSourceName := "data." + policyResourceName

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}

	confVars := config.Variables{
		"exoscale_api_key":    config.StringVariable(os.Getenv("EXOSCALE_API_KEY")),
		"exoscale_api_secret": config.StringVariable(os.Getenv("EXOSCALE_API_SECRET")),
	}

	jsonPolicyTpl := "{\n  \"default-service-strategy\": \"%s\",\n  \"services\": {\n    \"sos\": {\n      \"type\": \"allow\"\n    }\n  }\n}\n"
	jsonPolicy := fmt.Sprintf(jsonPolicyTpl, "allow")
	jsonPolicyDeny := fmt.Sprintf(jsonPolicyTpl, "deny")
	wsRegex := regexp.MustCompile(`\s+`)
	jsonPolicyNoWS := wsRegex.ReplaceAllString(jsonPolicy, "")
	jsonPolicyDenyNoWS := wsRegex.ReplaceAllString(jsonPolicyDeny, "")

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
						jsonPolicy,
					),
					resource.TestCheckResourceAttr(policyDataSourceName, "policy", jsonPolicyNoWS),
				),
			},
			// 2 Update policy
			{
				ConfigVariables: confVars,
				Config:          testutils.ParseTestdataConfig("./testdata/002.policy_update.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						policyResourceName,
						"bucket",
						fmt.Sprintf("terraform-provider-test-%d", testdataSpec.ID),
					),
					resource.TestCheckResourceAttr(
						policyResourceName,
						"policy",
						jsonPolicyDeny,
					),
					resource.TestCheckResourceAttr(policyDataSourceName, "policy", jsonPolicyDenyNoWS),
				),
			},
			// Import
			{
				ConfigVariables: confVars,
				ResourceName:    policyResourceName,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(s *terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", s.RootModule().Resources[policyResourceName].Primary.Attributes["bucket"], testdataSpec.Zone), nil
					}
				}(),
				ImportState: true,
			},
		},
	})
}
