package instance_test

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

var (
	dsListSecurityGroupName       = acctest.RandomWithPrefix(testutils.Prefix)
	dsListDiskSize          int64 = 10
	dsListName                    = acctest.RandomWithPrefix(testutils.Prefix)
	dsListSSHKeyName              = acctest.RandomWithPrefix(testutils.Prefix)
	dsListReverseDNS              = "tf-provider-rdns-test.exoscale.com"
	dsListType                    = "standard.tiny"
	dsListZone                    = "at-vie-2"

	dsListConfig = fmt.Sprintf(`
locals {
  zone = "%s"
}
data "exoscale_template" "ubuntu" {
  zone = local.zone
  name = "Linux Ubuntu 22.04 LTS 64-bit"
}
resource "exoscale_security_group" "test" {
  name = "%s"
}
resource "exoscale_ssh_key" "test" {
  name       = "%s"
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIGhXL32fOylqUARtE6mPuQPm37B15OlH7GDshQRBPhpx test"
}
resource "exoscale_compute_instance" "test" {
  zone                    = local.zone
  name                    = "%s"
  type                    = "%s"
  disk_size               = %d
  template_id             = data.exoscale_template.ubuntu.id
  ipv6                    = true
  ssh_key                 = exoscale_ssh_key.test.name
	reverse_dns             = "%s"
  timeouts {
    delete = "10m"
  }
}
`,
		dsListZone,
		dsListSecurityGroupName,
		dsListSSHKeyName,
		dsListName,
		dsListType,
		dsListDiskSize,
		dsListReverseDNS,
	)
)

func testListDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             dsListConfig,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_compute_instance_list" "test" {
  # we omit the zone to trigger an error as the zone attribute must be mandatory.
  name = %q
}
`,
					dsListConfig,
					dsListName,
				),
				ExpectError: regexp.MustCompile("Missing required argument"),
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
						"instances.#":             testutils.ValidateString("1"),
						"instances.0.id":          validation.ToDiagFunc(validation.NoZeroValues),
						"instances.0.name":        testutils.ValidateString(dsListName),
						"instances.0.type":        testutils.ValidateString(dsListType),
						"instances.0.ssh_key":     testutils.ValidateString(dsListSSHKeyName),
						"instances.0.disk_size":   testutils.ValidateString(fmt.Sprint(dsDiskSize)),
						"instances.0.reverse_dns": testutils.ValidateString(dsListReverseDNS + "."),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
			%s

			data "exoscale_compute_instance_list" "test" {
			  zone = local.zone
			  reverse_dns = %q
			}
			`,
					dsListConfig,
					dsListReverseDNS+".",
				),
				Check: resource.ComposeTestCheckFunc(
					dsCheckListAttrs("data.exoscale_compute_instance_list.test", testutils.TestAttrs{
						"instances.#":             testutils.ValidateString("1"),
						"instances.0.id":          validation.ToDiagFunc(validation.NoZeroValues),
						"instances.0.name":        testutils.ValidateString(dsListName),
						"instances.0.type":        testutils.ValidateString(dsListType),
						"instances.0.ssh_key":     testutils.ValidateString(dsListSSHKeyName),
						"instances.0.disk_size":   testutils.ValidateString(fmt.Sprint(dsDiskSize)),
						"instances.0.reverse_dns": testutils.ValidateString(dsListReverseDNS + "."),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
			%s

			data "exoscale_compute_instance_list" "test" {
			  zone = local.zone
			  disk_size = %d
			}
			`,
					dsListConfig,
					dsListDiskSize,
				),
				Check: resource.ComposeTestCheckFunc(
					dsCheckListAttrs("data.exoscale_compute_instance_list.test", testutils.TestAttrs{
						"instances.#":             testutils.ValidateString("1"),
						"instances.0.id":          validation.ToDiagFunc(validation.NoZeroValues),
						"instances.0.name":        testutils.ValidateString(dsListName),
						"instances.0.type":        testutils.ValidateString(dsListType),
						"instances.0.ssh_key":     testutils.ValidateString(dsListSSHKeyName),
						"instances.0.disk_size":   testutils.ValidateString(fmt.Sprint(dsDiskSize)),
						"instances.0.reverse_dns": testutils.ValidateString(dsListReverseDNS + "."),
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
