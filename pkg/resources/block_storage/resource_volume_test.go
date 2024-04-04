package block_storage_test

import (
	"bytes"
	"fmt"
	"testing"
	"text/template"
	"time"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testResourceVolume(t *testing.T) {
	fullResourceName := "exoscale_block_storage_volume.test_volume"

	testdataSpec := testutils.TestdataSpec{
		ID:   time.Now().UnixNano(),
		Zone: "ch-gva-2",
	}

	// Load tf templates
	tpl, err := template.ParseFiles("./testdata/001.volume_create.tf.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	err = tpl.Execute(buf, &testdataSpec)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: buf.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(
						fullResourceName,
						"name",
						fmt.Sprintf("test_volume_%d", testdataSpec.ID),
					),
					resource.TestCheckResourceAttr(fullResourceName, "size", "120"),
					resource.TestCheckResourceAttr(fullResourceName, "labels.foo", "bar"),
					resource.TestCheckResourceAttrSet(fullResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(fullResourceName, "blocksize"),
					resource.TestCheckResourceAttrSet(fullResourceName, "state"),
				),
			},
			{
				// Import
				ResourceName: fullResourceName,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(s *terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", s.RootModule().Resources[fullResourceName].Primary.ID, "ch-gva-2"), nil
					}
				}(),
				ImportState: true,
			},
		},
	})
}
