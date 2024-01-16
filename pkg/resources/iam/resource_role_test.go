package iam_test

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testResourceRole(t *testing.T) {
	fullResourceName := "exoscale_iam_role.test"

	tpl, err := template.ParseFiles("../../testutils/testdata/resource_iam_role.tmpl")
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
							Action:     "allow",
							Expression: "operation in ['list-sos-buckets-usage', 'list-buckets']",
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
	configCreate := buf.String()

	data.Permissions = "bypass-governance-retention"
	buf = &bytes.Buffer{}
	err = tpl.Execute(buf, &data)
	if err != nil {
		t.Fatal(err)
	}
	configUpdate := buf.String()

	policy := testutils.ResourceIAMOrgPolicyModel{
		DefaultServiceStrategy: "deny",
		Services: map[string]testutils.ResourceIAMPolicyServicesModel{
			"sos": {
				Type: "rules",
				Rules: []testutils.ResourceIAMPolicyServiceRules{
					{
						Action:     "allow",
						Expression: "operation in ['list-sos-buckets-usage', 'list-buckets']",
					},
					{
						Action:     "deny",
						Expression: "operation in ['list-objects', 'get-object']",
					},
				},
			},
		},
	}
	data.Policy = &policy

	buf = &bytes.Buffer{}
	err = tpl.Execute(buf, &data)
	if err != nil {
		t.Fatal(err)
	}
	configUpdatePolicy := buf.String()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create
			{
				Config: configCreate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullResourceName, "name", "test"),
					resource.TestCheckResourceAttr(fullResourceName, "description", "foo bar"),
					resource.TestCheckResourceAttr(fullResourceName, "editable", "true"),
					resource.TestCheckResourceAttr(fullResourceName, "labels.foo", "bar"),
					resource.TestCheckResourceAttr(fullResourceName, "policy.default_service_strategy", "allow"),
					resource.TestCheckResourceAttr(fullResourceName, "policy.services.%", "1"),
					resource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.type", "rules"),
					resource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.rules.#", "1"),
					resource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.rules.0.action", "allow"),
					resource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.rules.0.expression", "operation in ['list-sos-buckets-usage', 'list-buckets']"),
				),
			},
			// Update
			{
				Config: configUpdate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullResourceName, "permissions.#", "1"),
					resource.TestCheckResourceAttr(fullResourceName, "permissions.0", "bypass-governance-retention"),
				),
			},
			// Update policy
			{
				Config: configUpdatePolicy,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullResourceName, "policy.default_service_strategy", "deny"),
					resource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.type", "rules"),
					resource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.rules.#", "2"),
					resource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.rules.0.action", "allow"),
					resource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.rules.0.expression", "operation in ['list-sos-buckets-usage', 'list-buckets']"),
					resource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.rules.1.action", "deny"),
					resource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.rules.1.expression", "operation in ['list-objects', 'get-object']"),
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
			},
		},
	})
}
