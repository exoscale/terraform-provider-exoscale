package iam_test

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testResourceAPIKey(t *testing.T) {
	fullResourceName := "exoscale_iam_api_key.test"

	// Role
	tpl1, err := template.ParseFiles("../../testutils/testdata/resource_iam_role.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	data1 := testutils.ResourceIAMRole{
		ResourceName: "test",
		Name:         "test",
		Description:  "foo bar",
		Editable:     true,
		Labels:       map[string]string{"foo": "bar"},

		Policy: &testutils.ResourceIAMOrgPolicyModel{
			DefaultServiceStrategy: "allow",
			Services: map[string]testutils.ResourceIAMPolicyServicesModel{
				"sos": {
					Type: "rules",
					Rules: []testutils.ResourceIAMPolicyServiceRules{
						{
							Action:     "deny",
							Expression: "test",
						},
					},
				},
			},
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
		Name:         "test",
		RoleID:       "exoscale_iam_role.test.id",
	}

	buf = &bytes.Buffer{}
	err = tpl2.Execute(buf, &data2)
	if err != nil {
		t.Fatal(err)
	}
	configCreate += buf.String()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: configCreate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullResourceName, "name", "test"),
					resource.TestCheckResourceAttrSet(fullResourceName, "role_id"),
					resource.TestCheckResourceAttrSet(fullResourceName, "key"),
					resource.TestCheckResourceAttrSet(fullResourceName, "secret"),
				),
			},
			{
				// Import
				ResourceName: fullResourceName,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(s *terraform.State) (string, error) {
						return s.RootModule().Resources[fullResourceName].Primary.ID, nil
					}
				}(),
				ImportState:       true,
				ImportStateVerify: true,
				// Secret is not available after creation so we are ignoring it.
				ImportStateVerifyIgnore: []string{"secret"},
			},
		},
	})
}
