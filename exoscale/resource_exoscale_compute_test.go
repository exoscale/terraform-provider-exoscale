package exoscale

import (
	"errors"
	"fmt"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var (
	testAccResourceComputeSSHKeyName        = testPrefix + "-" + testRandomString()
	testAccResourceComputeSecurityGroupName = testPrefix + "-" + testRandomString()
	testAccResourceComputeZoneName          = testZoneName
	testAccResourceComputeTemplateName      = testInstanceTemplateName
	testAccResourceComputeTemplateID        = testInstanceTemplateID
	testAccResourceComputeName              = testPrefix + "-" + testRandomString()
	testAccResourceComputeNameUpdated       = testAccResourceComputeName + "-updated"
	testAccResourceComputeSize              = "Micro"
	testAccResourceComputeSizeUpdated       = "Small"
	testAccResourceComputeDiskSize          = "10"
	testAccResourceComputeDiskSizeUpdated   = "15"

	testAccResourceComputeConfigCreateTemplateByName = fmt.Sprintf(`
resource "exoscale_ssh_keypair" "key" {
  name = "%s"
}

resource "exoscale_compute" "vm" {
  zone = "%s"
  template = "%s"
  display_name = "%s"
  size = "%s"
  disk_size = "%s"
  key_pair = "${exoscale_ssh_keypair.key.name}"

  tags = {
    test = "terraform"
  }
}
`,
		testAccResourceComputeSSHKeyName,
		testAccResourceComputeZoneName,
		testAccResourceComputeTemplateName,
		testAccResourceComputeName,
		testAccResourceComputeSize,
		testAccResourceComputeDiskSize,
	)

	testAccResourceComputeConfigCreateTemplateByID = fmt.Sprintf(`
resource "exoscale_ssh_keypair" "key" {
  name = "%s"
}

resource "exoscale_compute" "vm" {
  template_id = "%s"
  zone = "%s"
  display_name = "%s"
  size = "%s"
  disk_size = "%s"
  key_pair = "${exoscale_ssh_keypair.key.name}"

  tags = {
    test = "terraform"
  }
}
`,
		testAccResourceComputeSSHKeyName,
		testInstanceTemplateID,
		testZoneName,
		testAccResourceComputeName,
		testAccResourceComputeSize,
		testAccResourceComputeDiskSize,
	)

	testAccResourceComputeConfigUpdate = fmt.Sprintf(`
resource "exoscale_ssh_keypair" "key" {
  name = "%s"
}

resource "exoscale_security_group" "sg" {
  name = "%s"
}

resource "exoscale_compute" "vm" {
  template_id = "%s"
  zone = "%s"
  display_name = "%s"
  size = "%s"
  disk_size = "%s"
  key_pair = "${exoscale_ssh_keypair.key.name}"

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
		testAccResourceComputeSSHKeyName,
		testAccResourceComputeSecurityGroupName,
		testInstanceTemplateID,
		testZoneName,
		testAccResourceComputeNameUpdated,
		testAccResourceComputeSizeUpdated,
		testAccResourceComputeDiskSizeUpdated,
		testAccResourceComputeSecurityGroupName,
	)
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
						"template":     ValidateString(testInstanceTemplateName),
						"template_id":  ValidateString(testInstanceTemplateID),
						"display_name": ValidateString(testAccResourceComputeName),
						"size":         ValidateString(testAccResourceComputeSize),
						"disk_size":    ValidateString(testAccResourceComputeDiskSize),
						"key_pair":     ValidateString(testAccResourceComputeSSHKeyName),
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
						"template_id":  ValidateString(testInstanceTemplateID),
						"display_name": ValidateString(testAccResourceComputeName),
						"size":         ValidateString(testAccResourceComputeSize),
						"disk_size":    ValidateString(testAccResourceComputeDiskSize),
						"key_pair":     ValidateString(testAccResourceComputeSSHKeyName),
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
						"template_id":       ValidateString(testInstanceTemplateID),
						"display_name":      ValidateString(testAccResourceComputeNameUpdated),
						"size":              ValidateString(testAccResourceComputeSizeUpdated),
						"disk_size":         ValidateString(testAccResourceComputeDiskSizeUpdated),
						"key_pair":          ValidateString(testAccResourceComputeSSHKeyName),
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
							"template_id":       ValidateString(testInstanceTemplateID),
							"display_name":      ValidateString(testAccResourceComputeNameUpdated),
							"size":              ValidateString(testAccResourceComputeSizeUpdated),
							"disk_size":         ValidateString(testAccResourceComputeDiskSizeUpdated),
							"key_pair":          ValidateString(testAccResourceComputeSSHKeyName),
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
