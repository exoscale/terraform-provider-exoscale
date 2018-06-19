package exoscale

import (
	"fmt"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccElasticIP(t *testing.T) {
	eip := new(egoscale.IPAddress)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckElasticIPDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccElasticIPCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckElasticIPExists("exoscale_ipaddress.eip", eip),
					testAccCheckElasticIPAttributes(eip),
					testAccCheckElasticIPCreateAttributes("ch-dk-2"),
				),
			},
		},
	})
}

func testAccCheckElasticIPExists(n string, eip *egoscale.IPAddress) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No elastic IP ID is set")
		}

		client := GetComputeClient(testAccProvider.Meta())
		eip.ID = rs.Primary.ID
		if err := client.Get(eip); err != nil {
			return err
		}

		return nil
	}
}

func testAccCheckElasticIPAttributes(eip *egoscale.IPAddress) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if eip.IPAddress == nil {
			return fmt.Errorf("eip IP address is nil")
		}

		return nil
	}
}

func testAccCheckElasticIPCreateAttributes(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_ipaddress" {
				continue
			}

			if rs.Primary.Attributes["zone"] != name {
				continue
			}

			if rs.Primary.Attributes["ip_address"] == "" {
				return fmt.Errorf("Elastic IP: expected ip address to be set")
			}

			return nil
		}

		return fmt.Errorf("Could not find elastic ip %s", name)
	}
}

func testAccCheckElasticIPDestroy(s *terraform.State) error {
	client := GetComputeClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_ipaddress" {
			continue
		}

		key := &egoscale.IPAddress{
			ID:        rs.Primary.ID,
			IsElastic: true,
		}
		if err := client.Get(key); err != nil {
			if r, ok := err.(*egoscale.ErrorResponse); ok {
				if r.ErrorCode == egoscale.ParamError {
					return nil
				}
			}
			return err
		}
		return fmt.Errorf("ipAddress: %#v still exists", key)
	}
	return nil
}

var testAccElasticIPCreate = `
resource "exoscale_ipaddress" "eip" {
  zone = "ch-dk-2"
  tags {
    test = "acceptance"
  }
}
`
