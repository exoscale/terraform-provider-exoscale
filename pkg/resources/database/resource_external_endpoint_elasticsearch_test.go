package database_test

import (
	"bytes"
	"fmt"
	"testing"
	"text/template"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

type TemplateModelExternalEndpointElasticsearch struct {
	ResourceName string
	Name         string
	Zone         string
	URL          string
	IndexPrefix  string
	IndexDaysMax int64
	Timeout      int64
}

func testResourceExternalEndpointElasticsearch(t *testing.T) {
	t.Parallel()

	tpl, err := template.ParseFiles("testdata/resource_external_endpoint_elasticsearch.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	fullResourceName := "exoscale_dbaas_external_endpoint_elasticsearch.test"
	rawName := testutils.TestResourceName()
	if len(rawName) > 40 {
		rawName = rawName[:40]
	}
	base := TemplateModelExternalEndpointElasticsearch{
		ResourceName: "test",
		Name:         rawName,
		Zone:         testutils.TestZoneName,
		URL:          "https://test-elasticsearch.example.com:9200",
		IndexPrefix:  "test-logs",
	}

	dataCreate := base
	dataCreate.IndexDaysMax = 3
	dataCreate.Timeout = 10

	bufCreate := &bytes.Buffer{}
	if err = tpl.Execute(bufCreate, &dataCreate); err != nil {
		t.Fatal(err)
	}

	dataUpdate := base
	dataUpdate.IndexDaysMax = 7
	dataUpdate.Timeout = 30

	bufUpdate := &bytes.Buffer{}
	if err = tpl.Execute(bufUpdate, &dataUpdate); err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             checkExternalEndpointDestroy(base.Name),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create
				Config: bufCreate.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullResourceName, "id"),
					resource.TestCheckResourceAttr(fullResourceName, "name", base.Name),
					resource.TestCheckResourceAttr(fullResourceName, "zone", testutils.TestZoneName),
					resource.TestCheckResourceAttr(fullResourceName, "url", base.URL),
					resource.TestCheckResourceAttr(fullResourceName, "index_prefix", base.IndexPrefix),
					resource.TestCheckResourceAttr(fullResourceName, "index_days_max", "3"),
					resource.TestCheckResourceAttr(fullResourceName, "timeout", "10"),
				),
			},
			{
				// Update
				Config: bufUpdate.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullResourceName, "index_days_max", "7"),
					resource.TestCheckResourceAttr(fullResourceName, "timeout", "30"),
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
				ImportStateVerifyIgnore: []string{"ca"},
			},
		},
	})
}
