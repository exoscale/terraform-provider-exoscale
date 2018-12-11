package exoscale

import (
	"fmt"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccSSHKeyPair(t *testing.T) {
	sshkey := new(egoscale.SSHKeyPair)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckSSHKeyPairDestroy,
		Steps: []resource.TestStep{
			resource.TestStep{
				Config: testAccSSHKeyPairCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckSSHKeyPairExists("exoscale_ssh_keypair.key", sshkey),
					testAccCheckSSHKeyPairAttributes(sshkey),
					testAccCheckSSHKeyPairCreateAttributes("terraform-test-keypair"),
				),
			},
		},
	})
}

func testAccCheckSSHKeyPairExists(n string, sshkey *egoscale.SSHKeyPair) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No key pair ID is set")
		}

		client := GetComputeClient(testAccProvider.Meta())
		sshkey.Name = rs.Primary.ID
		resp, err := client.Get(sshkey)
		if err != nil {
			return err
		}

		return Copy(sshkey, resp.(*egoscale.SSHKeyPair))
	}
}

func testAccCheckSSHKeyPairAttributes(sshkey *egoscale.SSHKeyPair) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(sshkey.Fingerprint) != 47 {
			return fmt.Errorf("SSH Key: fingerprint length doesn't match")
		}

		return nil
	}
}

func testAccCheckSSHKeyPairCreateAttributes(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_ssh_keypair" {
				continue
			}

			if rs.Primary.ID != name {
				continue
			}

			if rs.Primary.Attributes["private_key"] == "" {
				return fmt.Errorf("SSH key: expected private key to be set")
			}

			return nil
		}

		return fmt.Errorf("Could not find key pair %s", name)
	}
}

func testAccCheckSSHKeyPairDestroy(s *terraform.State) error {
	client := GetComputeClient(testAccProvider.Meta())

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "exoscale_ssh_keypair" {
			continue
		}

		key := &egoscale.SSHKeyPair{Name: rs.Primary.ID}
		_, err := client.Get(key)
		if err != nil {
			if r, ok := err.(*egoscale.ErrorResponse); ok {
				if r.ErrorCode == egoscale.ParamError {
					return nil
				}
			}
			return err
		}
	}
	return fmt.Errorf("SSH key: still exists")
}

var testAccSSHKeyPairCreate = `
resource "exoscale_ssh_keypair" "key" {
  name = "terraform-test-keypair"
}
`
