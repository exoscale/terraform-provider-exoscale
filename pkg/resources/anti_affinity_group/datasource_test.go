package anti_affinity_group_test

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	aagroup "github.com/exoscale/terraform-provider-exoscale/pkg/resources/anti_affinity_group"
	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

var dsGroupName = acctest.RandomWithPrefix(testutils.Prefix)

func testDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testutils.AccPreCheck(t) },
		ProviderFactories: testutils.Providers(),
		Steps: []resource.TestStep{
			{
				Config:      `data "exoscale_anti_affinity_group" "test" {}`,
				ExpectError: regexp.MustCompile("either name or id must be specified"),
			},
			{
				Config: fmt.Sprintf(`
resource "exoscale_anti_affinity_group" "test" {
  name = "%s"
}

data "exoscale_anti_affinity_group" "by-id" {
  id = exoscale_anti_affinity_group.test.id
}`, dsGroupName),
				Check: resource.ComposeTestCheckFunc(
					testDataSourceAttributes("data.exoscale_anti_affinity_group.by-id", testutils.TestAttrs{
						aagroup.AttrID:   validation.ToDiagFunc(validation.IsUUID),
						aagroup.AttrName: testutils.ValidateString(dsGroupName),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "exoscale_anti_affinity_group" "test" {
  name = "%s"
}

data "exoscale_anti_affinity_group" "by-name" {
  name = exoscale_anti_affinity_group.test.name
}`, dsGroupName),
				Check: resource.ComposeTestCheckFunc(
					testDataSourceAttributes("data.exoscale_anti_affinity_group.by-name", testutils.TestAttrs{
						aagroup.AttrID:   validation.ToDiagFunc(validation.IsUUID),
						aagroup.AttrName: testutils.ValidateString(dsGroupName),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "exoscale_anti_affinity_group" "test" {
  name = "%s"
}

resource "exoscale_compute" "test" {
  zone               = "%s"
  template           = "%s"
  size               = "Micro"
  disk_size          = "10"
  affinity_group_ids = [exoscale_anti_affinity_group.test.id]
}

data "exoscale_anti_affinity_group" "test" {
  id         = exoscale_anti_affinity_group.test.id
  depends_on = [exoscale_compute.test]
}`,
					dsGroupName,
					testutils.TestZoneName,
					testutils.TestInstanceTemplateName,
				),
				Check: resource.ComposeTestCheckFunc(
					testDataSourceAttributes("data.exoscale_anti_affinity_group.test", testutils.TestAttrs{
						aagroup.AttrID:               validation.ToDiagFunc(validation.IsUUID),
						aagroup.AttrInstances + ".#": testutils.ValidateString("1"),
						aagroup.AttrName:             testutils.ValidateString(dsGroupName),
					}),
				),
			},
		},
	})
}

func testDataSourceAttributes(ds string, expected testutils.TestAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return testutils.CheckResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("exoscale_anti_affinity_group data source not found in the state")
	}
}
