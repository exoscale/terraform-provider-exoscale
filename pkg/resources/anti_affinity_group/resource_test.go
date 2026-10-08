package anti_affinity_group_test

import (
	"errors"
	"testing"
	"time"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func testResource(t *testing.T) {
	t.Parallel()

	r := "exoscale_anti_affinity_group.test"
	rNoDescription := "exoscale_anti_affinity_group.test_no_description"

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}
	name := testutils.ResourceName(testdataSpec.ID)

	var created, replaced exoscale.AntiAffinityGroup

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             testutils.CheckAntiAffinityGroupDestroy(&replaced),
		Steps: []resource.TestStep{
			{
				Config: testutils.ParseTestdataConfig("./testdata/001.resource_create.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.CheckAntiAffinityGroupExistsV3(r, &created),
					resource.TestCheckResourceAttr(r, "name", name),
					resource.TestCheckResourceAttr(r, "description", "description-test"),
					resource.TestCheckResourceAttr(rNoDescription, "name", name+"-no-description"),
					resource.TestCheckNoResourceAttr(rNoDescription, "description"),
				),
			},
			{
				// A new description replaces the group, setting an empty one
				// on a group without description doesn't.
				Config: testutils.ParseTestdataConfig("./testdata/002.resource_replace.tf.tmpl", &testdataSpec),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(r, plancheck.ResourceActionReplace),
						plancheck.ExpectResourceAction(rNoDescription, plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					testutils.CheckAntiAffinityGroupExistsV3(r, &replaced),
					func(*terraform.State) error {
						if replaced.ID == created.ID {
							return errors.New("anti-affinity group was not replaced")
						}
						return nil
					},
					testutils.CheckAntiAffinityGroupDestroy(&created),
					resource.TestCheckResourceAttr(r, "description", "description-test-updated"),
					resource.TestCheckResourceAttr(rNoDescription, "description", ""),
				),
			},
			{
				ResourceName:            r,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"timeouts"},
			},
		},
	})
}
