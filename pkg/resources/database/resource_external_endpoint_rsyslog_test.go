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

type TemplateModelExternalEndpointRsyslog struct {
	ResourceName   string
	Name           string
	Zone           string
	Server         string
	Port           int64
	TLS            bool
	Format         string
	MaxMessageSize int64
	Logline        string
}

func testResourceExternalEndpointRsyslog(t *testing.T) {
	t.Parallel()

	tpl, err := template.ParseFiles("testdata/resource_external_endpoint_rsyslog.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	fullResourceName := "exoscale_dbaas_external_endpoint_rsyslog.test"
	rawName := testutils.TestResourceName()
	if len(rawName) > 40 {
		rawName = rawName[:40]
	}
	base := TemplateModelExternalEndpointRsyslog{
		ResourceName: "test",
		Name:         rawName,
		Zone:         testutils.TestZoneName,
		Server:       "1.2.3.4",
		Port:         514,
		TLS:          false,
		Format:       "rfc3164",
	}

	bufCreate := &bytes.Buffer{}
	if err = tpl.Execute(bufCreate, &base); err != nil {
		t.Fatal(err)
	}

	dataUpdate := base
	dataUpdate.Format = "rfc5424"
	dataUpdate.MaxMessageSize = 4096

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
					resource.TestCheckResourceAttr(fullResourceName, "server", "1.2.3.4"),
					resource.TestCheckResourceAttr(fullResourceName, "port", "514"),
					resource.TestCheckResourceAttr(fullResourceName, "tls", "false"),
					resource.TestCheckResourceAttr(fullResourceName, "format", "rfc3164"),
				),
			},
			{
				// Update: change format and set max_message_size
				Config: bufUpdate.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullResourceName, "format", "rfc5424"),
					resource.TestCheckResourceAttr(fullResourceName, "max_message_size", "4096"),
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
				ImportStateVerifyIgnore: []string{"ca", "cert", "key"},
			},
		},
	})
}
