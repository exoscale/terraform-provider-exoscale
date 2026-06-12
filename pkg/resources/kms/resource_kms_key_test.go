package kms_test

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

func testResourceKMSKey(t *testing.T) {
	t.Parallel()

	keyName := acctest.RandomWithPrefix(testutils.Prefix + "-kms-key")
	fullResourceName := "exoscale_kms_key.test"

	tpl, err := template.ParseFiles("../../testutils/testdata/resource_kms_key.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	data := testutils.ResourceKMSKeyModel{
		ResourceName: "test",
		Name:         keyName,
		Description:  "acceptance test key",
		Zone:         testutils.TestZoneName,
		Usage:        "encrypt-decrypt",
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, &data); err != nil {
		t.Fatal(err)
	}
	config := buf.String()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullResourceName, "id"),
					resource.TestCheckResourceAttr(fullResourceName, "name", keyName),
					resource.TestCheckResourceAttr(fullResourceName, "description", "acceptance test key"),
					resource.TestCheckResourceAttr(fullResourceName, "zone", testutils.TestZoneName),
					resource.TestCheckResourceAttr(fullResourceName, "usage", "encrypt-decrypt"),
					resource.TestCheckResourceAttr(fullResourceName, "status", "enabled"),
				),
			},
		},
	})
}
