package exoscale

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/egoscale"
)

var (
	testAccResourceNICSSHKeyName        = acctest.RandomWithPrefix(testPrefix)
	testAccResourceNICSNetworkName      = acctest.RandomWithPrefix(testPrefix)
	testAccResourceNICComputeName       = acctest.RandomWithPrefix(testPrefix)
	testAccResourceNICTemplateName      = testInstanceTemplateName
	testAccResourceNICCIPAddress        = "10.0.0.1"
	testAccResourceNICCIPAddressUpdated = "10.0.0.3"

	testAccResourceNICConfigCreate = fmt.Sprintf(`
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
  zone         = local.zone
  display_name = "%s"
  template_id  = data.exoscale_compute_template.template.id
  size         = "Micro"
  disk_size    = "10"
  key_pair     = exoscale_ssh_keypair.key.name
}

resource "exoscale_network" "net" {
  zone     = local.zone
  name     = "%s"
  start_ip = "10.0.0.1"
  end_ip   = "10.0.0.1"
  netmask  = "255.255.255.252"
}

resource "exoscale_nic" "nic" {
  compute_id = exoscale_compute.vm.id
  network_id = exoscale_network.net.id
  ip_address = "%s"
}
`,
		testZoneName,
		testAccResourceNICTemplateName,
		testAccResourceNICSSHKeyName,
		testAccResourceNICComputeName,
		testAccResourceNICSNetworkName,
		testAccResourceNICCIPAddress,
	)

	testAccResourceNICConfigUpdate = fmt.Sprintf(`
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
  zone         = local.zone
  display_name = "%s"
  template_id  = data.exoscale_compute_template.template.id
  size         = "Micro"
  disk_size    = "10"
  key_pair     = exoscale_ssh_keypair.key.name
}

resource "exoscale_network" "net" {
  zone     = local.zone
  name     = "%s"
  start_ip = "10.0.0.1"
  end_ip   = "10.0.0.1"
  netmask  = "255.255.255.248"
}

resource "exoscale_nic" "nic" {
  compute_id = exoscale_compute.vm.id
  network_id = exoscale_network.net.id
  ip_address = "%s"
}
`,
		testZoneName,
		testAccResourceNICTemplateName,
		testAccResourceNICSSHKeyName,
		testAccResourceNICComputeName,
		testAccResourceNICSNetworkName,
		testAccResourceNICCIPAddressUpdated,
	)
)

func TestAccResourceNIC(t *testing.T) {
	vm := new(egoscale.VirtualMachine)
	network := new(egoscale.Network)
	nic := new(egoscale.Nic)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceNICDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceNICConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceComputeExists("exoscale_compute.vm", vm),
					testAccCheckResourceNetworkExists("exoscale_network.net", network),
					testAccCheckResourceNICExists("exoscale_nic.nic", vm, nic),
					testAccCheckResourceNIC(nic, net.ParseIP("10.0.0.1")),
					testAccCheckResourceNICAttributes(testAttrs{
						"mac_address": validation.ToDiagFunc(validation.IsMACAddress),
						"ip_address":  validateString(testAccResourceNICCIPAddress),
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
						"mac_address": validation.ToDiagFunc(validation.IsMACAddress),
						"ip_address":  validateString(testAccResourceNICCIPAddressUpdated),
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
			if errors.Is(err, egoscale.ErrNotFound) {
				return nil
			}
			return err
		}
	}
	return errors.New("NIC still exists")
}
