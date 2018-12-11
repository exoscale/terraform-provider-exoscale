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
	nw := new(egoscale.Network)
	nic := new(egoscale.Nic)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNicDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNicCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeExists("exoscale_compute.vm", vm),
					testAccCheckNetworkExists("exoscale_network.net", nw),
					testAccCheckNicExists("exoscale_nic.nic", vm, nic),
					testAccCheckNicAttributes(nic, net.ParseIP("10.0.0.1")),
					testAccCheckNicCreateAttributes(),
				),
			}, {
				Config: testAccNicUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeExists("exoscale_compute.vm", vm),
					testAccCheckNetworkExists("exoscale_network.net", nw),
					testAccCheckNicExists("exoscale_nic.nic", vm, nic),
					testAccCheckNicAttributes(nic, net.ParseIP("10.0.0.3")),
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

		id, err := egoscale.ParseUUID(rs.Primary.ID)
		if err != nil {
			return err
		}

		client := GetComputeClient(testAccProvider.Meta())
		nic.VirtualMachineID = vm.ID
		nic.ID = id
		resp, err := client.Get(nic)
		if err != nil {
			return err
		}

		return Copy(nic, resp.(*egoscale.Nic))
	}
}

func testAccCheckNicAttributes(nic *egoscale.Nic, ipAddress net.IP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if nic.MACAddress == nil {
			return fmt.Errorf("nic is nil")
		}

		if !nic.IPAddress.Equal(ipAddress) {
			return fmt.Errorf("nic has bad IP address, got %s, want %s", nic.IPAddress, ipAddress)
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

		vmID, err := egoscale.ParseUUID(rs.Primary.Attributes["compute_id"])
		if err != nil {
			return err
		}

		nic := &egoscale.Nic{VirtualMachineID: vmID}
		_, err = client.Get(nic)
		if err != nil {
			if r, ok := err.(*egoscale.ErrorResponse); ok {
				if r.ErrorCode == egoscale.ParamError {
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

  start_ip = "10.0.0.1"
  end_ip = "10.0.0.1"
  netmask = "255.255.255.252"
}

resource "exoscale_nic" "nic" {
  compute_id = "${exoscale_compute.vm.id}"
  network_id = "${exoscale_network.net.id}"

  ip_address = "10.0.0.1"
}
`,
	defaultExoscaleTemplate,
	defaultExoscaleZone,
	defaultExoscaleZone,
	defaultExoscaleNetworkOffering,
)

var testAccNicUpdate = fmt.Sprintf(`
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

  start_ip = "10.0.0.1"
  end_ip = "10.0.0.1"
  netmask = "255.255.255.248"
}

resource "exoscale_nic" "nic" {
  compute_id = "${exoscale_compute.vm.id}"
  network_id = "${exoscale_network.net.id}"

  ip_address = "10.0.0.3"
}
`,
	defaultExoscaleTemplate,
	defaultExoscaleZone,
	defaultExoscaleZone,
	defaultExoscaleNetworkOffering,
)
