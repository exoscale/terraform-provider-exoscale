package exoscale

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/assert"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
)

var (
	testAccResourceSSHKeyName      = acctest.RandomWithPrefix(testPrefix)
	testAccResourceSSHKeyPublicKey = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDN7L45b4vO2ytH68ZU" +
		"C5PMS1b7JG78zGslwcJ0zolE5BuxsCYor248/FKGC5TXrME+yBu/uLqaAkioq4Wp1PzP6Zy5jEowWQDO" +
		"deER7uu1GgZShcvly2Oaf/UKLqTdwL+U3tCknqHY63fOAi1lBwmNTUu1uZ24iNiogfhXwQn7HJLQK9vf" +
		"oGwg+/qJIzeswR6XDa6qh0fuzdxWQ4JWHw2s8fv8WvGOlklmAg/uEi1kF5D6R7kJpOVaE20FLnT4sjA8" +
		"1iErhMIH77OaZqQKiyVH3i5m/lkQI/2e25ml8aculaWzHOs4ctd7l+K1ZWFYje3qMBY1sar1gd787eaqk6RZ"

	testAccResourceSSHKeyConfigCreate = fmt.Sprintf(`
resource "exoscale_ssh_key" "test" {
  name       = "%s"
  public_key = "%s"
}
`,
		testAccResourceSSHKeyName,
		testAccResourceSSHKeyPublicKey,
	)
)

func TestAccResourceSSHKey(t *testing.T) {
	var (
		r      = "exoscale_ssh_key.test"
		sshKey egoscale.SSHKey
	)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviders,
		CheckDestroy:      testAccCheckResourceSSHKeyDestroy(&sshKey),
		Steps: []resource.TestStep{
			{
				// Create (missing public_key)
				Config:      `resource "exoscale_ssh_key" "test" { name = "lolnope" }`,
				ExpectError: regexp.MustCompile(`The argument "public_key" is required, but no definition was found.`),
			},
			{
				// Create
				Config: testAccResourceSSHKeyConfigCreate,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckResourceSSHKeyExists(r, &sshKey),
					func(s *terraform.State) error {
						a := assert.New(t)

						a.Equal(testAccResourceSSHKeyName, *sshKey.Name)

						return nil
					},
					checkResourceState(r, checkResourceStateValidateAttributes(testAttrs{
						resSSHKeyAttrFingerprint: validation.ToDiagFunc(validation.NoZeroValues),
						resSSHKeyAttrName:        validateString(testAccResourceSSHKeyName),
					})),
				),
			},
			{
				// Import
				ResourceName:            r,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{resSSHKeyAttrPublicKey},
				ImportStateCheck: func(s []*terraform.InstanceState) error {
					return checkResourceAttributes(
						testAttrs{
							resSSHKeyAttrFingerprint: validation.ToDiagFunc(validation.NoZeroValues),
							resSSHKeyAttrName:        validateString(testAccResourceSSHKeyName),
						},
						s[0].Attributes)
				},
			},
		},
	})
}

func testAccCheckResourceSSHKeyExists(r string, sshKey *egoscale.SSHKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client := getClient(testAccProvider.Meta())

		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testEnvironment, testZoneName))
		res, err := client.GetSSHKey(ctx, testZoneName, rs.Primary.ID)
		if err != nil {
			return err
		}

		*sshKey = *res
		return nil
	}
}

func testAccCheckResourceSSHKeyDestroy(sshKey *egoscale.SSHKey) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := getClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testEnvironment, testZoneName))

		_, err := client.GetSSHKey(ctx, testZoneName, *sshKey.Name)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}

			return err
		}

		return errors.New("SSH Key still exists")
	}
}
