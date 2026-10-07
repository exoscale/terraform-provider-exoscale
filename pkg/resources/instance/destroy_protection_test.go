package instance_test

import (
	"fmt"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

// The error detail is wrapped by Terraform: words may be separated by a newline.
var destroyProtectionError = regexp.MustCompile(`Forbidden: Operation delete-instance on\s+resource .* is forbidden - reason: manual\s+instance protection`)

func testExplicitDestroyProtection(t *testing.T) {
	t.Parallel()

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1 Create a protected instance.
			{
				Config: testutils.ParseTestdataConfig("./testdata/013.destroy_protection_enabled.tf.tmpl", &testdataSpec),
				Check:  resource.TestCheckResourceAttr(instanceResource, "destroy_protected", "true"),
			},
			// 2 The API refuses to delete it.
			{
				Config:      testutils.ParseTestdataConfig("./testdata/016.instance_removed.tf.tmpl", &testdataSpec),
				ExpectError: destroyProtectionError,
			},
			// 3 Remove the protection.
			{
				Config: testutils.ParseTestdataConfig("./testdata/014.destroy_protection_disabled.tf.tmpl", &testdataSpec),
				Check:  resource.TestCheckResourceAttr(instanceResource, "destroy_protected", "false"),
			},
			// 4 The instance can be deleted.
			{
				Config: testutils.ParseTestdataConfig("./testdata/016.instance_removed.tf.tmpl", &testdataSpec),
				Check:  checkResourceDoesNotExist(instanceResource),
			},
		},
	})
}

func testDefaultDestroyProtection(t *testing.T) {
	t.Parallel()

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// 1 Create an instance without destroy_protected.
			{
				Config: testutils.ParseTestdataConfig("./testdata/015.destroy_protection_unset.tf.tmpl", &testdataSpec),
			},
			// 2 Protect it.
			{
				Config: testutils.ParseTestdataConfig("./testdata/013.destroy_protection_enabled.tf.tmpl", &testdataSpec),
				Check:  resource.TestCheckResourceAttr(instanceResource, "destroy_protected", "true"),
			},
			// 3 The API refuses to delete it.
			{
				Config:      testutils.ParseTestdataConfig("./testdata/016.instance_removed.tf.tmpl", &testdataSpec),
				ExpectError: destroyProtectionError,
			},
			// 4 Dropping destroy_protected from the configuration removes the
			// protection, as if false were the default value.
			{
				Config: testutils.ParseTestdataConfig("./testdata/015.destroy_protection_unset.tf.tmpl", &testdataSpec),
			},
			// 5 The instance can be deleted.
			{
				Config: testutils.ParseTestdataConfig("./testdata/016.instance_removed.tf.tmpl", &testdataSpec),
			},
		},
	})
}

func checkResourceDoesNotExist(name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if _, ok := s.RootModule().Resources[name]; ok {
			return fmt.Errorf("%s was not deleted after its destroy protection was removed", name)
		}

		return nil
	}
}
