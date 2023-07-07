package exoscale

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
)

var (
	testAccResourceIAMAccessKeyName = acctest.RandomWithPrefix(testPrefix)

	testAccResourceIAMAccessKeyConfigCreate = fmt.Sprintf(`
resource "exoscale_iam_access_key" "test" {
  name       = "%s"

  operations = ["list-instances"]
  resources  = ["sos/bucket:eat-terraform-provider-test"]
  tags       = ["sos"]
}
`,
		testAccResourceIAMAccessKeyName,
	)
)

func TestAccResourceIAMAccessKey(t *testing.T) {
	var (
		r            = "exoscale_iam_access_key.test"
		iamAccessKey egoscale.IAMAccessKey
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceIAMAccessKeyDestroy(&iamAccessKey),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceIAMAccessKeyConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceIAMAccessKeyExists(r, &iamAccessKey),
					func(s *terraform.State) error {
						a := assert.New(t)
						a.Equal(testAccResourceIAMAccessKeyName, *iamAccessKey.Name)
						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resIAMAccessKeyAttrKey:  validation.ToDiagFunc(validation.NoZeroValues),
						resIAMAccessKeyAttrName: validateString(testAccResourceIAMAccessKeyName),
						/*						resIAMAccessKeyAttrOperations:
												resIAMAccessKeyAttrResources:*/
						resIAMAccessKeyAttrSecret: validation.ToDiagFunc(validation.NoZeroValues),
						/*						resIAMAccessKeyAttrLabels:*/

					})),
				),
			},
		},
	})
}

func testAccCheckResourceIAMAccessKeyExists(r string, iamAccessKey *egoscale.IAMAccessKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client := GetComputeClient(testAccProvider.Meta())

		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testEnvironment, testZoneName))
		res, err := client.GetIAMAccessKey(ctx, testZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*iamAccessKey = *res
		return nil
	}
}

func testAccCheckResourceIAMAccessKeyDestroy(iamAccessKey *egoscale.IAMAccessKey) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testEnvironment, testZoneName))

		_, err := client.GetIAMAccessKey(ctx, testZoneName, *iamAccessKey.Key)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("Access Key still exists")
	}
}
