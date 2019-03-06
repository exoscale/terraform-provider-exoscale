package exoscale

import (
	"fmt"
	"net"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccNetwork(t *testing.T) {
	net := new(egoscale.Network)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckNetworkDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccNetworkCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkExists("exoscale_network.net", net),
					testAccCheckNetworkAttributes(net),
					testAccCheckNetworkCreateAttributes("terraform-test-network"),
				),
			}, {
				Config: testAccNetworkUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckNetworkExists("exoscale_network.net", net),
					testAccCheckManagedNetworkAttributes(net),
					testAccCheckNetworkCreateAttributes("terraform-test-network"),
				),
			},
		},
	})
}

func testAccCheckNetworkExists(n string, net *egoscale.Network) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Network ID is set")
		}

		id, err := egoscale.ParseUUID(rs.Primary.ID)
		if err != nil {
			return err
		}

		client := GetComputeClient(testAccProvider.Meta())
		net.ID = id
		resp, err := client.Get(net)
		if err != nil {
			return err
		}

		return Copy(net, resp.(*egoscale.Network))
	}
}

func testAccCheckNetworkAttributes(net *egoscale.Network) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if net.ID == nil {
			return fmt.Errorf("Network is nil")
		}

		if net.StartIP != nil {
			return fmt.Errorf("StartIP is not nil, got %s", net.StartIP)
		}

		return nil
	}
}

func testAccCheckManagedNetworkAttributes(nw *egoscale.Network) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if nw.ID == nil {
			return fmt.Errorf("Network is nil")
		}

		if !nw.StartIP.Equal(net.ParseIP("10.0.0.1")) {
			return fmt.Errorf("StartIP doesn't match, got %s", nw.StartIP)
		}

		return nil
	}
}

func testAccCheckNetworkCreateAttributes(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_network" {
				continue
			}

			if rs.Primary.Attributes["name"] != name {
				continue
			}

			if rs.Primary.Attributes["display_text"] == "" {
				return fmt.Errorf("Network: expected display text to be set")
			}

			return nil
		}

		return fmt.Errorf("Could not find Network name: %s", name)
	}
}

func testAccCheckNetworkDestroy(s *terraform.State) error {
	client := GetComputeClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_network" {
			continue
		}

		id, err := egoscale.ParseUUID(rs.Primary.ID)
		if err != nil {
			return err
		}

		key := &egoscale.Network{ID: id}
		_, err = client.Get(key)
		if err != nil {
			if r, ok := err.(*egoscale.ErrorResponse); ok {
				if r.ErrorCode == egoscale.ParamError {
					return nil
				}
			}
			return err
		}
	}
	return fmt.Errorf("Network: still exists")
}

var testAccNetworkCreate = fmt.Sprintf(`
resource "exoscale_network" "net" {
  name = "terraform-test-network"
  display_text = "Terraform Acceptance Test"
  zone = %q
  network_offering = %q

  tags = {
    managedby = "terraform"
  }
}
`,
	defaultExoscaleZone,
	defaultExoscaleNetworkOffering,
)

var testAccNetworkUpdate = fmt.Sprintf(`
resource "exoscale_network" "net" {
  name = "terraform-test-network"
  display_text = "Terraform Acceptance Test"
  zone = %q
  network_offering = %q

  start_ip = "10.0.0.1"
  end_ip = "10.0.0.5"
  netmask = "255.255.255.248"
}
`,
	defaultExoscaleZone,
	defaultExoscaleNetworkOffering,
)
