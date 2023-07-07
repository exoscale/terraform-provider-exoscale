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

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

type TemplateModelOpensearch struct {
	ResourceName string

	Name string
	Plan string
	Zone string

	MaintenanceDow        string
	MaintenanceTime       string
	TerminationProtection bool

	IpFilter                 []string
	ForkFromService          string
	RecoveryBackupName       string
	IndexPatterns            []TemplateModelOpensearchIndexPattern
	IndexTemplate            TemplateModelOpensearchIndexTemplate
	Dashboards               TemplateModelOpensearchDashboards
	KeepIndexRefreshInterval bool
	MaxIndexCount            int64
	OpensearchSettings       string
	Version                  string
}

type TemplateModelOpensearchIndexPattern struct {
	MaxIndexCount    int64
	Pattern          string
	SortingAlgorithm string
}

type TemplateModelOpensearchIndexTemplate struct {
	MappingNestedObjectsLimit int64
	NumberOfReplicas          int64
	NumberOfShards            int64
}

type TemplateModelOpensearchDashboards struct {
	Enabled         bool
	MaxOldSpaceSize int64
	RequestTimeout  int64
}

func testResourceOpensearch(t *testing.T) {
	tpl, err := template.ParseFiles("testdata/resource_opensearch.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	fullResourceName := "exoscale_database.test"
	dataBase := TemplateModelOpensearch{
		ResourceName:          "test",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "hobbyist-2",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
		Version:               "1",
	}

	dataCreate := dataBase
	dataCreate.MaintenanceDow = "monday"
	dataCreate.MaintenanceTime = "01:23:00"
	dataCreate.IndexPatterns = []TemplateModelOpensearchIndexPattern{
		{2, "log.?", "alphabetical"},
		{12, "internet.*", "creation_date"},
	}
	dataCreate.IndexTemplate = TemplateModelOpensearchIndexTemplate{5, 4, 3}
	dataCreate.Dashboards = TemplateModelOpensearchDashboards{true, 129, 30001}
	dataCreate.KeepIndexRefreshInterval = true
	dataCreate.IpFilter = []string{"0.0.0.0/0"}
	buf := &bytes.Buffer{}
	err = tpl.Execute(buf, &dataCreate)
	if err != nil {
		t.Fatal(err)
	}
	configCreate := buf.String()

	dataUpdate := dataBase
	dataUpdate.MaintenanceDow = "tuesday"
	dataUpdate.MaintenanceTime = "02:34:00"
	dataUpdate.IndexPatterns = []TemplateModelOpensearchIndexPattern{
		{4, "log.?", "alphabetical"},
		{12, "internet.*", "creation_date"},
	}
	dataUpdate.IndexTemplate = TemplateModelOpensearchIndexTemplate{5, 4, 3}
	dataUpdate.Dashboards = TemplateModelOpensearchDashboards{true, 132, 30006}
	dataUpdate.KeepIndexRefreshInterval = true
	dataUpdate.MaxIndexCount = 4
	dataUpdate.IpFilter = []string{"1.1.1.1/32"}
	dataUpdate.IpFilter = nil
	buf = &bytes.Buffer{}
	err = tpl.Execute(buf, &dataUpdate)
	if err != nil {
		t.Fatal(err)
	}
	configUpdate := buf.String()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             CheckDestroy("opensearch", dataBase.Name),
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
						err := CheckExistsOpensearch(dataBase.Name, &dataCreate)
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
						err := CheckExistsOpensearch(dataBase.Name, &dataUpdate)
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

func CheckExistsOpensearch(name string, data *TemplateModelOpensearch) error {
	client, err := testutils.APIClient()
	if err != nil {
		return err
	}

	ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testutils.TestEnvironment(), testutils.TestZoneName))

	res, err := client.GetDbaasServiceOpensearchWithResponse(ctx, oapi.DbaasServiceName(name))
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

	if *service.TerminationProtection != false {
		return fmt.Errorf("termination_protection: expected false, got true")
	}

	if !cmp.Equal(data.IpFilter, *service.IpFilter, cmpopts.EquateEmpty()) {
		return fmt.Errorf("opensearch.ip_filter: expected %q, got %q", data.IpFilter, *service.IpFilter)
	}

	if v := string(service.Maintenance.Dow); data.MaintenanceDow != v {
		return fmt.Errorf("opensearch.maintenance_dow: expected %q, got %q", data.MaintenanceDow, v)
	}

	if data.MaintenanceTime != service.Maintenance.Time {
		return fmt.Errorf("opensearch.maintenance_time: expected %q, got %q", data.MaintenanceTime, service.Maintenance.Time)
	}

	if data.OpensearchSettings != "" {
		obj := map[string]interface{}{}
		s, err := strconv.Unquote(data.OpensearchSettings)
		if err != nil {
			return err
		}
		err = json.Unmarshal([]byte(s), &obj)
		if err != nil {
			return err
		}
		if !cmp.Equal(
			obj,
			*service.OpensearchSettings,
		) {
			return fmt.Errorf("opensearch.opensearch_settings: expected %q, got %q", obj, *service.OpensearchSettings)
		}
	}

	if data.Version != *service.Version {
		return fmt.Errorf("opensearch.version: expected %q, got %q", data.Version, *service.Version)
	}

	return nil
}
