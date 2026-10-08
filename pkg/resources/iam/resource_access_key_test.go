package iam_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	exoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func testResourceAccessKey(t *testing.T) {
	t.Parallel()

	r := "exoscale_iam_access_key.test"

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}
	name := testutils.ResourceName(testdataSpec.ID)

	var created, replaced exoscale.IAMAccessKey

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             checkAccessKeyDestroy(&created, &replaced),
		Steps: []resource.TestStep{
			{
				Config:      testutils.ParseTestdataConfig("./testdata/003.access_key_invalid_resource.tf.tmpl", &testdataSpec),
				ExpectError: regexp.MustCompile(`invalid IAM access key resource`),
			},
			{
				// The API returns the operations granted by the tags along
				// with the configured ones: the re-plan must be empty.
				Config: testutils.ParseTestdataConfig("./testdata/001.access_key_create.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAccessKeyExists(r, &created),
					resource.TestCheckResourceAttrPair(r, "id", r, "key"),
					resource.TestCheckResourceAttr(r, "name", name),
					resource.TestCheckTypeSetElemAttr(r, "operations.*", "list-instances"),
					resource.TestCheckResourceAttr(r, "resources.#", "1"),
					resource.TestCheckTypeSetElemAttr(r, "resources.*", fmt.Sprintf("sos/bucket:%s", name)),
					resource.TestCheckResourceAttr(r, "tags.#", "1"),
					resource.TestCheckTypeSetElemAttr(r, "tags.*", "sos"),
					resource.TestCheckResourceAttrWith(r, "tags_operations.#", notZero),
					resource.TestCheckResourceAttrSet(r, "secret"),
				),
			},
			{
				Config: testutils.ParseTestdataConfig("./testdata/002.access_key_replace.tf.tmpl", &testdataSpec),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(r, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkAccessKeyExists(r, &replaced),
					resource.TestCheckResourceAttr(r, "name", name+"-replaced"),
					resource.TestCheckResourceAttrWith(r, "operations.#", notZero),
					resource.TestCheckNoResourceAttr(r, "resources"),
					resource.TestCheckTypeSetElemAttr(r, "tags.*", "sos"),
					resource.TestCheckResourceAttrSet(r, "secret"),
					func(*terraform.State) error {
						if *replaced.Key == *created.Key {
							return errors.New("IAM access key was not replaced")
						}
						return nil
					},
				),
			},
			{
				// The secret is only returned on creation.
				ResourceName:            r,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"secret", "timeouts"},
			},
		},
	})
}

func checkAccessKeyExists(r string, accessKey *exoscale.IAMAccessKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		client, err := testutils.APIClient()
		if err != nil {
			return err
		}

		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testutils.TestEnvironment(), config.DefaultZone),
		)
		res, err := client.GetIAMAccessKey(ctx, config.DefaultZone, rs.Primary.ID)
		if err != nil {
			return err
		}

		*accessKey = *res
		return nil
	}
}

func checkAccessKeyDestroy(accessKeys ...*exoscale.IAMAccessKey) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := testutils.APIClient()
		if err != nil {
			return err
		}

		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testutils.TestEnvironment(), config.DefaultZone),
		)

		for _, accessKey := range accessKeys {
			if accessKey.Key == nil {
				continue
			}

			_, err := client.GetIAMAccessKey(ctx, config.DefaultZone, *accessKey.Key)
			if errors.Is(err, exoapi.ErrNotFound) {
				continue
			}
			if err != nil {
				return err
			}

			return fmt.Errorf("IAM access key %q still exists", *accessKey.Key)
		}

		return nil
	}
}

func notZero(v string) error {
	if v == "0" {
		return errors.New("expected a non-empty set")
	}
	return nil
}
