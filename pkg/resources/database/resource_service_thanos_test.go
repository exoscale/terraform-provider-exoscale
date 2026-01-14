package database_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"text/template"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

type TemplateModelThanos struct {
	ResourceName string

	Name string
	Plan string
	Zone string

	MaintenanceDow        string
	MaintenanceTime       string
	TerminationProtection bool

	IpFilter       []string
	ThanosSettings string
}

func testResourceThanos(t *testing.T) {
	tpl, err := template.ParseFiles("testdata/resource_thanos.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	fullResourceName := "exoscale_dbaas.test"
	dataBase := TemplateModelThanos{
		ResourceName:          "test",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "startup-4",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
	}

	dataCreate := dataBase
	dataCreate.MaintenanceDow = "monday"
	dataCreate.MaintenanceTime = "01:23:00"
	dataCreate.IpFilter = []string{"1.2.3.4/32"}
	buf := &bytes.Buffer{}
	err = tpl.Execute(buf, &dataCreate)
	if err != nil {
		t.Fatal(err)
	}
	configCreate := buf.String()

	dataUpdate := dataBase
	dataUpdate.MaintenanceDow = "tuesday"
	dataUpdate.MaintenanceTime = "02:34:00"
	dataUpdate.IpFilter = []string{"9.1.1.9/32"}
	dataUpdate.ThanosSettings = strconv.Quote(`{"compactor":{"retention.days":30}}`)
	buf = &bytes.Buffer{}
	err = tpl.Execute(buf, &dataUpdate)
	if err != nil {
		t.Fatal(err)
	}
	configUpdate := buf.String()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             CheckServiceDestroy("thanos", dataBase.Name),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create
				Config: configCreate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(fullResourceName, "disk_size"),
					resource.TestCheckResourceAttrSet(fullResourceName, "node_cpus"),
					resource.TestCheckResourceAttrSet(fullResourceName, "node_memory"),
					resource.TestCheckResourceAttrSet(fullResourceName, "nodes"),
					resource.TestCheckResourceAttrSet(fullResourceName, "ca_certificate"),
					resource.TestCheckResourceAttrSet(fullResourceName, "updated_at"),
				),
			},
			{
				// Update
				Config: configUpdate,
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						err := CheckExistsThanos(dataBase.Name, &dataUpdate)
						if err != nil {
							return err
						}

						return nil
					},
				),
			},
			{
				// Import
				ResourceName: fullResourceName,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", dataBase.Name, dataBase.Zone), nil
					}
				}(),
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: strings.Fields("updated_at state"),
			},
		},
	})
}

func CheckExistsThanos(name string, data *TemplateModelThanos) error {
	ctx := context.Background()

	defaultClientV3, err := testutils.APIClientV3()
	if err != nil {
		return err
	}

	client, err := utils.SwitchClientZone(
		ctx,
		defaultClientV3,
		testutils.TestZoneName,
	)
	if err != nil {
		return err
	}

	service, err := client.GetDBAASServiceThanos(ctx, name)
	if err != nil {
		return err
	}

	if data.Plan != service.Plan {
		return fmt.Errorf("plan: expected %q, got %q", data.Plan, service.Plan)
	}

	if *service.TerminationProtection != false {
		return fmt.Errorf("termination_protection: expected false, got true")
	}

	if !cmp.Equal(data.IpFilter, service.IPFilter, cmpopts.EquateEmpty()) {
		return fmt.Errorf("thanos.ip_filter: expected %q, got %q", data.IpFilter, service.IPFilter)
	}

	if v := string(service.Maintenance.Dow); data.MaintenanceDow != v {
		return fmt.Errorf("thanos.maintenance_dow: expected %q, got %q", data.MaintenanceDow, v)
	}

	if data.MaintenanceTime != service.Maintenance.Time {
		return fmt.Errorf("thanos.maintenance_time: expected %q, got %q", data.MaintenanceTime, service.Maintenance.Time)
	}

	if data.ThanosSettings != "" {
		var expectedSettings, actualSettings map[string]interface{}

		// Parse expected settings
		s, err := strconv.Unquote(data.ThanosSettings)
		if err != nil {
			return err
		}
		if err := json.Unmarshal([]byte(s), &expectedSettings); err != nil {
			return err
		}

		// Parse actual settings
		actualJSON, err := json.Marshal(service.ThanosSettings)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(actualJSON, &actualSettings); err != nil {
			return err
		}

		if !cmp.Equal(expectedSettings, actualSettings) {
			return fmt.Errorf("thanos.thanos_settings: expected %s, got %s", s, string(actualJSON))
		}
	}

	return nil
}
