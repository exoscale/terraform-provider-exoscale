package exoscale

import (
	"fmt"
	"net"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccNic(t *testing.T) {
	vm := new(egoscale.VirtualMachine)
	net := new(egoscale.Network)
	nic := new(egoscale.Nic)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNicDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccNicCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeExists("exoscale_compute.vm", vm),
					testAccCheckNetworkExists("exoscale_network.net", net),
					testAccCheckNicExists("exoscale_nic.nic", vm, nic),
					testAccCheckNicAttributes(nic),
					testAccCheckNicCreateAttributes(),
				),
			},
		},
	})
}

func testAccCheckNicExists(n string, vm *egoscale.VirtualMachine, nic *egoscale.Nic) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no nic ID is set")
		}

		client := GetComputeClient(testAccProvider.Meta())
		nic.VirtualMachineID = vm.ID
		nic.ID = rs.Primary.ID
		if err := client.Get(nic); err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckNicAttributes(nic *egoscale.Nic) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if nic.MACAddress == nil {
			return fmt.Errorf("nic is nil")
		}

		return nil
	}
}

func testAccCheckNicCreateAttributes() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_nic" {
				continue
			}
			_, err := net.ParseMAC(rs.Primary.Attributes["mac_address"])
			if err != nil {
				return fmt.Errorf("Bad MAC %s", err)
			}

			return nil
		}

		return fmt.Errorf("could not find nic mac address")
	}
}

func testAccCheckNicDestroy(s *terraform.State) error {
	client := GetComputeClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_nic" {
			continue
		}

		nic := &egoscale.Nic{VirtualMachineID: rs.Primary.Attributes["compute_id"]}
		if err := client.Get(nic); err != nil {
			if r, ok := err.(*egoscale.ErrorResponse); ok {
				if r.ErrorText == "Virtual machine id does not exist" {
					return nil
				}
			}
			return err
		}
	}
	return fmt.Errorf("nic still exists")
}

var testAccNicCreate = fmt.Sprintf(`
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

  timeouts {
    create = "10m"
    delete = "30m"
  }
}

resource "exoscale_network" "net" {
  name = "terraform-test-network"
  display_text = "Terraform Acceptance Test"
  zone = %q
  network_offering = %q
}

resource "exoscale_nic" "nic" {
  compute_id = "${exoscale_compute.vm.id}"
  network_id = "${exoscale_network.net.id}"
}
`,
	EXOSCALE_TEMPLATE,
	EXOSCALE_ZONE,
	EXOSCALE_ZONE,
	EXOSCALE_NETWORK_OFFERING,
)
