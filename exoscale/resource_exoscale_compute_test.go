package exoscale

import (
	"errors"
	"fmt"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccResourceCompute(t *testing.T) {
	sg := new(egoscale.SecurityGroup)
	vm := new(egoscale.VirtualMachine)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceComputeDestroy,
		Steps: []resource.TestStep{
			{
				// This should go away once `template` attribute is phased out
				Config: testAccResourceComputeConfigCreateTemplateByName,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceComputeExists("exoscale_compute.vm", vm),
					testAccCheckResourceCompute(vm),
					testAccCheckResourceComputeAttributes(testAttrs{
						"template":     ValidateString(defaultExoscaleTemplate),
						"template_id":  ValidateString(defaultExoscaleTemplateID),
						"name":         ValidateString("terraform-test-compute1"),
						"display_name": ValidateString("Terraform test compute 1"),
						"size":         ValidateString("Micro"),
						"disk_size":    ValidateString("12"),
						"key_pair":     ValidateString("terraform-test-keypair"),
						"tags.test":    ValidateString("terraform"),
					}),
				),
			},
			{
				Config: testAccResourceComputeConfigCreateTemplateByID,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceComputeExists("exoscale_compute.vm", vm),
					testAccCheckResourceCompute(vm),
					testAccCheckResourceComputeAttributes(testAttrs{
						"template_id":  ValidateString(defaultExoscaleTemplateID),
						"name":         ValidateString("terraform-test-compute1"),
						"display_name": ValidateString("Terraform test compute 1"),
						"size":         ValidateString("Micro"),
						"disk_size":    ValidateString("12"),
						"key_pair":     ValidateString("terraform-test-keypair"),
						"tags.test":    ValidateString("terraform"),
					}),
				),
			},
			{
				Config: testAccResourceComputeConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSecurityGroupExists("exoscale_security_group.sg", sg),
					testAccCheckResourceComputeExists("exoscale_compute.vm", vm),
					testAccCheckResourceCompute(vm),
					testAccCheckResourceComputeAttributes(testAttrs{
						"template_id":       ValidateString(defaultExoscaleTemplateID),
						"name":              ValidateString("terraform-test-compute2"),
						"display_name":      ValidateString("Terraform test compute 2"),
						"size":              ValidateString("Small"),
						"disk_size":         ValidateString("18"),
						"key_pair":          ValidateString("terraform-test-keypair"),
						"security_groups.#": ValidateString("2"),
						"ip6":               ValidateString("true"),
						"user_data":         ValidateString("#cloud-config\npackage_upgrade: true\n"),
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
							"template_id":       ValidateString(defaultExoscaleTemplateID),
							"name":              ValidateString("terraform-test-compute2"),
							"display_name":      ValidateString("Terraform test compute 2"),
							"size":              ValidateString("Small"),
							"disk_size":         ValidateString("18"),
							"key_pair":          ValidateString("terraform-test-keypair"),
							"security_groups.#": ValidateString("2"),
							"ip6":               ValidateString("true"),
							"user_data":         ValidateString("#cloud-config\npackage_upgrade: true\n"),
						},
						s[0].Attributes)
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
			if r, ok := err.(*egoscale.ErrorResponse); ok {
				if r.ErrorCode == egoscale.ParamError {
					return nil
				}
			}
			return err
		}
	}
	return errors.New("Compute instance still exists")
}

var testAccResourceComputeConfigCreateTemplateByName = fmt.Sprintf(`
resource "exoscale_ssh_keypair" "key" {
  name = "terraform-test-keypair"
}

resource "exoscale_compute" "vm" {
  template = %q
  zone = %q
  name = "terraform-test-compute1"
  display_name = "Terraform test compute 1"
  size = "Micro"
  disk_size = "12"
  key_pair = "${exoscale_ssh_keypair.key.name}"

  tags = {
    test = "terraform"
  }

  timeouts {
    create = "10m"
  }
}
`,
	defaultExoscaleTemplate,
	defaultExoscaleZone,
)

var testAccResourceComputeConfigCreateTemplateByID = fmt.Sprintf(`
resource "exoscale_ssh_keypair" "key" {
  name = "terraform-test-keypair"
}

resource "exoscale_compute" "vm" {
  template_id = %q
  zone = %q
  name = "terraform-test-compute1"
  display_name = "Terraform test compute 1"
  size = "Micro"
  disk_size = "12"
  key_pair = "${exoscale_ssh_keypair.key.name}"

  tags = {
    test = "terraform"
  }

  timeouts {
    create = "10m"
  }
}
`,
	defaultExoscaleTemplateID,
	defaultExoscaleZone,
)

var testAccResourceComputeConfigUpdate = fmt.Sprintf(`
resource "exoscale_ssh_keypair" "key" {
  name = "terraform-test-keypair"
}

resource "exoscale_security_group" "sg" {
  name = "terraform-test-security-group"
}

resource "exoscale_compute" "vm" {
  template_id = %q
  zone = %q
  name = "terraform-test-compute2"
  display_name = "Terraform test compute 2"
  size = "Small"
  disk_size = "18"
  key_pair = "${exoscale_ssh_keypair.key.name}"

  user_data = <<EOF
#cloud-config
package_upgrade: true
EOF

  security_groups = ["default", "terraform-test-security-group"]

  ip6 = true

  timeouts {
    delete = "30m"
  }

  # Ensure SG exists before we reference it
  depends_on = ["exoscale_security_group.sg"]
}
`,
	defaultExoscaleTemplateID,
	defaultExoscaleZone,
)
