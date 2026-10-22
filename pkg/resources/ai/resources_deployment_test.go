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

// aiDeploymentTestdataSpec adds the deployment settings to the testdata templates.
type aiDeploymentTestdataSpec struct {
	testutils.TestdataSpec
	Name      string
	GpuType   string
	Parameter string
}

func TestAIDeployment(t *testing.T) {
	t.Skip("AI acc test disabled, waiting for an API change")
	t.Parallel()

	r := "exoscale_ai_deployment.test"

	testdataSpecCreate := aiDeploymentTestdataSpec{
		TestdataSpec: testutils.TestdataSpec{
			ID:   time.Now().UnixNano(),
			Zone: testAIZone,
		},
		GpuType:   "gpu3080ti",
		Parameter: "--max-model-len=512",
	}
	testdataSpecCreate.Name = testutils.ResourceName(testdataSpecCreate.ID)

	testdataSpecUpdate := testdataSpecCreate
	testdataSpecUpdate.Name = testdataSpecCreate.Name + "-updated"
	testdataSpecUpdate.Parameter = "--max-model-len=1024"

	var created, updated exoscale.GetDeploymentResponse

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             checkAIDeploymentDestroy(testAIZone, &created),
		Steps: []resource.TestStep{
			{ // Create
				Config: testutils.ParseTestdataConfig("./testdata/101.ai_deployment.tf.tmpl", &testdataSpecCreate),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAIDeploymentExists(r, &created),
					resource.TestCheckResourceAttrPtr(r, "id", (*string)(&created.ID)),
					resource.TestCheckResourceAttr(r, "zone", testAIZone),
					resource.TestCheckResourceAttr(r, "name", testdataSpecCreate.Name),
					resource.TestCheckResourceAttrPair(r, "model_id", "exoscale_ai_model.test", "id"),
					resource.TestCheckResourceAttr(r, "gpu_type", testdataSpecCreate.GpuType),
					resource.TestCheckResourceAttr(r, "gpu_count", "1"),
					resource.TestCheckResourceAttr(r, "replicas", "1"),
					resource.TestCheckResourceAttr(r, "inference_engine_parameters.#", "1"),
					resource.TestCheckTypeSetElemAttr(r, "inference_engine_parameters.*", testdataSpecCreate.Parameter),
					resource.TestCheckResourceAttrSet(r, "inference_engine_version"),
					resource.TestCheckResourceAttrSet(r, "deployment_url"),
					resource.TestCheckResourceAttrSet(r, "state"),
					resource.TestCheckResourceAttrSet(r, "created_at"),
				),
			},
			{ // Update
				Config: testutils.ParseTestdataConfig("./testdata/101.ai_deployment.tf.tmpl", &testdataSpecUpdate),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(r, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAIDeploymentExists(r, &updated),
					resource.TestCheckResourceAttr(r, "name", testdataSpecUpdate.Name),
					resource.TestCheckResourceAttr(r, "inference_engine_parameters.#", "1"),
					resource.TestCheckTypeSetElemAttr(r, "inference_engine_parameters.*", testdataSpecUpdate.Parameter),
					func(*terraform.State) error {
						if updated.ID != created.ID {
							return errors.New("AI deployment was replaced instead of updated")
						}
						return nil
					},
				),
			},
			{ // Import
				ResourceName: r,
				ImportState:  true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return fmt.Sprintf("%s@%s", s.RootModule().Resources[r].Primary.ID, testAIZone), nil
				},
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts", "state", "state_details", "updated_at"},
			},
		},
	})
}

func checkAIDeploymentExists(r string, deployment *exoscale.GetDeploymentResponse) resource.TestCheckFunc {
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

		res, err := client.GetDeployment(ctx, exoscale.UUID(rs.Primary.ID))
		if err != nil {
			return err
		}

		*deployment = *res
		return nil
	}
}

func checkAIDeploymentDestroy(zone string, deployment *exoscale.GetDeploymentResponse) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if deployment.ID == "" {
			return errors.New("AI deployment ID was not captured during the test")
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

		_, err = client.GetDeployment(ctx, deployment.ID)
		if errors.Is(err, exoscale.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}

		return fmt.Errorf("AI deployment %q still exists", deployment.ID)
	}
}
