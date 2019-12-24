package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

const (
	// Reference template used for tests: "Linux Ubuntu 18.04 LTS 64-bit" @ ch-gva-2 (featured)
	// cs --region cloudstack listTemplates \
	//     templatefilter=featured \
	//     zoneid=1128bd56-b4d9-4ac6-a7b9-c715b187ce11 \
	//     name="Linux Ubuntu 18.04 LTS 64-bit"
	dataSourceComputeTemplateID       = "287b6306-fdeb-4dc6-855d-90c4f68f572b"
	dataSourceComputeTemplateName     = "Linux Ubuntu 18.04 LTS 64-bit"
	dataSourceComputeTemplateUsername = "ubuntu"
)

func TestAccDataSourceComputeTemplate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:  func() { testAccPreCheck(t) },
		Providers: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config: `
data "exoscale_compute_template" "ubuntu_lts" {
  zone = "ch-gva-2"
}`,
				ExpectError: regexp.MustCompile("either name or id must be specified"),
			},
			{
				Config: fmt.Sprintf(`
data "exoscale_compute_template" "ubuntu_lts" {
  zone   = "ch-gva-2"
  name   = "%s"
  filter = "featured"
}`, dataSourceComputeTemplateName),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceComputeTemplateAttributes(testAttrs{
						"id":       ValidateString(dataSourceComputeTemplateID),
						"name":     ValidateString(dataSourceComputeTemplateName),
						"username": ValidateString(dataSourceComputeTemplateUsername),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
data "exoscale_compute_template" "ubuntu_lts" {
  zone   = "ch-gva-2"
  id     = "%s"
  filter = "featured"
}`, dataSourceComputeTemplateID),
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceComputeTemplateAttributes(testAttrs{
						"id":       ValidateString(dataSourceComputeTemplateID),
						"name":     ValidateString(dataSourceComputeTemplateName),
						"username": ValidateString(dataSourceComputeTemplateUsername),
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
