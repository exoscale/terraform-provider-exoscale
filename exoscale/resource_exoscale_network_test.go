package exoscale

import (
	"errors"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/egoscale"
)

var (
	testAccResourceNetworkZoneName       = testZoneName
	testAccResourceNetworkName           = acctest.RandomWithPrefix(testPrefix)
	testAccResourceNetworkNameUpdated    = testAccResourceNetworkName + "-updated"
	testAccResourceNetworkDisplayText    = testDescription
	testAccResourceNetworkStartIP        = "10.0.0.10"
	testAccResourceNetworkStartIPUpdated = "10.0.0.1"
	testAccResourceNetworkEndIP          = "10.0.0.50"
	testAccResourceNetworkEndIPUpdated   = "10.0.0.100"
	testAccResourceNetworkNetmask        = "255.255.0.0"
	testAccResourceNetworkNetmaskUpdated = "255.0.0.0"

	testAccResourceNetworkConfigCreate = fmt.Sprintf(`
resource "exoscale_network" "net" {
  zone = "%s"
  name = "%s"
  display_text = "%s"

  start_ip = "%s"
  end_ip = "%s"
  netmask = "%s"

  tags = {
    managedby = "terraform"
  }
}
`,
		testAccResourceNetworkZoneName,
		testAccResourceNetworkName,
		testAccResourceNetworkDisplayText,
		testAccResourceNetworkStartIP,
		testAccResourceNetworkEndIP,
		testAccResourceNetworkNetmask,
	)

	testAccResourceNetworkConfigUpdate = fmt.Sprintf(`
resource "exoscale_network" "net" {
  zone = "%s"
  name = "%s"

  start_ip = "%s"
  end_ip = "%s"
  netmask = "%s"
}
`,
		testAccResourceNetworkZoneName,
		testAccResourceNetworkNameUpdated,
		testAccResourceNetworkStartIPUpdated,
		testAccResourceNetworkEndIPUpdated,
		testAccResourceNetworkNetmaskUpdated,
	)
)

func TestAccResourceNetwork(t *testing.T) {
	network := new(egoscale.Network)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceNetworkDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceNetworkConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceNetworkExists("exoscale_network.net", network),
					testAccCheckResourceNetwork(network),
					testAccCheckResourceNetworkAttributes(testAttrs{
						"name":           validateString(testAccResourceNetworkName),
						"start_ip":       validateString(testAccResourceNetworkStartIP),
						"end_ip":         validateString(testAccResourceNetworkEndIP),
						"netmask":        validateString(testAccResourceNetworkNetmask),
						"tags.managedby": validateString("terraform"),
					}),
				),
			},
			{
				Config: testAccResourceNetworkConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceNetworkExists("exoscale_network.net", network),
					testAccCheckResourceNetwork(network),
					testAccCheckResourceNetworkAttributes(testAttrs{
						"name":         validateString(testAccResourceNetworkNameUpdated),
						"display_text": validateString(testAccResourceNetworkDisplayText),
						"start_ip":     validateString(testAccResourceNetworkStartIPUpdated),
						"end_ip":       validateString(testAccResourceNetworkEndIPUpdated),
						"netmask":      validateString(testAccResourceNetworkNetmaskUpdated),
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
							"name":         validateString(testAccResourceNetworkNameUpdated),
							"display_text": validateString(testAccResourceNetworkDisplayText),
							"start_ip":     validateString(testAccResourceNetworkStartIPUpdated),
							"end_ip":       validateString(testAccResourceNetworkEndIPUpdated),
							"netmask":      validateString(testAccResourceNetworkNetmaskUpdated),
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

func testAccCheckResourceNetwork(network *egoscale.Network) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if network.ID == nil {
			return errors.New("Network is nil")
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
			if errors.Is(err, egoscale.ErrNotFound) {
				return nil
			}
			return err
		}
	}

	return errors.New("Network still exists")
}
