package exoscale

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var (
	testAccDataSourceComputeTemplateName     = testInstanceTemplateName
	testAccDataSourceComputeTemplateUsername = testInstanceTemplateUsername
	testAccDataSourceComputeTemplateFilter   = testInstanceTemplateFilter
	testAccDataSourceComputeTemplateZone     = testZoneName
)

func TestAccDataSourceComputeTemplate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
data "exoscale_compute_template" "test" {
  zone = "%s"
}`,
					testAccDataSourceComputeTemplateZone),
				ExpectError: regexp.MustCompile("either name or id must be specified"),
			},
			{
				Config: fmt.Sprintf(`
data "exoscale_compute_template" "by_name" {
  zone   = "%s"
  name   = "%s"
  filter = "%s"
}

data "exoscale_compute_template" "by_id" {
  zone   = data.exoscale_compute_template.by_name.zone
  id     = data.exoscale_compute_template.by_name.id
  filter = data.exoscale_compute_template.by_name.filter
}
`,
					testAccDataSourceComputeTemplateZone,
					testAccDataSourceComputeTemplateName,
					testAccDataSourceComputeTemplateFilter,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceComputeTemplateAttributes("by_name", testAttrs{
						"name":     validateString(testAccDataSourceComputeTemplateName),
						"username": validateString(testAccDataSourceComputeTemplateUsername),
					}),
					testAccDataSourceComputeTemplateAttributes("by_id", testAttrs{
						"name":     validateString(testAccDataSourceComputeTemplateName),
						"username": validateString(testAccDataSourceComputeTemplateUsername),
					}),
				),
			},
		},
	})
}

func testAccDataSourceComputeTemplateAttributes(name string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for resourceName, resourceState := range s.RootModule().Resources {
			if resourceState.Type != "exoscale_compute_template" || resourceName != "data.exoscale_compute_template."+name {
				continue
			}

			return checkResourceAttributes(expected, resourceState.Primary.Attributes)
		}

		return fmt.Errorf("exoscale_compute_template data source '%s' not found in the state", name)
	}
}
