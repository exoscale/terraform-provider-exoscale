package anti_affinity_group_test

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

	aagroup "github.com/exoscale/terraform-provider-exoscale/pkg/resources/anti_affinity_group"
	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

var (
	rGroupName        = acctest.RandomWithPrefix(testutils.TestZoneName)
	rGroupDescription = acctest.RandString(10)

	rConfigCreate = fmt.Sprintf(`
resource "exoscale_anti_affinity_group" "test" {
  name        = "%s"
  description = "%s"
}
`,
		rGroupName,
		rGroupDescription,
	)
)

func testResource(t *testing.T) {
	var (
		r   = aagroup.Name + ".test"
		res egoscale.AntiAffinityGroup
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testutils.AccPreCheck(t) },
		ProviderFactories: testutils.Providers(),
		CheckDestroy:      testGroupDestroy(&res),
		Steps: []resource.TestStep{
			{
				// Create
				Config: rConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					rExists(r, &res),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(rGroupDescription, *res.Description)
						a.Equal(rGroupName, *res.Name)

						return nil
					},
					testutils.CheckResourceState(r, testutils.CheckResourceStateValidateAttributes(testutils.TestAttrs{
						aagroup.AttrDescription: testutils.ValidateString(rGroupDescription),
						aagroup.AttrName:        testutils.ValidateString(rGroupName),
					})),
				),
			},
			{
				// Import
				ResourceName:      r,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return testutils.CheckResourceAttributes(
						testutils.TestAttrs{
							aagroup.AttrDescription: testutils.ValidateString(rGroupDescription),
							aagroup.AttrName:        testutils.ValidateString(rGroupName),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func rExists(r string, res *egoscale.AntiAffinityGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client, err := testutils.APIClient()
		if err != nil {
			return err
		}

		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testutils.TestEnvironment(), testutils.TestZoneName),
		)
		data, err := client.GetAntiAffinityGroup(ctx, testutils.TestZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*res = *data
		return nil
	}
}

func testGroupDestroy(res *egoscale.AntiAffinityGroup) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client, err := testutils.APIClient()
		if err != nil {
			return err
		}

		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testutils.TestEnvironment(), testutils.TestZoneName),
		)

		_, err = client.GetAntiAffinityGroup(ctx, testutils.TestZoneName, *res.ID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("Anti-Affinity Group still exists")
	}
}
