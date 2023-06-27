package anti_affinity_group_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	egoscale "github.com/exoscale/egoscale/v2"

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
		CheckDestroy:      testutils.CheckAntiAffinityGroupDestroy(&res),
		Steps: []resource.TestStep{
			{
				// Create
				Config: rConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testutils.CheckAntiAffinityGroupExists(r, &res),
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
