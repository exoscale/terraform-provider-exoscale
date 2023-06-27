package exoscale

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

var (
	testAccDataSourceTemplateName        = testInstanceTemplateName
	testAccDataSourceTemplateDefaultUser = testInstanceTemplateUsername
	testAccDataSourceTemplateVisibility  = testInstanceTemplateVisibility
	testAccDataSourceTemplateZone        = testZoneName
	testAccDataSourceTemplateConfig      = fmt.Sprintf(`
locals {
  zone = "%s"
}
data "exoscale_template" "test" {
	zone = local.zone
	name = "%s"
}
`,
		testAccDataSourceTemplateZone,
		testAccDataSourceTemplateName,
	)
)

func TestAccDataSourceTemplate(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		Steps: []resource.TestStep{
			{
				Config:      ` data "exoscale_template" "test" { zone = "lolnope" }`,
				ExpectError: regexp.MustCompile("either id or name must be specified"),
			},
			{
				Config: testAccDataSourceTemplateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccDataSourceTemplateAttributes("data.exoscale_template.test", testAttrs{
						dsTemplateAttrDefaultUser: validateString(testAccDataSourceTemplateDefaultUser),
						dsTemplateAttrID:          validation.ToDiagFunc(validation.IsUUID),
						dsTemplateAttrName:        validateString(testAccDataSourceTemplateName),
						dsTemplateAttrVisibility:  validateString(testAccDataSourceTemplateVisibility),
						dsTemplateAttrZone:        validateString(testAccDataSourceTemplateZone),
					}),
				),
			},
		},
	})
}

func testAccDataSourceTemplateAttributes(ds string, expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return checkResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("exoscale_template data source not found in the state")
	}
}
