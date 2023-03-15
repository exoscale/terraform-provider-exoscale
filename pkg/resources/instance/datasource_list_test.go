package instance_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

var (
	dsListSecurityGroupName       = acctest.RandomWithPrefix(testutils.Prefix)
	dsListDiskSize          int64 = 10
	dsListName                    = acctest.RandomWithPrefix(testutils.Prefix)
	dsListSSHKeyName              = acctest.RandomWithPrefix(testutils.Prefix)
	dsListType                    = "standard.tiny"

	dsListConfig = fmt.Sprintf(`
locals {
  zone = "%s"
}
data "exoscale_compute_template" "ubuntu" {
  zone = local.zone
  name = "Linux Ubuntu 20.04 LTS 64-bit"
}
resource "exoscale_security_group" "test" {
  name = "%s"
}
resource "exoscale_ssh_key" "test" {
  name       = "%s"
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIB8bfA67mQWv4eGND/XVtPx1JW6RAqafub1lV1EcpB+b test"
}
resource "exoscale_compute_instance" "test" {
  zone                    = local.zone
  name                    = "%s"
  type                    = "%s"
  disk_size               = %d
  template_id             = data.exoscale_compute_template.ubuntu.id
  ipv6                    = true
  ssh_key                 = exoscale_ssh_key.test.name
  timeouts {
    delete = "10m"
  }
}
`,
		testutils.TestZoneName,
		dsListSecurityGroupName,
		dsListSSHKeyName,
		dsListName,
		dsListType,
		dsListDiskSize,
	)
)

func testListDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testutils.AccPreCheck(t) },
		ProviderFactories: testutils.Providers(),
		Steps: []resource.TestStep{
			{
				Config:             dsListConfig,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_compute_instance_list" "test" {
  zone = local.zone
  name = %q
}
`,
					dsListConfig,
					dsListName,
				),
				Check: resource.ComposeTestCheckFunc(
					dsCheckListAttrs("data.exoscale_compute_instance_list.test", testutils.TestAttrs{
						"instances.#":         testutils.ValidateString("1"),
						"instances.0.id":      validation.ToDiagFunc(validation.NoZeroValues),
						"instances.0.name":    testutils.ValidateString(dsListName),
						"instances.0.type":    testutils.ValidateString(dsListType),
						"instances.0.ssh_key": testutils.ValidateString(dsListSSHKeyName),
					}),
				),
			},
		},
	})
}

func dsCheckListAttrs(ds string, expected testutils.TestAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for name, res := range s.RootModule().Resources {
			if name == ds {
				return testutils.CheckResourceAttributes(expected, res.Primary.Attributes)
			}
		}

		return errors.New("exoscale_compute_instance data source not found in the state")
	}
}
