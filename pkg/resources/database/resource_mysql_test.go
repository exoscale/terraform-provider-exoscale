package database_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"text/template"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/egoscale/v2/oapi"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

type TemplateModelMysql struct {
	ResourceName string

	Name string
	Plan string
	Zone string

	MaintenanceDow        string
	MaintenanceTime       string
	TerminationProtection bool

	AdminPassword  string
	AdminUsername  string
	BackupSchedule string
	IpFilter       []string
	MysqlSettings  string
	Version        string
}

func testResourceMysql(t *testing.T) {
	tpl, err := template.ParseFiles("testdata/resource_mysql.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	fullResourceName := "exoscale_database.test"
	dataBase := TemplateModelMysql{
		ResourceName:          "test",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "hobbyist-2",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
		Version:               "8",
	}

	dataCreate := dataBase
	dataCreate.MaintenanceDow = "monday"
	dataCreate.MaintenanceTime = "01:23:00"
	dataCreate.BackupSchedule = "01:23"
	dataCreate.IpFilter = []string{"1.2.3.4/32"}
	dataCreate.MysqlSettings = strconv.Quote(`{"sql_mode":"ANSI,TRADITIONAL","sql_require_primary_key":true}`)
	buf := &bytes.Buffer{}
	err = tpl.Execute(buf, &dataCreate)
	if err != nil {
		t.Fatal(err)
	}
	configCreate := buf.String()

	dataUpdate := dataBase
	dataUpdate.MaintenanceDow = "tuesday"
	dataUpdate.MaintenanceTime = "02:34:00"
	dataUpdate.BackupSchedule = "23:45"
	dataUpdate.IpFilter = nil
	dataUpdate.MysqlSettings = strconv.Quote(`{"long_query_time":5,"sql_mode":"ANSI,TRADITIONAL","sql_require_primary_key":true}`)
	buf = &bytes.Buffer{}
	err = tpl.Execute(buf, &dataUpdate)
	if err != nil {
		t.Fatal(err)
	}
	configUpdate := buf.String()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             CheckDestroy("mysql", dataBase.Name),
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
					func(s *terraform.State) error {
						err := CheckExistsMysql(dataBase.Name, &dataCreate)
						if err != nil {
							return err
						}

						return nil
					},
				),
			},
			{
				// Update
				Config: configUpdate,
				Check: resource.ComposeAggregateTestCheckFunc(
					func(s *terraform.State) error {
						err := CheckExistsMysql(dataBase.Name, &dataUpdate)
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
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func CheckExistsMysql(name string, data *TemplateModelMysql) error {
	client, err := testutils.APIClient()
	if err != nil {
		return err
	}

	ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testutils.TestEnvironment(), testutils.TestZoneName))

	res, err := client.GetDbaasServiceMysqlWithResponse(ctx, oapi.DbaasServiceName(name))
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

	if !cmp.Equal(data.IpFilter, *service.IpFilter, cmpopts.EquateEmpty()) {
		return fmt.Errorf("mysql.ip_filter: expected %q, got %q", data.IpFilter, *service.IpFilter)
	}

	if v := string(service.Maintenance.Dow); data.MaintenanceDow != v {
		return fmt.Errorf("mysql.maintenance_dow: expected %q, got %q", data.MaintenanceDow, v)
	}

	if data.MaintenanceTime != service.Maintenance.Time {
		return fmt.Errorf("mysql.maintenance_time: expected %q, got %q", data.MaintenanceTime, service.Maintenance.Time)
	}

	if data.MysqlSettings != "" {
		obj := map[string]interface{}{}
		s, err := strconv.Unquote(data.MysqlSettings)
		if err != nil {
			return err
		}
		err = json.Unmarshal([]byte(s), &obj)
		if err != nil {
			return err
		}
		if !cmp.Equal(
			obj,
			*service.MysqlSettings,
		) {
			return fmt.Errorf("mysql.mysql_settings: expected %q, got %q", obj, *service.MysqlSettings)
		}
	}

	if data.Version != *service.Version {
		return fmt.Errorf("mysql.version: expected %q, got %q", data.Version, *service.Version)
	}

	return nil
}
