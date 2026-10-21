package kms_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func TestKMSKey(t *testing.T) {
	t.Parallel()

	fullResourceName := "exoscale_kms_key.test"

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: testutils.TestZoneName,
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.ParseTestdataConfig(
					"./testdata/001.kms_key_create.tf.tmpl",
					&testdataSpec,
				),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullResourceName, "id"),
					resource.TestCheckResourceAttr(fullResourceName, "name", testutils.ResourceName(testdataSpec.ID)),
					resource.TestCheckResourceAttr(fullResourceName, "description", "acceptance test key"),
					resource.TestCheckResourceAttr(fullResourceName, "zone", testutils.TestZoneName),
					resource.TestCheckResourceAttr(fullResourceName, "usage", "encrypt-decrypt"),
					resource.TestCheckResourceAttr(fullResourceName, "status", "enabled"),
				),
			},
			// Import
			{
				ResourceName: fullResourceName,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(s *terraform.State) (string, error) {
						return fmt.Sprintf(
							"%s@%s",
							s.RootModule().Resources[fullResourceName].Primary.ID,
							testdataSpec.Zone,
						), nil
					}
				}(),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
