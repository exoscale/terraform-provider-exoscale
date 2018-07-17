package exoscale

import (
	"fmt"
	"net"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccSecondaryIP(t *testing.T) {
	vm := new(egoscale.VirtualMachine)
	eip := new(egoscale.IPAddress)
	secondaryip := new(egoscale.NicSecondaryIP)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSecondaryIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccSecondaryIPCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckComputeExists("exoscale_compute.vm", vm),
					testAccCheckElasticIPExists("exoscale_ipaddress.eip", eip),
					testAccCheckSecondaryIPExists("exoscale_secondary_ipaddress.ip", vm, secondaryip),
					testAccCheckSecondaryIPAttributes(secondaryip),
					testAccCheckSecondaryIPCreateAttributes(),
				),
			},
		},
	})
}

func testAccCheckSecondaryIPExists(n string, vm *egoscale.VirtualMachine, secondaryip *egoscale.NicSecondaryIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no secondaryip IP ID is set")
		}

		nic := vm.DefaultNic()
		if len(nic.SecondaryIP) != 1 {
			return fmt.Errorf("no secondaryip field in VM")
		}

		*secondaryip = nic.SecondaryIP[0]

		return nil
	}
}

func testAccCheckSecondaryIPAttributes(nic *egoscale.NicSecondaryIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if nic.IPAddress == nil {
			return fmt.Errorf("ip address is nil")
		}

		return nil
	}
}

func testAccCheckSecondaryIPCreateAttributes() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_secondary_ipaddress" {
				continue
			}
			ip := net.ParseIP(rs.Primary.Attributes["ip_address"])
			if ip == nil {
				return fmt.Errorf("Bad IP %s", rs.Primary.Attributes["ip_address"])
			}

			return nil
		}

		return fmt.Errorf("could not find secondary IP address")
	}
}

func testAccCheckSecondaryIPDestroy(s *terraform.State) error {
	client := GetComputeClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_secondary_ipaddress" {
			continue
		}

		vm := &egoscale.VirtualMachine{ID: rs.Primary.Attributes["compute_id"]}
		if err := client.Get(vm); err != nil {
			if r, ok := err.(*egoscale.ErrorResponse); ok {
				if r.ErrorCode == egoscale.ParamError {
					return nil
				}
			}
			return err
		}
	}

	return fmt.Errorf("secondary ip still exists")
}

var testAccSecondaryIPCreate = fmt.Sprintf(`
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

resource "exoscale_ipaddress" "eip" {
  zone = %q
}

resource "exoscale_secondary_ipaddress" "ip" {
  compute_id = "${exoscale_compute.vm.id}"
  ip_address = "${exoscale_ipaddress.eip.ip_address}"
}
`,
	EXOSCALE_TEMPLATE,
	EXOSCALE_ZONE,
	EXOSCALE_ZONE,
)
