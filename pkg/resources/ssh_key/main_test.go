package ssh_key_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"testing"
	"time"

	exoscale "github.com/exoscale/egoscale/v3"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func TestSSHKey(t *testing.T) {
	t.Parallel()

	r := "exoscale_ssh_key.test"

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}
	name := testutils.ResourceName(testdataSpec.ID)

	var created, replaced exoscale.SSHKey

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		CheckDestroy:             checkSSHKeyDestroy(name),
		Steps: []resource.TestStep{
			{
				Config:      testutils.ParseTestdataConfig("./testdata/001.ssh_key_missing_public_key.tf.tmpl", &testdataSpec),
				ExpectError: regexp.MustCompile(`The argument "public_key" is required`),
			},
			{
				Config: testutils.ParseTestdataConfig("./testdata/002.ssh_key_create.tf.tmpl", &testdataSpec),
				Check: resource.ComposeAggregateTestCheckFunc(
					checkSSHKeyExists(r, &created),
					resource.TestCheckResourceAttr(r, "id", name),
					resource.TestCheckResourceAttr(r, "name", name),
					resource.TestCheckResourceAttrPtr(r, "fingerprint", &created.Fingerprint),
				),
			},
			{
				// A new public key replaces the SSH key.
				Config: testutils.ParseTestdataConfig("./testdata/003.ssh_key_replace.tf.tmpl", &testdataSpec),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(r, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					checkSSHKeyExists(r, &replaced),
					resource.TestCheckResourceAttrPtr(r, "fingerprint", &replaced.Fingerprint),
					func(*terraform.State) error {
						if replaced.Fingerprint == created.Fingerprint {
							return errors.New("SSH key was not replaced")
						}
						return nil
					},
				),
			},
			{
				// The API doesn't return the public key.
				ResourceName:            r,
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"public_key", "timeouts"},
			},
		},
	})
}

func checkSSHKeyExists(r string, sshKey *exoscale.SSHKey) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		client, err := testutils.APIClientV3()
		if err != nil {
			return err
		}

		res, err := client.GetSSHKey(context.Background(), rs.Primary.ID)
		if err != nil {
			return err
		}

		*sshKey = *res
		return nil
	}
}

func checkSSHKeyDestroy(name string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		client, err := testutils.APIClientV3()
		if err != nil {
			return err
		}

		_, err = client.GetSSHKey(context.Background(), name)
		if errors.Is(err, exoscale.ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}

		return fmt.Errorf("SSH key %q still exists", name)
	}
}
