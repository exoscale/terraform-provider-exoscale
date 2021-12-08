package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var testAccDataSourceAntiAffinityGroupName = acctest.RandomWithPrefix(testPrefix)

func TestAccDataSourceAntiAffinityGroup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
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
}`, testAccDataSourceAntiAffinityGroupName),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAntiAffinityGroupAttributes("data.exoscale_anti_affinity_group.by-id", testAttrs{
						dsAntiAffinityGroupAttrID:   validation.ToDiagFunc(validation.IsUUID),
						dsAntiAffinityGroupAttrName: validateString(testAccDataSourceAntiAffinityGroupName),
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
}`, testAccDataSourceAntiAffinityGroupName),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAntiAffinityGroupAttributes("data.exoscale_anti_affinity_group.by-name", testAttrs{
						dsAntiAffinityGroupAttrID:   validation.ToDiagFunc(validation.IsUUID),
						dsAntiAffinityGroupAttrName: validateString(testAccDataSourceAntiAffinityGroupName),
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
					testAccDataSourceAntiAffinityGroupName,
					testZoneName,
					testInstanceTemplateName,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceAntiAffinityGroupAttributes("data.exoscale_anti_affinity_group.test", testAttrs{
						dsAntiAffinityGroupAttrID:               validation.ToDiagFunc(validation.IsUUID),
						dsAntiAffinityGroupAttrInstances + ".#": validateString("1"),
						dsAntiAffinityGroupAttrName:             validateString(testAccDataSourceAntiAffinityGroupName),
					}),
				),
			},
		},
	})
}

func testAccDataSourceAntiAffinityGroupAttributes(ds string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return checkResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("exoscale_anti_affinity_group data source not found in the state")
	}
}
