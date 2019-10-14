package exoscale

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccResourceNIC(t *testing.T) {
	vm := new(egoscale.VirtualMachine)
	network := new(egoscale.Network)
	nic := new(egoscale.Nic)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceNICDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceNICConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceComputeExists("exoscale_compute.vm", vm),
					testAccCheckResourceNetworkExists("exoscale_network.net", network),
					testAccCheckResourceNICExists("exoscale_nic.nic", vm, nic),
					testAccCheckResourceNIC(nic, net.ParseIP("10.0.0.1")),
					testAccCheckResourceNICAttributes(testAttrs{
						"mac_address": ValidateRegexp("^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$"),
						"ip_address":  ValidateString("10.0.0.1"),
					}),
				),
			}, {
				Config: testAccResourceNICConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceComputeExists("exoscale_compute.vm", vm),
					testAccCheckResourceNetworkExists("exoscale_network.net", network),
					testAccCheckResourceNICExists("exoscale_nic.nic", vm, nic),
					testAccCheckResourceNIC(nic, net.ParseIP("10.0.0.3")),
					testAccCheckResourceNICAttributes(testAttrs{
						"mac_address": ValidateRegexp("^([0-9A-Fa-f]{2}[:-]){5}([0-9A-Fa-f]{2})$"),
						"ip_address":  ValidateString("10.0.0.3"),
					}),
				),
			},
		},
	})
}

func testAccCheckResourceNICExists(n string, vm *egoscale.VirtualMachine, nic *egoscale.Nic) resource.TestCheckFunc {
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
		nic.VirtualMachineID = vm.ID
		nic.ID = id
		resp, err := client.Get(nic)
		if err != nil {
			return err
		}

		return Copy(nic, resp.(*egoscale.Nic))
	}
}

func testAccCheckResourceNIC(nic *egoscale.Nic, ipAddress net.IP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if nic.MACAddress == nil {
			return errors.New("NIC is nil")
		}

		if !nic.IPAddress.Equal(ipAddress) {
			return fmt.Errorf("expected NIC IP address %v, got %s", ipAddress, nic.IPAddress)
		}

		return nil
	}
}

func testAccCheckResourceNICAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_nic" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("resource not found in the state")
	}
}

func testAccCheckResourceNICDestroy(s *terraform.State) error {
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
	return errors.New("NIC still exists")
}

var testAccResourceNICConfigCreate = fmt.Sprintf(`
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

var testAccResourceNICConfigUpdate = fmt.Sprintf(`
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