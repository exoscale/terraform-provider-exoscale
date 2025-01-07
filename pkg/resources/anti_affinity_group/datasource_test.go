package anti_affinity_group_test

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/exoscale/testutils"
	aagroup "github.com/exoscale/terraform-provider-exoscale/pkg/resources/anti_affinity_group"
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
					dsTestAttributes("data.exoscale_anti_affinity_group.by-id", testutils.TestAttrs{
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
					dsTestAttributes("data.exoscale_anti_affinity_group.by-name", testutils.TestAttrs{
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

data "exoscale_template" "ubuntu" {
  zone = "%s"
  name = "%s"
}

data "exoscale_security_group" "default" {
  name = "default"
}

resource "exoscale_compute_instance" "test" {
  zone               = "%s"
  name               = "test"
  template_id        = data.exoscale_template.ubuntu.id
  security_group_ids = [data.exoscale_security_group.default.id]
  type               = "standard.micro"
  disk_size          = "10"
  anti_affinity_group_ids = [exoscale_anti_affinity_group.test.id]
}

data "exoscale_anti_affinity_group" "test" {
  id         = exoscale_anti_affinity_group.test.id
  depends_on = [exoscale_compute_instance.test]
}`,
					dsGroupName,
					testutils.TestZoneName,
					testutils.TestInstanceTemplateName,
					testutils.TestZoneName,
				),
				Check: resource.ComposeTestCheckFunc(
					dsTestAttributes("data."+aagroup.Name+".test", testutils.TestAttrs{
						aagroup.AttrID:               validation.ToDiagFunc(validation.IsUUID),
						aagroup.AttrInstances + ".#": testutils.ValidateString("1"),
						aagroup.AttrName:             testutils.ValidateString(dsGroupName),
					}),
				),
			},
		},
	})
}

func dsTestAttributes(ds string, expected testutils.TestAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return testutils.CheckResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("exoscale_anti_affinity_group data source not found in the state")
	}
}
