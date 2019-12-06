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

const (
	testNetworkDisplayText        = "Terraform Acceptance Test (create)"
	testNetworkStartIP            = "10.0.0.10"
	testNetworkEndIP              = "10.0.0.50"
	testNetworkNetmask            = "255.255.0.0"
	testNetworkDisplayTextUpdated = "Terraform Acceptance Test (update)"
	testNetworkStartIPUpdated     = "10.0.0.1"
	testNetworkEndIPUpdated       = "10.0.0.100"
	testNetworkNetmaskUpdated     = "255.0.0.0"
)

func TestAccResourceNetwork(t *testing.T) {
	network := new(egoscale.Network)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceNetworkDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceNetworkConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceNetworkExists("exoscale_network.net", network),
					testAccCheckResourceNetwork(network, net.ParseIP(testNetworkStartIP)),
					testAccCheckResourceNetworkAttributes(testAttrs{
						"display_text":   ValidateString(testNetworkDisplayText),
						"start_ip":       ValidateString(testNetworkStartIP),
						"end_ip":         ValidateString(testNetworkEndIP),
						"netmask":        ValidateString(testNetworkNetmask),
						"tags.managedby": ValidateString("terraform"),
					}),
				),
			},
			{
				Config: testAccResourceNetworkConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceNetworkExists("exoscale_network.net", network),
					testAccCheckResourceNetwork(network, net.ParseIP(testNetworkStartIPUpdated)),
					testAccCheckResourceNetworkAttributes(testAttrs{
						"display_text": ValidateString(testNetworkDisplayTextUpdated),
						"start_ip":     ValidateString(testNetworkStartIPUpdated),
						"end_ip":       ValidateString(testNetworkEndIPUpdated),
						"netmask":      ValidateString(testNetworkNetmaskUpdated),
					}),
				),
			},
			{
				ResourceName:      "exoscale_network.net",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							"display_text": ValidateString(testNetworkDisplayTextUpdated),
							"start_ip":     ValidateString(testNetworkStartIPUpdated),
							"end_ip":       ValidateString(testNetworkEndIPUpdated),
							"netmask":      ValidateString(testNetworkNetmaskUpdated),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceNetworkExists(name string, network *egoscale.Network) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[name]
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
		network.ID = id
		network.Name = "" // Reset network name to avoid side-effects from previous test steps
		resp, err := client.Get(network)
		if err != nil {
			return err
		}

		return Copy(network, resp.(*egoscale.Network))
	}
}

func testAccCheckResourceNetwork(network *egoscale.Network, expectedStartIP net.IP) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if network.ID == nil {
			return errors.New("Network is nil")
		}

		if !network.StartIP.Equal(expectedStartIP) {
			return fmt.Errorf("expected start IP %v, got %v", expectedStartIP, network.StartIP)
		}

		return nil
	}
}

func testAccCheckResourceNetworkAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_network" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("resource not found in the state")
	}
}

func testAccCheckResourceNetworkDestroy(s *terraform.State) error {
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

	return errors.New("Network still exists")
}

var testAccResourceNetworkConfigCreate = fmt.Sprintf(`
resource "exoscale_network" "net" {
  zone = %q
  name = "terraform-test-network1"
  display_text = %q

  start_ip = %q
  end_ip = %q
  netmask = %q

  tags = {
    managedby = "terraform"
  }
}
`,
	defaultExoscaleZone,
	testNetworkDisplayText,
	testNetworkStartIP,
	testNetworkEndIP,
	testNetworkNetmask,
)

var testAccResourceNetworkConfigUpdate = fmt.Sprintf(`
resource "exoscale_network" "net" {
  zone = %q
  name = "terraform-test-network2"
  display_text = %q

  start_ip = %q
  end_ip = %q
  netmask = %q
}
`,
	defaultExoscaleZone,
	testNetworkDisplayTextUpdated,
	testNetworkStartIPUpdated,
	testNetworkEndIPUpdated,
	testNetworkNetmaskUpdated,
)
