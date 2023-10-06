package iam_test

import (
	"bytes"
	"html/template"
	"testing"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func testResourceOrgPolicy(t *testing.T) {
	fullResourceName := "exoscale_iam_org_policy.test"
	expression := acctest.RandomWithPrefix(testutils.Prefix)

	tpl, err := template.ParseFiles("../../testutils/testdata/resource_iam_org_policy.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	data := testutils.ResourceIAMOrgPolicyModel{
		ResourceName:           "test",
		DefaultServiceStrategy: "allow",
		Services: map[string]testutils.ResourceIAMPolicyServicesModel{
			"sos": testutils.ResourceIAMPolicyServicesModel{
				Type: "rules",
				Rules: []testutils.ResourceIAMPolicyServiceRules{
					{
						Action:     "deny",
						Expression: expression,
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

	data.Services = map[string]testutils.ResourceIAMPolicyServicesModel{}
	buf = &bytes.Buffer{}
	err = tpl.Execute(buf, &data)
	if err != nil {
		t.Fatal(err)
	}
	configUpdate := buf.String()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create (actually update)
			{
				Config: configCreate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullResourceName, "default_service_strategy", "allow"),
					resource.TestCheckResourceAttr(fullResourceName, "services.%", "1"),
					resource.TestCheckResourceAttr(fullResourceName, "services.sos.type", "rules"),
					resource.TestCheckResourceAttr(fullResourceName, "services.sos.rules.#", "1"),
					resource.TestCheckResourceAttr(fullResourceName, "services.sos.rules.0.action", "deny"),
					resource.TestCheckResourceAttr(fullResourceName, "services.sos.rules.0.expression", expression),
				),
			},
			// Update (reverts to default org policy)
			{
				Config: configUpdate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullResourceName, "default_service_strategy", "allow"),
					resource.TestCheckResourceAttr(fullResourceName, "services.%", "0"),
				),
			},
		},
	})
}
