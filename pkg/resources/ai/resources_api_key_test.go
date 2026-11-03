package ai_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

func TestAIAPIKey(t *testing.T) {
	t.Parallel()

	r := "exoscale_ai_api_key.test"

	// The test only grants access to public models: no deployment (and no
	// GPU) is needed.
	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testAIZone,
	}
	name := testutils.ResourceName(testdataSpec.ID)

	var created, updated exoscale.GetAIAPIKeyResponse

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             checkAIAPIKeyDestroy(testAIZone, &created),
		Steps: []resource.TestStep{
			{ // Create
				Config: testutils.ParseTestdataConfig("./testdata/201.ai_api_key_all_models.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAIAPIKeyExists(r, &created),
					resource.TestCheckResourceAttrPtr(r, "id", (*string)(&created.ID)),
					resource.TestCheckResourceAttr(r, "zone", testAIZone),
					resource.TestCheckResourceAttr(r, "name", name),
					resource.TestCheckResourceAttr(r, "all_models", "true"),
					resource.TestCheckResourceAttr(r, "all_deployments", "false"),
					resource.TestCheckNoResourceAttr(r, "models"),
					resource.TestCheckNoResourceAttr(r, "deployment_ids"),
					resource.TestCheckResourceAttrSet(r, "secret"),
					resource.TestCheckResourceAttrSet(r, "created_at"),
					resource.TestCheckResourceAttrSet(r, "updated_at"),
				),
			},
			{ // Update
				// From all the public models to an allowlist.
				Config: testutils.ParseTestdataConfig("./testdata/202.ai_api_key_models.tf.tmpl", &testdataSpec),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(r, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAIAPIKeyExists(r, &updated),
					resource.TestCheckResourceAttr(r, "all_models", "false"),
					resource.TestCheckResourceAttr(r, "models.#", "1"),
					resource.TestCheckTypeSetElemAttr(r, "models.*", "Qwen/Qwen3-0.6B"),
					// The secret is only returned at creation: it must be kept.
					resource.TestCheckResourceAttrSet(r, "secret"),
					func(*terraform.State) error {
						if updated.ID != created.ID {
							return errors.New("AI API key was replaced instead of updated")
						}
						return nil
					},
				),
			},
			{ // Import
				// The API never returns the secret after the creation.
				ResourceName: r,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf("%s@%s", s.RootModule().Resources[r].Primary.ID, testAIZone), nil
				},
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret", "timeouts"},
			},
		},
	})
}

func checkAIAPIKeyExists(r string, apiKey *exoscale.GetAIAPIKeyResponse) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		defaultClient, err := testutils.APIClientV3()
		if err != nil {
			return err
		}

		ctx := context.Background()
		client, err := utils.SwitchClientZone(ctx, defaultClient, exoscale.ZoneName(rs.Primary.Attributes["zone"]))
		if err != nil {
			return err
		}

		res, err := client.GetAIAPIKey(ctx, exoscale.UUID(rs.Primary.ID))
		if err != nil {
			return err
		}

		if res.RevokedAT != nil {
			return fmt.Errorf("AI API key %q is revoked", res.ID)
		}

		*apiKey = *res
		return nil
	}
}

// checkAIAPIKeyDestroy checks that the key is revoked: the API has no delete,
// a revoked key may still be returned with its revocation date.
func checkAIAPIKeyDestroy(zone string, apiKey *exoscale.GetAIAPIKeyResponse) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if apiKey.ID == "" {
			return errors.New("AI API key ID was not captured during the test")
		}

		defaultClient, err := testutils.APIClientV3()
		if err != nil {
			return err
		}

		ctx := context.Background()
		client, err := utils.SwitchClientZone(ctx, defaultClient, exoscale.ZoneName(zone))
		if err != nil {
			return err
		}

		res, err := client.GetAIAPIKey(ctx, apiKey.ID)
		if errors.Is(err, exoscale.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}

		if res.RevokedAT != nil {
			return nil
		}

		return fmt.Errorf("AI API key %q is still active", apiKey.ID)
	}
}
