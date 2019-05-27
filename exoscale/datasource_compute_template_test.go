package exoscale

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
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
			resource.TestStep{
				Config: fmt.Sprintf(`
data "exoscale_compute_template" "ubuntu_lts" {
  zone = "ch-gva-2"
  name = "%s"
  id   = "%s"
}`,
					datasourceComputeTemplateName,
					datasourceComputeTemplateID),
				ExpectError: regexp.MustCompile("either name or id must be specified"),
			},
			resource.TestStep{
				Config: fmt.Sprintf(`
data "exoscale_compute_template" "ubuntu_lts" {
  zone   = "ch-gva-2"
  name   = "%s"
  filter = "featured"
}`, datasourceComputeTemplateName),
				Check: resource.ComposeTestCheckFunc(
					testAccDatasourceComputeTemplateAttributes(t, "name"),
				),
			},
			resource.TestStep{
				Config: fmt.Sprintf(`
data "exoscale_compute_template" "ubuntu_lts" {
  zone   = "ch-gva-2"
  id     = "%s"
  filter = "featured"
}`, datasourceComputeTemplateID),
				Check: resource.ComposeTestCheckFunc(
					testAccDatasourceComputeTemplateAttributes(t, "id"),
				),
			},
		},
	})
}

func testAccDatasourceComputeTemplateAttributes(t *testing.T, attr string) resource.TestCheckFunc {
	t.Logf("Testing compute_template data source by %s", attr)
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_compute_template" {
				continue
			}

			if rs.Primary.ID != datasourceComputeTemplateID {
				return fmt.Errorf("expected template ID %q, got %q",
					datasourceComputeTemplateID,
					rs.Primary.ID)
			}

			if name, ok := rs.Primary.Attributes["name"]; !ok {
				return fmt.Errorf("template name missing")
			} else if name != datasourceComputeTemplateName {
				return fmt.Errorf("expected name ID %q, got %q",
					datasourceComputeTemplateName,
					rs.Primary.Attributes["name"])
			}

			if username, ok := rs.Primary.Attributes["username"]; !ok {
				return fmt.Errorf("template username missing")
			} else if username != datasourceComputeTemplateUsername {
				return fmt.Errorf("expected username ID %q, got %q",
					datasourceComputeTemplateUsername,
					rs.Primary.Attributes["username"])
			}
		}

		return nil
	}
}
