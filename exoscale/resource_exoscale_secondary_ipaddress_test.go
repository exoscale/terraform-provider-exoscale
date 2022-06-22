package exoscale

import (
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	testAccResourceSecondaryIPAddressComputeName  = acctest.RandomWithPrefix(testPrefix)
	testAccResourceSecondaryIPAddressSSHKeyName   = acctest.RandomWithPrefix(testPrefix)
	testAccResourceSecondaryIPAddressTemplateName = testInstanceTemplateName

	testAccResourceSecondaryIPAddressConfig = fmt.Sprintf(`
locals {
  zone = "%s"
}

resource "exoscale_ssh_keypair" "key" {
  name = "%s"
}

resource "exoscale_ipaddress" "eip" {
  zone = local.zone

  tags = {
    terraform = "acceptance"
  }
}

data "exoscale_compute_template" "template" {
  zone = local.zone
  name = "%s"
}

resource "exoscale_compute" "vm" {
  zone         = local.zone
  display_name = "%s"
  size         = "Micro"
  disk_size    = "10"
  key_pair     = exoscale_ssh_keypair.key.name
  template_id  = data.exoscale_compute_template.template.id

  # prevents bad ordering during the deletion
  depends_on = ["exoscale_ipaddress.eip"]

  tags = {
    terraform = "acceptance"
  }
}

resource "exoscale_secondary_ipaddress" "ip" {
  compute_id = exoscale_compute.vm.id
  ip_address = exoscale_ipaddress.eip.ip_address
}
`,
		testZoneName,
		testAccResourceSecondaryIPAddressSSHKeyName,
		testAccResourceSecondaryIPAddressTemplateName,
		testAccResourceSecondaryIPAddressComputeName,
	)
)

func TestAccResourceSecondaryIPAddress(t *testing.T) {
	vm := new(egoscale.VirtualMachine)
	eip := new(egoscale.IPAddress)
	secondaryip := new(egoscale.NicSecondaryIP)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceSecondaryIPAddressDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceSecondaryIPAddressConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceComputeExists("exoscale_compute.vm", vm),
					testAccCheckIPAddressExists("exoscale_ipaddress.eip", eip),
					testAccCheckResourceSecondaryIPAddressExists("exoscale_secondary_ipaddress.ip", vm, secondaryip),
					testAccCheckResourceSecondaryIPAddress(secondaryip),
					testAccCheckResourceSecondaryIPAddressAttributes(testAttrs{
						"nic_id":     validation.ToDiagFunc(validation.IsUUID),
						"network_id": validation.ToDiagFunc(validation.IsUUID),
					}),
				),
			},
			{
				ResourceName:      "exoscale_secondary_ipaddress.ip",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							"nic_id":     validation.ToDiagFunc(validation.IsUUID),
							"network_id": validation.ToDiagFunc(validation.IsUUID),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceSecondaryIPAddressExists(n string, vm *egoscale.VirtualMachine, secondaryip *egoscale.NicSecondaryIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		nic := vm.DefaultNic()
		if nic == nil || len(nic.SecondaryIP) != 1 {
			return errors.New("no secondaryip field in VM")
		}

		return Copy(secondaryip, nic.SecondaryIP[0])
	}
}

func testAccCheckResourceSecondaryIPAddress(nic *egoscale.NicSecondaryIP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if nic.IPAddress == nil {
			return errors.New("ip address is nil")
		}

		return nil
	}
}

func testAccCheckResourceSecondaryIPAddressAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_secondary_ipaddress" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("resource not found in the state")
	}
}

func testAccCheckResourceSecondaryIPAddressDestroy(s *terraform.State) error {
	client := GetComputeClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_secondary_ipaddress" {
			continue
		}

		vmID, err := egoscale.ParseUUID(rs.Primary.Attributes["compute_id"])
		if err != nil {
			return err
		}

		vm := &egoscale.VirtualMachine{ID: vmID}
		_, err = client.Get(vm)
		if err != nil {
			if errors.Is(err, egoscale.ErrNotFound) {
				return nil
			}
			return err
		}

		nic := vm.DefaultNic()
		if nic == nil {
			return nil
		}

		ipAddress := net.ParseIP(rs.Primary.Attributes["ip_address"])
		if ipAddress == nil {
			return errors.New("not a valid IP address")
		}

		for _, ip := range nic.SecondaryIP {
			if ip.IPAddress.Equal(ipAddress) {
				return errors.New("Secondary IP address still exists")
			}
		}
	}

	return nil
}
