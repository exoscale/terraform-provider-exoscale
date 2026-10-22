package ai_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

// testAIZone is a zone where the AI service is available.
const testAIZone = "at-vie-2"

// aiModelTestdataSpec adds the model name to the testdata templates.
type aiModelTestdataSpec struct {
	testutils.TestdataSpec
	Name string
}

func TestAIModel(t *testing.T) {
	t.Skip("AI acc test disabled, waiting for an API change")
	t.Parallel()

	r := "exoscale_ai_model.test"

	// Tiny public (ungated) safetensors models: a few MB each, to keep the
	// download and the storage cost of the test negligible.
	createdName := "hf-internal-testing/tiny-random-LlamaForCausalLM"
	replacedName := "trl-internal-testing/tiny-Qwen2ForCausalLM-2.5"

	testdataSpecCreate := aiModelTestdataSpec{
		TestdataSpec: testutils.TestdataSpec{
			ID:   0, // The id will not be used because it is bound to an external resource (the hugging face model name).
			Zone: testAIZone,
		},
		Name: createdName,
	}

	testdataSpecUpdate := testdataSpecCreate
	testdataSpecUpdate.Name = replacedName

	var created, replaced exoscale.GetModelResponse

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             checkAIModelDestroy(testAIZone, &created, &replaced),
		Steps: []resource.TestStep{
			{
				Config: testutils.ParseTestdataConfig("./testdata/001.ai_model_missing_name.tf.tmpl", &aiModelTestdataSpec{
					TestdataSpec: testutils.TestdataSpec{
						ID:   time.Now().UnixNano(),
						Zone: testAIZone,
					},
				}),
				ExpectError: regexp.MustCompile(`The argument "name" is required`),
			},
			{
				Config: testutils.ParseTestdataConfig("./testdata/002.ai_model.tf.tmpl", &testdataSpecCreate),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAIModelExists(r, &created),
					resource.TestCheckResourceAttrPtr(r, "id", (*string)(&created.ID)),
					resource.TestCheckResourceAttr(r, "zone", testAIZone),
					resource.TestCheckResourceAttr(r, "name", createdName),
					resource.TestCheckNoResourceAttr(r, "huggingface_token"),
					resource.TestCheckResourceAttrSet(r, "state"),
					resource.TestCheckResourceAttrSet(r, "created_at"),
					resource.TestCheckResourceAttrSet(r, "updated_at"),
				),
			},
			{
				// A new name replaces the model.
				Config: testutils.ParseTestdataConfig("./testdata/002.ai_model.tf.tmpl", &testdataSpecUpdate),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(r, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAIModelExists(r, &replaced),
					resource.TestCheckResourceAttrPtr(r, "id", (*string)(&replaced.ID)),
					resource.TestCheckResourceAttr(r, "name", replacedName),
					func(*terraform.State) error {
						if replaced.ID == created.ID {
							return errors.New("AI model was not replaced")
						}
						return nil
					},
				),
			},
			{
				// The API doesn't return the Hugging Face token, and the model
				// may still be downloading: its state, size and update date
				// can change between the last refresh and the import.
				ResourceName: r,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf("%s@%s", s.RootModule().Resources[r].Primary.ID, testAIZone), nil
				},
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"huggingface_token", "timeouts", "state", "model_size", "updated_at"},
			},
		},
	})
}

func checkAIModelExists(r string, model *exoscale.GetModelResponse) resource.TestCheckFunc {
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

		res, err := client.GetModel(ctx, exoscale.UUID(rs.Primary.ID))
		if err != nil {
			return err
		}

		*model = *res
		return nil
	}
}

func checkAIModelDestroy(zone string, models ...*exoscale.GetModelResponse) resource.TestCheckFunc {
	return func(*terraform.State) error {
		defaultClient, err := testutils.APIClientV3()
		if err != nil {
			return err
		}

		ctx := context.Background()
		client, err := utils.SwitchClientZone(ctx, defaultClient, exoscale.ZoneName(zone))
		if err != nil {
			return err
		}

		for _, model := range models {
			if model.ID == "" {
				return errors.New("AI model ID was not captured during the test")
			}

			_, err := client.GetModel(ctx, model.ID)
			if errors.Is(err, exoscale.ErrNotFound) {
				continue
			}
			if err != nil {
				return err
			}

			return fmt.Errorf("AI model %q still exists", model.ID)
		}

		return nil
	}
}
