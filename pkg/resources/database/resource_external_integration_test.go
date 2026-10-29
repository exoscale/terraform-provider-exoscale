package database_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"text/template"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	v3 "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

const envDBAASPostgresService = "EXOSCALE_TEST_DBAAS_POSTGRES_SERVICE"

type TemplateModelExternalIntegration struct {
	EndpointResourceName string
	EndpointName         string
	ResourceName         string
	SourceServiceName    string
	Type                 string
	Zone                 string
}

func testResourceExternalIntegration(t *testing.T) {
	t.Parallel()

	pgServiceName := os.Getenv(envDBAASPostgresService)
	if pgServiceName == "" {
		t.Skipf("env %s not set; skipping external integration acceptance test "+
			"(set it to the name of a pre-existing PostgreSQL DBaaS service in zone %s)",
			envDBAASPostgresService, testutils.TestZoneName)
	}

	tpl, err := template.ParseFiles("testdata/resource_external_integration.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	fullIntegrationResource := "exoscale_dbaas_external_integration.test"
	fullEndpointResource := "exoscale_dbaas_external_endpoint_prometheus.test_endpoint"

	rawEndpointName := acctest.RandomWithPrefix(testutils.Prefix)
	if len(rawEndpointName) > 40 {
		rawEndpointName = rawEndpointName[:40]
	}
	endpointName := rawEndpointName
	data := TemplateModelExternalIntegration{
		EndpointResourceName: "test_endpoint",
		EndpointName:         endpointName,
		ResourceName:         "test",
		SourceServiceName:    pgServiceName,
		Type:                 "prometheus",
		Zone:                 testutils.TestZoneName,
	}

	buf := &bytes.Buffer{}
	if err = tpl.Execute(buf, &data); err != nil {
		t.Fatal(err)
	}
	config := buf.String()

	// integrationID is populated during the Check step so CheckDestroy can use it.
	var integrationID string

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testutils.AccPreCheck(t) },
		CheckDestroy: resource.ComposeTestCheckFunc(
			checkExternalEndpointDestroy(endpointName),
			func(_ *terraform.State) error {
				if integrationID == "" {
					return nil
				}
				return checkExternalIntegrationDestroyed(integrationID)
			},
		),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullIntegrationResource, "id"),
					resource.TestCheckResourceAttr(fullIntegrationResource, "source_service_name", pgServiceName),
					resource.TestCheckResourceAttr(fullIntegrationResource, "type", "prometheus"),
					resource.TestCheckResourceAttr(fullIntegrationResource, "zone", testutils.TestZoneName),
					resource.TestCheckResourceAttrSet(fullIntegrationResource, "dest_endpoint_id"),
					resource.TestCheckResourceAttrSet(fullIntegrationResource, "dest_endpoint_name"),
					resource.TestCheckResourceAttrSet(fullIntegrationResource, "source_service_type"),
					resource.TestCheckResourceAttrPair(
						fullIntegrationResource, "dest_endpoint_id",
						fullEndpointResource, "id",
					),
					// Capture integration ID for CheckDestroy.
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources[fullIntegrationResource]
						if ok && rs.Primary != nil {
							integrationID = rs.Primary.Attributes["id"]
						}
						return nil
					},
				),
			},
			{
				// Import
				ResourceName: fullIntegrationResource,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[fullIntegrationResource]
					if !ok {
						return "", fmt.Errorf("resource %q not found in state", fullIntegrationResource)
					}
					return fmt.Sprintf("%s@%s", rs.Primary.Attributes["id"], testutils.TestZoneName), nil
				},
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func checkExternalIntegrationDestroyed(integrationID string) error {
	ctx := context.Background()

	clientV3, err := testutils.APIClientV3()
	if err != nil {
		return err
	}

	client, err := utils.SwitchClientZone(ctx, clientV3, testutils.TestZoneName)
	if err != nil {
		return err
	}

	id, err := v3.ParseUUID(integrationID)
	if err != nil {
		return fmt.Errorf("parsing integration ID %q: %w", integrationID, err)
	}

	_, err = client.GetDBAASExternalIntegration(ctx, id)
	if err == nil {
		return fmt.Errorf("external integration %q still exists", integrationID)
	}
	if errors.Is(err, v3.ErrNotFound) {
		return nil
	}
	return err
}
