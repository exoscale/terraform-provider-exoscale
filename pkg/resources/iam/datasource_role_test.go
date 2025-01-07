package iam_test

import (
	"bytes"
	"html/template"
	"testing"

	"github.com/exoscale/terraform-provider-exoscale/exoscale/testutils"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testDataSourceRole(t *testing.T) {
	// Role
	tpl, err := template.ParseFiles("./testdata/resource_iam_role.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	data := testutils.ResourceIAMRole{
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
	err = tpl.Execute(buf, &data)
	if err != nil {
		t.Fatal(err)
	}
	config := buf.String() + "\n"

	// Data Source by ID
	fullResourceName1 := "data.exoscale_iam_role.test1"
	config += `data "exoscale_iam_role" "test1" {
  id = exoscale_iam_role.test.id
}
`

	// Data Source by Name
	fullResourceName2 := "data.exoscale_iam_role.test2"
	config += `data "exoscale_iam_role" "test2" {
  name = exoscale_iam_role.test.name
}`

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullResourceName1, "name", "test"),
					resource.TestCheckResourceAttr(fullResourceName1, "description", "foo bar"),
					resource.TestCheckResourceAttr(fullResourceName1, "editable", "true"),
					resource.TestCheckResourceAttr(fullResourceName1, "labels.foo", "bar"),
					resource.TestCheckResourceAttr(fullResourceName1, "policy.default_service_strategy", "allow"),
					resource.TestCheckResourceAttr(fullResourceName1, "policy.services.%", "1"),
					resource.TestCheckResourceAttr(fullResourceName1, "policy.services.sos.type", "rules"),
					resource.TestCheckResourceAttr(fullResourceName1, "policy.services.sos.rules.#", "1"),
					resource.TestCheckResourceAttr(fullResourceName1, "policy.services.sos.rules.0.action", "deny"),
					resource.TestCheckResourceAttr(fullResourceName1, "policy.services.sos.rules.0.expression", "test"),
					resource.TestCheckResourceAttr(fullResourceName2, "description", "foo bar"),
					resource.TestCheckResourceAttr(fullResourceName2, "editable", "true"),
					resource.TestCheckResourceAttr(fullResourceName2, "labels.foo", "bar"),
					resource.TestCheckResourceAttr(fullResourceName2, "policy.default_service_strategy", "allow"),
					resource.TestCheckResourceAttr(fullResourceName2, "policy.services.%", "1"),
					resource.TestCheckResourceAttr(fullResourceName2, "policy.services.sos.type", "rules"),
					resource.TestCheckResourceAttr(fullResourceName2, "policy.services.sos.rules.#", "1"),
					resource.TestCheckResourceAttr(fullResourceName2, "policy.services.sos.rules.0.action", "deny"),
					resource.TestCheckResourceAttr(fullResourceName2, "policy.services.sos.rules.0.expression", "test"),
				),
			},
		},
	})
}
