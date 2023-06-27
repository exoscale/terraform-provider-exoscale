package database_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"text/template"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/google/go-cmp/cmp"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

type TemplateModelPg struct {
	ResourceName string

	Name string
	Plan string
	Zone string

	MaintenanceDow        string
	MaintenanceTime       string
	TerminationProtection bool

	AdminPassword     string
	AdminUsername     string
	BackupSchedule    string
	IpFilter          []string
	PgSettings        string
	PgbouncerSettings string
	PglookoutSettings string
	Version           string
}

func testResourcePg(t *testing.T) {
	tpl, err := template.ParseFiles("testdata/resource_pg.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	fullResourceName := "exoscale_database.test"
	data := TemplateModelPg{
		ResourceName:          "test",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "hobbyist-2",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
		Version:               "13",
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             CheckDestroy("pg", data.Name),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create
				Config: func() string {
					data.MaintenanceDow = "monday"
					data.MaintenanceTime = "01:23:00"
					data.BackupSchedule = "01:23"
					data.IpFilter = []string{"1.2.3.4/32"}
					data.PgSettings = `{"timezone":"Europe/Zurich"}`
					data.PgbouncerSettings = `{"min_pool_size":10}`

					buf := &bytes.Buffer{}
					err := tpl.Execute(buf, &data)
					if err != nil {
						t.Fatal(err)
					}

					return buf.String()
				}(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(fullResourceName, "disk_size"),
					resource.TestCheckResourceAttrSet(fullResourceName, "node_cpus"),
					resource.TestCheckResourceAttrSet(fullResourceName, "node_memory"),
					resource.TestCheckResourceAttrSet(fullResourceName, "nodes"),
					resource.TestCheckResourceAttrSet(fullResourceName, "ca"),
					resource.TestCheckResourceAttrSet(fullResourceName, "updated_at"),
					func(s *terraform.State) error {
						err := CheckExistsPg(data.Name, &data)
						if err != nil {
							return err
						}

						return nil
					},
				),
			},
			{
				// Update
				Config: func() string {
					data.MaintenanceDow = "tuesday"
					data.MaintenanceTime = "02:34:00"
					data.BackupSchedule = "23:45"
					data.IpFilter = nil
					data.PgSettings = `{"timezone":"Europe/Zurich","autovacuum_max_workers":5}`
					data.PgbouncerSettings = `{"min_pool_size":10,"autodb_pool_size":5}`
					data.PglookoutSettings = `{"max_failover_replication_time_lag":30}`

					buf := &bytes.Buffer{}
					err := tpl.Execute(buf, &data)
					if err != nil {
						t.Fatal(err)
					}

					return buf.String()
				}(),
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						err := CheckExistsPg(data.Name, &data)
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
						return fmt.Sprintf("%s@%s", data.Name, data.Zone), nil
					}
				}(),
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func CheckExistsPg(name string, data *TemplateModelPg) error {
	client, err := testutils.APIClient()
	if err != nil {
		return err
	}

	ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testutils.TestEnvironment(), testutils.TestZoneName))

	res, err := client.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(name))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusOK {
		return fmt.Errorf("API request error: unexpected status %s", res.Status())
	}
	service := res.JSON200

	if data.Plan != service.Plan {
		return fmt.Errorf("plan: expected %q, got %q", data.Plan, service.Plan)
	}

	if v := fmt.Sprintf("%02d:%02d", *service.BackupSchedule.BackupHour, *service.BackupSchedule.BackupMinute); data.BackupSchedule != v {
		return fmt.Errorf("backup_schedule: expected %q, got %q", data.BackupSchedule, v)
	}

	if *service.TerminationProtection != false {
		return fmt.Errorf("termination_protection: expected false, got true")
	}

	if !cmp.Equal(data.IpFilter, *service.IpFilter) {
		return fmt.Errorf("pg.ip_filter: expected %q, got %q", data.IpFilter, *service.IpFilter)
	}

	if v := string(service.Maintenance.Dow); data.MaintenanceDow != v {
		return fmt.Errorf("pg.maintenance_dow: expected %q, got %q", data.MaintenanceDow, v)
	}

	if data.MaintenanceTime != service.Maintenance.Time {
		return fmt.Errorf("pg.maintenance_time: expected %q, got %q", data.MaintenanceTime, service.Maintenance.Time)
	}

	obj := map[string]interface{}{}

	err = json.Unmarshal([]byte(data.PgSettings), &obj)
	if err != nil {
		return err
	}
	if !cmp.Equal(
		obj,
		*service.PgSettings,
	) {
		return fmt.Errorf("pg.pg_settings: expected %q, got %q", obj, *service.PgSettings)
	}

	err = json.Unmarshal([]byte(data.PgbouncerSettings), &obj)
	if err != nil {
		return err
	}
	if !cmp.Equal(
		obj,
		*service.PgbouncerSettings,
	) {
		return fmt.Errorf("pg.pg_settings: expected %q, got %q", obj, *service.PgbouncerSettings)
	}

	err = json.Unmarshal([]byte(data.PgbouncerSettings), &obj)
	if err != nil {
		return err
	}
	if !cmp.Equal(
		obj,
		*service.PgbouncerSettings,
	) {
		return fmt.Errorf("pg.pg_settings: expected %q, got %q", obj, *service.PgbouncerSettings)
	}

	if data.Version != *service.Version {
		return fmt.Errorf("pg.version: expected %q, got %q", data.Version, *service.Version)
	}

	return nil
}
