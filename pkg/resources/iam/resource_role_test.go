package iam_test

import (
	"bytes"
	"testing"
	"text/template"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	tfresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func testResourceRole(t *testing.T) {
	t.Parallel()

	var (
		roleName         string = acctest.RandomWithPrefix(testutils.Prefix + "-role")
		fullResourceName string = "exoscale_iam_role.test"
	)

	tpl, err := template.ParseFiles("../../testutils/testdata/resource_iam_role.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	data := testutils.ResourceIAMRole{
		ResourceName: "test",
		Name:         roleName,
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

	tfresource.Test(t, tfresource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []tfresource.TestStep{
			// Create
			{
				Config: configCreate,
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(fullResourceName, "name", roleName),
					tfresource.TestCheckResourceAttr(fullResourceName, "description", "foo bar"),
					tfresource.TestCheckResourceAttr(fullResourceName, "editable", "true"),
					tfresource.TestCheckResourceAttr(fullResourceName, "labels.foo", "bar"),
					tfresource.TestCheckResourceAttr(fullResourceName, "policy.default_service_strategy", "allow"),
					tfresource.TestCheckResourceAttr(fullResourceName, "policy.services.%", "1"),
					tfresource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.type", "rules"),
					tfresource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.rules.#", "1"),
					tfresource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.rules.0.action", "allow"),
					tfresource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.rules.0.expression", "operation in ['list-sos-buckets-usage', 'list-buckets']"),
				),
			},
			// Update
			{
				Config: configUpdate,
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(fullResourceName, "permissions.#", "1"),
					tfresource.TestCheckResourceAttr(fullResourceName, "permissions.0", "bypass-governance-retention"),
				),
			},
			// Update policy
			{
				Config: configUpdatePolicy,
				Check: tfresource.ComposeAggregateTestCheckFunc(
					tfresource.TestCheckResourceAttr(fullResourceName, "policy.default_service_strategy", "deny"),
					tfresource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.type", "rules"),
					tfresource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.rules.#", "2"),
					tfresource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.rules.0.action", "allow"),
					tfresource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.rules.0.expression", "operation in ['list-sos-buckets-usage', 'list-buckets']"),
					tfresource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.rules.1.action", "deny"),
					tfresource.TestCheckResourceAttr(fullResourceName, "policy.services.sos.rules.1.expression", "operation in ['list-objects', 'get-object']"),
				),
			},
			{
				// Import
				ResourceName: fullResourceName,
				ImportStateIdFunc: func() tfresource.ImportStateIdFunc {
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
