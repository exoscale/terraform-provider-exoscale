package database_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"
	"text/template"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	v3 "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

type TemplateModelExternalEndpointPrometheus struct {
	ResourceName      string
	Name              string
	Zone              string
	BasicAuthUsername string
	BasicAuthPassword string
}

func testResourceExternalEndpointPrometheus(t *testing.T) {
	t.Parallel()

	tpl, err := template.ParseFiles("testdata/resource_external_endpoint_prometheus.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	fullResourceName := "exoscale_dbaas_external_endpoint_prometheus.test"
	rawName := testutils.TestResourceName()
	if len(rawName) > 40 {
		rawName = rawName[:40]
	}
	base := TemplateModelExternalEndpointPrometheus{
		ResourceName: "test",
		Name:         rawName,
		Zone:         testutils.TestZoneName,
	}

	dataCreate := base
	dataCreate.BasicAuthUsername = "testuser"
	dataCreate.BasicAuthPassword = "s3cr3tpassword"

	bufCreate := &bytes.Buffer{}
	if err = tpl.Execute(bufCreate, &dataCreate); err != nil {
		t.Fatal(err)
	}

	dataUpdate := base
	dataUpdate.BasicAuthUsername = "updateduser"
	dataUpdate.BasicAuthPassword = "newpassword"

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
					resource.TestCheckResourceAttr(fullResourceName, "basic_auth_username", "testuser"),
				),
			},
			{
				// Update
				Config: bufUpdate.String(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(fullResourceName, "basic_auth_username", "updateduser"),
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
				ImportStateVerifyIgnore: []string{"basic_auth_password"},
			},
		},
	})
}

func checkExternalEndpointDestroy(name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		ctx := context.Background()

		clientV3, err := testutils.APIClientV3()
		if err != nil {
			return err
		}

		client, err := utils.SwitchClientZone(ctx, clientV3, testutils.TestZoneName)
		if err != nil {
			return err
		}

		list, err := client.ListDBAASExternalEndpoints(ctx)
		if err != nil {
			return fmt.Errorf("listing external endpoints: %w", err)
		}

		_, err = list.FindDBAASExternalEndpoint(name)
		if err == nil {
			return fmt.Errorf("external endpoint %q still exists", name)
		}
		if errors.Is(err, v3.ErrNotFound) {
			return nil
		}
		return err
	}
}
