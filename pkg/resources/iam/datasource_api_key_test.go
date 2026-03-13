package iam_test

import (
	"bytes"
	"html/template"
	"testing"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testDataSourceAPIKey(t *testing.T) {
	t.Parallel()

	var (
		roleName         string = acctest.RandomWithPrefix(testutils.Prefix + "-role")
		apiKeyName       string = acctest.RandomWithPrefix(testutils.Prefix + "-api-key")
		fullResourceName string = "data.exoscale_iam_api_key.test"
	)

	// Role
	tpl1, err := template.ParseFiles("../../testutils/testdata/resource_iam_role.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	data1 := testutils.ResourceIAMRole{
		ResourceName: "test",
		Name:         roleName,
		Description:  "Foo Bar",
		Policy: &testutils.ResourceIAMOrgPolicyModel{
			DefaultServiceStrategy: "allow",
		},
	}

	buf := &bytes.Buffer{}
	err = tpl1.Execute(buf, &data1)
	if err != nil {
		t.Fatal(err)
	}
	configCreate := buf.String() + "\n"

	// API Key
	tpl2, err := template.ParseFiles("../../testutils/testdata/resource_iam_api_key.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	data2 := testutils.ResourceAPIKeyModel{
		ResourceName: "test",
		Name:         apiKeyName,
		RoleID:       "exoscale_iam_role.test.id",
	}

	buf = &bytes.Buffer{}
	err = tpl2.Execute(buf, &data2)
	if err != nil {
		t.Fatal(err)
	}
	configCreate += buf.String() + "\n"

	// Data Source
	configCreate += `data "exoscale_iam_api_key" "test" {
  key = exoscale_iam_api_key.test.key
}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configCreate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullResourceName, "name", apiKeyName),
					resource.TestCheckResourceAttrSet(fullResourceName, "role_id"),
				),
			},
		},
	})
}
