package exoscale

import (
	"fmt"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccCompute(t *testing.T) {
	vm := new(egoscale.VirtualMachine)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckComputeDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccComputeCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeExists("exoscale_compute.vm", vm),
					testAccCheckComputeAttributes(vm),
					testAccCheckComputeCreateAttributes("terraform-test-compute"),
				),
			},
			{
				Config: testAccComputeUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeExists("exoscale_compute.vm", vm),
					testAccCheckComputeAttributes(vm),
					testAccCheckComputeCreateAttributes("acceptance-hello"),
				),
			},
		},
	})
}

func testAccCheckComputeExists(n string, vm *egoscale.VirtualMachine) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Compute ID is set")
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

func testAccCheckComputeAttributes(vm *egoscale.VirtualMachine) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if vm.ID == nil {
			return fmt.Errorf("compute is nil")
		}

		return nil
	}
}

func testAccCheckComputeCreateAttributes(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_compute" {
				continue
			}

			if rs.Primary.Attributes["display_name"] != name {
				continue
			}

			if rs.Primary.Attributes["key_pair"] != "terraform-test-keypair" {
				return fmt.Errorf("Bad key name")
			}

			return nil
		}

		return fmt.Errorf("Could not find compute name: %s", name)
	}
}

func testAccCheckComputeDestroy(s *terraform.State) error {
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
	return fmt.Errorf("compute still exists")
}

var testAccComputeCreate = fmt.Sprintf(`
resource "exoscale_ssh_keypair" "key" {
  name = "terraform-test-keypair"
}

resource "exoscale_compute" "vm" {
  display_name = "terraform-test-compute"
  template = %q
  zone = %q
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

var testAccComputeUpdate = fmt.Sprintf(`
resource "exoscale_ssh_keypair" "key" {
  name = "terraform-test-keypair"
}

resource "exoscale_compute" "vm" {
  display_name = "acceptance-hello"
  template = %q
  zone = %q
  size = "Small"
  disk_size = "18"
  key_pair = "${exoscale_ssh_keypair.key.name}"

  ip6 = true

  timeouts {
    delete = "30m"
  }
}
`,
	defaultExoscaleTemplate,
	defaultExoscaleZone,
)
