package exoscale

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/assert"
)

var (
	testAccResourcePrivateNetworkDescription        = acctest.RandString(10)
	testAccResourcePrivateNetworkDescriptionUpdated = testAccResourcePrivateNetworkDescription + "-updated"
	testAccResourcePrivateNetworkEndIP              = "10.0.0.50"
	testAccResourcePrivateNetworkEndIPUpdated       = "10.0.0.100"
	testAccResourcePrivateNetworkName               = acctest.RandString(10)
	testAccResourcePrivateNetworkNameUpdated        = testAccResourcePrivateNetworkName + "-updated"
	testAccResourcePrivateNetworkNetmask            = "255.255.0.0"
	testAccResourcePrivateNetworkNetmaskUpdated     = "255.0.0.0"
	testAccResourcePrivateNetworkStartIP            = "10.0.0.10"
	testAccResourcePrivateNetworkStartIPUpdated     = "10.0.0.1"

	testAccResourcePrivateNetworkConfigCreate = fmt.Sprintf(`
resource "exoscale_private_network" "test" {
  zone        = "%s"
  name        = "%s"
  description = "%s"
  start_ip    = "%s"
  end_ip      = "%s"
  netmask     = "%s"
}
`,
		testZoneName,
		testAccResourcePrivateNetworkName,
		testAccResourcePrivateNetworkDescription,
		testAccResourcePrivateNetworkStartIP,
		testAccResourcePrivateNetworkEndIP,
		testAccResourcePrivateNetworkNetmask,
	)

	testAccResourcePrivateNetworkConfigUpdate = fmt.Sprintf(`
resource "exoscale_private_network" "test" {
  zone        = "%s"
  name        = "%s"
  description = "%s"
  start_ip    = "%s"
  end_ip      = "%s"
  netmask     = "%s"
}
`,
		testZoneName,
		testAccResourcePrivateNetworkNameUpdated,
		testAccResourcePrivateNetworkDescriptionUpdated,
		testAccResourcePrivateNetworkStartIPUpdated,
		testAccResourcePrivateNetworkEndIPUpdated,
		testAccResourcePrivateNetworkNetmaskUpdated,
	)
)

func TestAccResourcePrivateNetwork(t *testing.T) {
	var (
		r              = "exoscale_private_network.test"
		privateNetwork egoscale.PrivateNetwork
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourcePrivateNetworkDestroy(&privateNetwork),
		Steps: []resource.TestStep{
			{
				// Create
				Config: testAccResourcePrivateNetworkConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourcePrivateNetworkExists(r, &privateNetwork),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(testAccResourcePrivateNetworkDescription, *privateNetwork.Description)
						a.Equal(testAccResourcePrivateNetworkName, *privateNetwork.Name)
						a.True(privateNetwork.StartIP.Equal(net.ParseIP(testAccResourcePrivateNetworkStartIP)))
						a.True(privateNetwork.EndIP.Equal(net.ParseIP(testAccResourcePrivateNetworkEndIP)))
						a.True(privateNetwork.Netmask.Equal(net.ParseIP(testAccResourcePrivateNetworkNetmask)))

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resPrivateNetworkAttrDescription: validateString(testAccResourcePrivateNetworkDescription),
						resPrivateNetworkAttrEndIP:       validateString(testAccResourcePrivateNetworkEndIP),
						resPrivateNetworkAttrName:        validateString(testAccResourcePrivateNetworkName),
						resPrivateNetworkAttrNetmask:     validateString(testAccResourcePrivateNetworkNetmask),
						resPrivateNetworkAttrStartIP:     validateString(testAccResourcePrivateNetworkStartIP),
					})),
				),
			},
			{
				// Update
				Config: testAccResourcePrivateNetworkConfigUpdate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourcePrivateNetworkExists(r, &privateNetwork),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(testAccResourcePrivateNetworkDescriptionUpdated, *privateNetwork.Description)
						a.Equal(testAccResourcePrivateNetworkNameUpdated, *privateNetwork.Name)
						a.True(privateNetwork.StartIP.Equal(net.ParseIP(testAccResourcePrivateNetworkStartIPUpdated)))
						a.True(privateNetwork.EndIP.Equal(net.ParseIP(testAccResourcePrivateNetworkEndIPUpdated)))
						a.True(privateNetwork.Netmask.Equal(net.ParseIP(testAccResourcePrivateNetworkNetmaskUpdated)))

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resPrivateNetworkAttrDescription: validateString(testAccResourcePrivateNetworkDescriptionUpdated),
						resPrivateNetworkAttrEndIP:       validateString(testAccResourcePrivateNetworkEndIPUpdated),
						resPrivateNetworkAttrName:        validateString(testAccResourcePrivateNetworkNameUpdated),
						resPrivateNetworkAttrNetmask:     validateString(testAccResourcePrivateNetworkNetmaskUpdated),
						resPrivateNetworkAttrStartIP:     validateString(testAccResourcePrivateNetworkStartIPUpdated),
					})),
				),
			},
			{
				// Import
				ResourceName:      r,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(privateNetwork *egoscale.PrivateNetwork) resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", *privateNetwork.ID, testZoneName), nil
					}
				}(&privateNetwork),
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resPrivateNetworkAttrDescription: validateString(testAccResourcePrivateNetworkDescriptionUpdated),
							resPrivateNetworkAttrEndIP:       validateString(testAccResourcePrivateNetworkEndIPUpdated),
							resPrivateNetworkAttrName:        validateString(testAccResourcePrivateNetworkNameUpdated),
							resPrivateNetworkAttrNetmask:     validateString(testAccResourcePrivateNetworkNetmaskUpdated),
							resPrivateNetworkAttrStartIP:     validateString(testAccResourcePrivateNetworkStartIPUpdated),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourcePrivateNetworkExists(r string, privateNetwork *egoscale.PrivateNetwork) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client := GetComputeClient(testAccProvider.Meta())

		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testEnvironment, testZoneName))
		res, err := client.GetPrivateNetwork(ctx, testZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*privateNetwork = *res
		return nil
	}
}

func testAccCheckResourcePrivateNetworkDestroy(privateNetwork *egoscale.PrivateNetwork) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testEnvironment, testZoneName))

		_, err := client.GetPrivateNetwork(ctx, testZoneName, *privateNetwork.ID)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("Private Network still exists")
	}
}
