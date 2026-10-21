package database_test

import (
	"bytes"
	"fmt"
	"testing"
	"text/template"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

type TemplateModelExternalEndpointDatadog struct {
	ResourceName  string
	Name          string
	Zone          string
	DatadogAPIKey string
	Site          string
}

func testResourceExternalEndpointDatadog(t *testing.T) {
	t.Parallel()

	tpl, err := template.ParseFiles("testdata/resource_external_endpoint_datadog.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	fullResourceName := "exoscale_dbaas_external_endpoint_datadog.test"
	rawName := acctest.RandomWithPrefix(testutils.Prefix)
	if len(rawName) > 40 {
		rawName = rawName[:40]
	}
	data := TemplateModelExternalEndpointDatadog{
		ResourceName:  "test",
		Name:          rawName,
		Zone:          testutils.TestZoneName,
		DatadogAPIKey: "abcdef1234567890abcdef1234567890",
		Site:          "datadoghq.com",
	}

	buf := &bytes.Buffer{}
	if err = tpl.Execute(buf, &data); err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             checkExternalEndpointDestroy(data.Name),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create
				Config: buf.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullResourceName, "id"),
					resource.TestCheckResourceAttr(fullResourceName, "name", data.Name),
					resource.TestCheckResourceAttr(fullResourceName, "zone", testutils.TestZoneName),
					resource.TestCheckResourceAttr(fullResourceName, "site", "datadoghq.com"),
				),
			},
			{
				// Import
				ResourceName: fullResourceName,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[fullResourceName]
					if !ok {
						return "", fmt.Errorf("resource %q not found in state", fullResourceName)
					}
					return fmt.Sprintf("%s@%s", rs.Primary.Attributes["id"], testutils.TestZoneName), nil
				},
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"datadog_api_key"},
			},
		},
	})
}
