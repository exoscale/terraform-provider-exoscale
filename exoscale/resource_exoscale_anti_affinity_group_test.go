package exoscale

import (
	"context"
	"errors"
	"fmt"
	"testing"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
)

var (
	testAccResourceAntiAffinityGroupName        = acctest.RandomWithPrefix(testPrefix)
	testAccResourceAntiAffinityGroupDescription = acctest.RandString(10)

	testAccResourceAntiAffinityGroupConfigCreate = fmt.Sprintf(`
resource "exoscale_anti_affinity_group" "test" {
  name        = "%s"
  description = "%s"
}
`,
		testAccResourceAntiAffinityGroupName,
		testAccResourceAntiAffinityGroupDescription,
	)
)

func TestAccResourceAntiAffinityGroup(t *testing.T) {
	var (
		r                 = "exoscale_anti_affinity_group.test"
		antiAffinityGroup egoscale.AntiAffinityGroup
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceAntiAffinityGroupDestroy(&antiAffinityGroup),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourceAntiAffinityGroupConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceAntiAffinityGroupExists(r, &antiAffinityGroup),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(testAccResourceAntiAffinityGroupDescription, *antiAffinityGroup.Description)
						a.Equal(testAccResourceAntiAffinityGroupName, *antiAffinityGroup.Name)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resAntiAffinityGroupAttrDescription: validateString(testAccResourceAntiAffinityGroupDescription),
						resAntiAffinityGroupAttrName:        validateString(testAccResourceAntiAffinityGroupName),
					})),
				),
			},
			{
				// Import
				ResourceName:      r,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resAntiAffinityGroupAttrDescription: validateString(testAccResourceAntiAffinityGroupDescription),
							resAntiAffinityGroupAttrName:        validateString(testAccResourceAntiAffinityGroupName),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceAntiAffinityGroupExists(r string, antiAffinityGroup *egoscale.AntiAffinityGroup) resource.TestCheckFunc {
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
		res, err := client.GetAntiAffinityGroup(ctx, testZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*antiAffinityGroup = *res
		return nil
	}
}

func testAccCheckResourceAntiAffinityGroupDestroy(antiAffinityGroup *egoscale.AntiAffinityGroup) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testEnvironment, testZoneName))

		_, err := client.GetAntiAffinityGroup(ctx, testZoneName, *antiAffinityGroup.ID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("Anti-Affinity Group still exists")
	}
}
