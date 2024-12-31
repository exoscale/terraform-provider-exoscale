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

	"github.com/exoscale/terraform-provider-exoscale/pkg/resources/instance"
	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

var (
	dsAntiAffinityGroupName       = acctest.RandomWithPrefix(testutils.Prefix)
	dsDiskSize              int64 = 10
	dsLabelValue                  = acctest.RandomWithPrefix(testutils.Prefix)
	dsName                        = acctest.RandomWithPrefix(testutils.Prefix)
	dsPrivateNetworkName          = acctest.RandomWithPrefix(testutils.Prefix)
	dsSSHKeyName                  = acctest.RandomWithPrefix(testutils.Prefix)
	dsReverseDNS                  = "tf-provider-rdns-test.exoscale.com"
	dsSecurityGroupName           = acctest.RandomWithPrefix(testutils.Prefix)
	dsType                        = "standard.tiny"
	dsUserData                    = acctest.RandString(10)

	dsConfig = fmt.Sprintf(`
locals {
  zone = "%s"
}

data "exoscale_template" "ubuntu" {
  zone = local.zone
  name = "Linux Ubuntu 20.04 LTS 64-bit"
}

resource "exoscale_security_group" "test" {
  name = "%s"
}

resource "exoscale_anti_affinity_group" "test" {
  name = "%s"
}

resource "exoscale_private_network" "test" {
  zone = local.zone
  name = "%s"
}

resource "exoscale_elastic_ip" "test" {
  zone = local.zone
}

resource "exoscale_ssh_key" "test" {
  name       = "%s"
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIBbM7A2vC0avqeFBvc0QdZMb6YjP4rTD0VLfV0tnbkGD test"
}

resource "exoscale_compute_instance" "test" {
  zone                    = local.zone
  name                    = "%s"
  type                    = "%s"
  disk_size               = %d
  template_id             = data.exoscale_template.ubuntu.id
  ipv6                    = true
  anti_affinity_group_ids = [exoscale_anti_affinity_group.test.id]
  security_group_ids      = [exoscale_security_group.test.id]
  elastic_ip_ids          = [exoscale_elastic_ip.test.id]
  user_data               = "%s"
  ssh_key                 = exoscale_ssh_key.test.name
	reverse_dns             = "%s"

  network_interface {
    network_id = exoscale_private_network.test.id
  }

  labels = {
    test = "%s"
  }

  timeouts {
    delete = "10m"
  }
}
`,
		testutils.TestZoneName,
		dsSecurityGroupName,
		dsAntiAffinityGroupName,
		dsPrivateNetworkName,
		dsSSHKeyName,
		dsName,
		dsType,
		dsDiskSize,
		dsUserData,
		dsReverseDNS,
		dsLabelValue,
	)
)

func testDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testutils.AccPreCheck(t) },
		ProviderFactories: testutils.Providers(),
		Steps: []resource.TestStep{
			{
				Config:      `data "exoscale_compute_instance" "test" { zone = "ch-gva-2" }`,
				ExpectError: regexp.MustCompile("either name or id must be specified"),
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_compute_instance" "by-id" {
  zone = local.zone
  id   = exoscale_compute_instance.test.id
}
`,
					dsConfig,
				),
				Check: resource.ComposeTestCheckFunc(
					dsCheckAttrs("data.exoscale_compute_instance.by-id", testutils.TestAttrs{
						instance.AttrID:   validation.ToDiagFunc(validation.IsUUID),
						instance.AttrName: testutils.ValidateString(dsName),
					}),
				),
			},
			{
				Config: fmt.Sprintf(`
%s

data "exoscale_compute_instance" "by-name" {
  zone = local.zone
  name = exoscale_compute_instance.test.name
}
`,
					dsConfig,
				),
				Check: resource.ComposeTestCheckFunc(
					dsCheckAttrs("data.exoscale_compute_instance.by-name", testutils.TestAttrs{
						instance.AttrID:         validation.ToDiagFunc(validation.IsUUID),
						instance.AttrName:       testutils.ValidateString(dsName),
						instance.AttrReverseDNS: testutils.ValidateString(dsReverseDNS),
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

		return errors.New("exoscale_compute_instance data source not found in the state")
	}
}
