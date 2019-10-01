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
	// Reference template used for tests: "Linux Ubuntu 18.04 LTS 64-bit" @ CH-GVA-2 (featured)
	// cs --region cloudstack listTemplates \
	//     templatefilter=featured \
	//     zoneid=1128bd56-b4d9-4ac6-a7b9-c715b187ce11 \
	//     name="Linux Ubuntu 18.04 LTS 64-bit"
	datasourceComputeTemplateID       = "095250e3-7c56-441a-a25b-100a3d3f5a6e"
	datasourceComputeTemplateName     = "Linux Ubuntu 18.04 LTS 64-bit"
	datasourceComputeTemplateUsername = "ubuntu"
)

func TestAccDatasourceComputeTemplate(t *testing.T) {
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
}`, datasourceComputeTemplateName),
				Check: resource.ComposeTestCheckFunc(
					testAccDatasourceComputeTemplateAttributes(testAttrs{
						"id":       ValidateString(datasourceComputeTemplateID),
						"name":     ValidateString(datasourceComputeTemplateName),
						"username": ValidateString(datasourceComputeTemplateUsername),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
data "exoscale_compute_template" "ubuntu_lts" {
  zone   = "ch-gva-2"
  id     = "%s"
  filter = "featured"
}`, datasourceComputeTemplateID),
				Check: resource.ComposeTestCheckFunc(
					testAccDatasourceComputeTemplateAttributes(testAttrs{
						"id":       ValidateString(datasourceComputeTemplateID),
						"name":     ValidateString(datasourceComputeTemplateName),
						"username": ValidateString(datasourceComputeTemplateUsername),
					}),
				),
			},
		},
	})
}

func testAccDatasourceComputeTemplateAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_compute_template" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("compute_template datasource not found in the state")
	}
}
