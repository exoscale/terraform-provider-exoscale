package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	testAccDataSourceComputeTemplateID       = testInstanceTemplateID
	testAccDataSourceComputeTemplateName     = testInstanceTemplateName
	testAccDataSourceComputeTemplateUsername = testInstanceTemplateUsername
	testAccDataSourceComputeTemplateFilter   = testInstanceTemplateFilter
	testAccDataSourceTemplateZone            = testZoneName
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
					testAccDataSourceTemplateZone),
				ExpectError: regexp.MustCompile("either name or id must be specified"),
			},
			{
				Config: fmt.Sprintf(`
data "exoscale_compute_template" "test" {
  zone   = "%s"
  name   = "%s"
  filter = "%s"
}`,
					testAccDataSourceTemplateZone,
					testAccDataSourceComputeTemplateName,
					testAccDataSourceComputeTemplateFilter,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceComputeTemplateAttributes(testAttrs{
						"id":       ValidateString(testAccDataSourceComputeTemplateID),
						"name":     ValidateString(testAccDataSourceComputeTemplateName),
						"username": ValidateString(testAccDataSourceComputeTemplateUsername),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
data "exoscale_compute_template" "test" {
  zone   = "%s"
  id     = "%s"
  filter = "%s"
}`,
					testAccDataSourceTemplateZone,
					testAccDataSourceComputeTemplateID,
					testAccDataSourceComputeTemplateFilter,
				),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceComputeTemplateAttributes(testAttrs{
						"id":       ValidateString(testAccDataSourceComputeTemplateID),
						"name":     ValidateString(testAccDataSourceComputeTemplateName),
						"username": ValidateString(testAccDataSourceComputeTemplateUsername),
					}),
				),
			},
		},
	})
}

func testAccDataSourceComputeTemplateAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_compute_template" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("exoscale_compute_template data source not found in the state")
	}
}
