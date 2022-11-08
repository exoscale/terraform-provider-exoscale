package exoscale

import (
	"errors"
	"fmt"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	testAccResourceComputeSSHKeyName         = acctest.RandomWithPrefix(testPrefix)
	testAccResourceComputeSecurityGroupName  = acctest.RandomWithPrefix(testPrefix)
	testAccResourceComputeTemplateName       = testInstanceTemplateName
	testAccResourceComputeDisplayName        = acctest.RandomWithPrefix(testPrefix)
	testAccResourceComputeDisplayNameUpdated = testAccResourceComputeDisplayName + "-updated"
	testAccResourceComputeHostname           = acctest.RandomWithPrefix(testPrefix)
	testAccResourceComputeSize               = "Micro"
	testAccResourceComputeSizeUpdated        = "Small"
	testAccResourceComputeDiskSize           = "10"
	testAccResourceComputeDiskSizeUpdated    = "15"
	testAccResourceComputeReverseDNS         = "test.com."
	testAccResourceComputeReverseDNSUpdated  = "test-updated.com."

	testAccResourceComputeConfigCreateTemplateByName = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_ssh_keypair" "key" {
  name = "%s"
}

resource "exoscale_compute" "vm" {
  zone         = local.zone
  template     = "%s"
  display_name = "%s"
  size         = "%s"
  disk_size    = "%s"
  key_pair     = exoscale_ssh_keypair.key.name
  reverse_dns  = "%s"

  tags = {
    test = "terraform"
  }
}
`,
		testZoneName,
		testAccResourceComputeSSHKeyName,
		testAccResourceComputeTemplateName,
		testAccResourceComputeDisplayName,
		testAccResourceComputeSize,
		testAccResourceComputeDiskSize,
		testAccResourceComputeReverseDNS,
	)

	testAccResourceComputeConfigCreateTemplateByID = fmt.Sprintf(`
locals {
  zone = "%s"
}

data "exoscale_compute_template" "template" {
  zone = local.zone
  name = "%s"
}

resource "exoscale_ssh_keypair" "key" {
  name = "%s"
}

resource "exoscale_compute" "vm" {
  template_id  = data.exoscale_compute_template.template.id
  zone         = local.zone
  display_name = "%s"
  size         = "%s"
  disk_size    = "%s"
  key_pair     = exoscale_ssh_keypair.key.name
  reverse_dns  = "%s"

  tags = {
    test = "terraform"
  }
}
`,
		testZoneName,
		testAccResourceComputeTemplateName,
		testAccResourceComputeSSHKeyName,
		testAccResourceComputeDisplayName,
		testAccResourceComputeSize,
		testAccResourceComputeDiskSize,
		testAccResourceComputeReverseDNS,
	)

	testAccResourceComputeConfigUpdate = fmt.Sprintf(`
locals {
  zone = "%s"
}

data "exoscale_compute_template" "template" {
  zone = local.zone
  name = "%s"
}

resource "exoscale_ssh_keypair" "key" {
  name = "%s"
}

resource "exoscale_security_group" "sg" {
  name = "%s"
}

resource "exoscale_compute" "vm" {
  template_id  = data.exoscale_compute_template.template.id
  zone         = local.zone
  display_name = "%s"
  hostname     = "%s"
  size         = "%s"
  disk_size    = "%s"
  key_pair     = exoscale_ssh_keypair.key.name
  reverse_dns  = "%s"

  user_data = <<EOF
#cloud-config
package_upgrade: true
EOF

  security_groups = ["default", "%s"]

  ip6 = true

  timeouts {
    delete = "10m"
  }

  # Ensure SG exists before we reference it
  depends_on = ["exoscale_security_group.sg"]
}
`,
		testZoneName,
		testAccResourceComputeTemplateName,
		testAccResourceComputeSSHKeyName,
		testAccResourceComputeSecurityGroupName,
		testAccResourceComputeDisplayNameUpdated,
		testAccResourceComputeHostname,
		testAccResourceComputeSizeUpdated,
		testAccResourceComputeDiskSizeUpdated,
		testAccResourceComputeReverseDNSUpdated,
		testAccResourceComputeSecurityGroupName,
	)
)

func TestAccResourceCompute(t *testing.T) {
	vm := new(egoscale.VirtualMachine)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceComputeDestroy,
		Steps: []resource.TestStep{
			{
				// This should go away once `template` attribute is phased out
				Config: testAccResourceComputeConfigCreateTemplateByName,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceComputeExists("exoscale_compute.vm", vm),
					testAccCheckResourceCompute(vm),
					testAccCheckResourceComputeAttributes(testAttrs{
						"template":     validateString(testAccResourceComputeTemplateName),
						"template_id":  validation.ToDiagFunc(validation.IsUUID),
						"display_name": validateString(testAccResourceComputeDisplayName),
						"hostname":     validateString(testAccResourceComputeDisplayName),
						"name":         validateString(testAccResourceComputeDisplayName),
						"size":         validateString(testAccResourceComputeSize),
						"disk_size":    validateString(testAccResourceComputeDiskSize),
						"key_pair":     validateString(testAccResourceComputeSSHKeyName),
						"tags.test":    validateString("terraform"),
						"reverse_dns":  validateString(testAccResourceComputeReverseDNS),
					}),
				),
			},
			{
				Config: testAccResourceComputeConfigCreateTemplateByID,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceComputeExists("exoscale_compute.vm", vm),
					testAccCheckResourceCompute(vm),
					testAccCheckResourceComputeAttributes(testAttrs{
						"template_id":  validation.ToDiagFunc(validation.IsUUID),
						"display_name": validateString(testAccResourceComputeDisplayName),
						"hostname":     validateString(testAccResourceComputeDisplayName),
						"name":         validateString(testAccResourceComputeDisplayName),
						"size":         validateString(testAccResourceComputeSize),
						"disk_size":    validateString(testAccResourceComputeDiskSize),
						"key_pair":     validateString(testAccResourceComputeSSHKeyName),
						"tags.test":    validateString("terraform"),
						"reverse_dns":  validateString(testAccResourceComputeReverseDNS),
					}),
				),
			},
			{
				Config: testAccResourceComputeConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceComputeExists("exoscale_compute.vm", vm),
					testAccCheckResourceCompute(vm),
					testAccCheckResourceComputeAttributes(testAttrs{
						"template_id":       validation.ToDiagFunc(validation.IsUUID),
						"display_name":      validateString(testAccResourceComputeDisplayNameUpdated),
						"hostname":          validateString(testAccResourceComputeHostname),
						"name":              validateString(testAccResourceComputeHostname),
						"size":              validateString(testAccResourceComputeSizeUpdated),
						"disk_size":         validateString(testAccResourceComputeDiskSizeUpdated),
						"key_pair":          validateString(testAccResourceComputeSSHKeyName),
						"security_groups.#": validateString("2"),
						"ip6":               validateString("true"),
						"user_data":         validateString("#cloud-config\npackage_upgrade: true\n"),
						"reverse_dns":       validateString(testAccResourceComputeReverseDNSUpdated),
					}),
				),
			},
			{
				ResourceName:            "exoscale_compute.vm",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"username", "password", "user_data_base64"},
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							"template_id":       validation.ToDiagFunc(validation.IsUUID),
							"display_name":      validateString(testAccResourceComputeDisplayNameUpdated),
							"hostname":          validateString(testAccResourceComputeHostname),
							"name":              validateString(testAccResourceComputeHostname),
							"size":              validateString(testAccResourceComputeSizeUpdated),
							"disk_size":         validateString(testAccResourceComputeDiskSizeUpdated),
							"key_pair":          validateString(testAccResourceComputeSSHKeyName),
							"security_groups.#": validateString("2"),
							"ip6":               validateString("true"),
							"user_data":         validateString("#cloud-config\npackage_upgrade: true\n"),
							"reverse_dns":       validateString(testAccResourceComputeReverseDNSUpdated),
						},
						func(s []*terraform.InstanceState) map[string]string {
							for _, state := range s {
								if state.ID == vm.ID.String() {
									return state.Attributes
								}
							}
							return nil
						}(s),
					)
				},
			},
		},
	})
}

func testAccCheckResourceComputeExists(n string, vm *egoscale.VirtualMachine) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		id, err := egoscale.ParseUUID(rs.Primary.ID)
		if err != nil {
			return err
		}

		client := GetComputeClient(testAccProvider.Meta())

		resp, err := client.Get(&egoscale.VirtualMachine{
			ID: id,
		})
		if err != nil {
			return err
		}

		return Copy(vm, resp.(*egoscale.VirtualMachine))
	}
}

func testAccCheckResourceCompute(vm *egoscale.VirtualMachine) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if vm.ID == nil {
			return errors.New("Compute instance ID is nil")
		}

		return nil
	}
}

func testAccCheckResourceComputeAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_compute" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("resource not found in the state")
	}
}

func testAccCheckResourceComputeDestroy(s *terraform.State) error {
	client := GetComputeClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_compute" {
			continue
		}

		id, err := egoscale.ParseUUID(rs.Primary.ID)
		if err != nil {
			return err
		}

		vm := &egoscale.VirtualMachine{ID: id}
		_, err = client.Get(vm)
		if err != nil {
			if errors.Is(err, egoscale.ErrNotFound) {
				return nil
			}
			return err
		}
	}
	return errors.New("Compute instance still exists")
}
