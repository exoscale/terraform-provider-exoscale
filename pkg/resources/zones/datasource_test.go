package zones_test

import (
	"errors"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func TestGetZones(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "exoscale_zones" "example_zones" {}
`,
				Check: resource.ComposeTestCheckFunc(
					dsCheckAttrs("data.exoscale_zones.example_zones", testutils.TestAttrs{
						"zones.#": testutils.ValidateString("6"),
						"zones.0": testutils.ValidateString("ch-gva-2"),
						"zones.1": testutils.ValidateString("ch-dk-2"),
						"zones.2": testutils.ValidateString("at-vie-1"),
						"zones.3": testutils.ValidateString("de-fra-1"),
						"zones.4": testutils.ValidateString("bg-sof-1"),
						"zones.5": testutils.ValidateString("at-vie-2"),
					}),
				),
			},
		},
	})
}

func dsCheckAttrs(ds string, expected testutils.TestAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return testutils.CheckResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("exoscale_zones data source not found in the state")
	}
}
