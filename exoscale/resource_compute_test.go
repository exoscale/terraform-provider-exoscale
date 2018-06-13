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
			resource.TestStep{
				Config: testAccComputeCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeExists("exoscale_compute.vm", vm),
					testAccCheckComputeAttributes(vm),
					testAccCheckComputeCreateAttributes("terraform-test-compute"),
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

		client := GetComputeClient(testAccProvider.Meta())
		vm.ID = rs.Primary.ID
		if err := client.Get(vm); err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckComputeAttributes(vm *egoscale.VirtualMachine) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if vm.ID == "" {
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

			if rs.Primary.Attributes["name"] != name {
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

		key := &egoscale.VirtualMachine{ID: rs.Primary.ID}
		if err := client.Get(key); err != nil {
			if r, ok := err.(*egoscale.ErrorResponse); ok {
				if r.ErrorCode == egoscale.ParamError {
					return nil
				}
			}
			return err
		}
	}
	return fmt.Errorf("Compute: still exists")
}

var testAccComputeCreate = `
resource "exoscale_ssh_keypair" "key" {
  name = "terraform-test-keypair"
}

resource "exoscale_compute" "vm" {
  display_name = "terraform-test-compute"
  template = "Linux Ubuntu 17.10 64-bit"
  zone = "ch-dk-2"
  size = "Micro"
  disk_size = "12"
  key_pair = "${exoscale_ssh_keypair.key.name}"

  tags {
    test = "terraform"
  }

  timeouts {
    create = "10m"
    delete = "30m"
  }
}
`
