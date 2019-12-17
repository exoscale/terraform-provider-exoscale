package exoscale

import (
	"errors"
	"fmt"
	"testing"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

var (
	testAccResourceSSHKeyName = testPrefix + "-" + testRandomString()
	testAccResourceSSHKey     = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDN7L45b4vO2ytH68ZU" +
		"C5PMS1b7JG78zGslwcJ0zolE5BuxsCYor248/FKGC5TXrME+yBu/uLqaAkioq4Wp1PzP6Zy5jEowWQDO" +
		"deER7uu1GgZShcvly2Oaf/UKLqTdwL+U3tCknqHY63fOAi1lBwmNTUu1uZ24iNiogfhXwQn7HJLQK9vf" +
		"oGwg+/qJIzeswR6XDa6qh0fuzdxWQ4JWHw2s8fv8WvGOlklmAg/uEi1kF5D6R7kJpOVaE20FLnT4sjA8" +
		"1iErhMIH77OaZqQKiyVH3i5m/lkQI/2e25ml8aculaWzHOs4ctd7l+K1ZWFYje3qMBY1sar1gd787eaqk6RZ"
	testAccResourceSSHKeyFingerprint = "4d:31:21:c4:77:9f:19:91:6e:84:9d:7c:12:a8:11:1f"

	testAccResourceSSHKeypairConfig = fmt.Sprintf(`
resource "exoscale_ssh_keypair" "key" {
  name       = "%s"
  public_key = "%s"
}
`,
		testAccResourceSSHKeyName,
		testAccResourceSSHKey)
)

func TestAccResourceSSHKeypair(t *testing.T) {
	sshkey := new(egoscale.SSHKeyPair)

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckResourceSSHKeypairDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccResourceSSHKeypairConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSSHKeypairExists("exoscale_ssh_keypair.key", sshkey),
					testAccCheckResourceSSHKeypair(sshkey),
					testAccCheckResourceSSHKeypairAttributes(testAttrs{
						"public_key":  ValidateString(testAccResourceSSHKey),
						"fingerprint": ValidateString(testAccResourceSSHKeyFingerprint),
					}),
				),
			},
			{
				ResourceName:            "exoscale_ssh_keypair.key",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"public_key"},
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							"name":        ValidateString(testAccResourceSSHKeyName),
							"fingerprint": ValidateString(testAccResourceSSHKeyFingerprint),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceSSHKeypairExists(n string, sshkey *egoscale.SSHKeyPair) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
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

func testAccCheckResourceSSHKeypair(sshkey *egoscale.SSHKeyPair) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(sshkey.Fingerprint) != 47 {
			return fmt.Errorf("expected SSH fingerprint length %d, got %d", 47, len(sshkey.Fingerprint))
		}

		return nil
	}
}

func testAccCheckResourceSSHKeypairAttributes(expected testAttrs) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "exoscale_ssh_keypair" {
				continue
			}

			return checkResourceAttributes(expected, rs.Primary.Attributes)
		}

		return errors.New("resource not found in the state")
	}
}

func testAccCheckResourceSSHKeypairDestroy(s *terraform.State) error {
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
	return errors.New("SSH Keypair still exists")
}
