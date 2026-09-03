package database_test

import (
	"bytes"
	"fmt"
	"strconv"
	"testing"
	"text/template"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
)

type TemplateModelClickhouse struct {
	ResourceName string

	Name string
	Plan string
	Zone string

	MaintenanceDow        string
	MaintenanceTime       string
	TerminationProtection bool

	IpFilter           []string
	Version            string
	ClickhouseSettings string
}

func testResourceClickhouse(t *testing.T) {
	t.Parallel()

	tpl, err := template.ParseFiles("testdata/resource_clickhouse.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	fullResourceName := "exoscale_database.test"
	dataBase := TemplateModelClickhouse{
		ResourceName:          "test",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "standard-1",
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
	buf = &bytes.Buffer{}
	err = tpl.Execute(buf, &dataUpdate)
	if err != nil {
		t.Fatal(err)
	}
	configUpdate := buf.String()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             CheckServiceDestroy("clickhouse", dataBase.Name),
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
					resource.TestCheckResourceAttrSet(fullResourceName, "uri"),
					checkURIWellFormed(fullResourceName),
				),
			},
			{
				// Update
				Config: configUpdate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(fullResourceName, "disk_size"),
					resource.TestCheckResourceAttrSet(fullResourceName, "node_cpus"),
					resource.TestCheckResourceAttrSet(fullResourceName, "node_memory"),
					resource.TestCheckResourceAttrSet(fullResourceName, "nodes"),
					resource.TestCheckResourceAttrSet(fullResourceName, "ca_certificate"),
					resource.TestCheckResourceAttrSet(fullResourceName, "updated_at"),
					resource.TestCheckResourceAttrSet(fullResourceName, "uri"),
					checkURIWellFormed(fullResourceName),
				),
			},
			{
				// Import
				ResourceName:            fullResourceName,
				ImportState:               true,
				ImportStateVerify:         true,
				ImportStateId:             fmt.Sprintf("%s/%s", dataBase.Name, testutils.TestZoneName),
				ImportStateVerifyIgnore:   []string{"clickhouse.0.fork_from_service", "clickhouse.0.recovery_backup_name"},
			},
		},
	})
}

func TestAccClickhouseService_basic(t *testing.T) {
	testResourceClickhouse(t)
}

func TestAccClickhouseService_version(t *testing.T) {
	t.Parallel()

	tpl, err := template.ParseFiles("testdata/resource_clickhouse.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	fullResourceName := "exoscale_database.test"
	data := TemplateModelClickhouse{
		ResourceName:          "test",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "standard-1",
		Zone:                  testutils.TestZoneName,
		Version:               "24.3",
		TerminationProtection: false,
	}

	buf := &bytes.Buffer{}
	err = tpl.Execute(buf, &data)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             CheckServiceDestroy("clickhouse", data.Name),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: buf.String(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(fullResourceName, "clickhouse.0.version", "24.3"),
				),
			},
		},
	})
}

func TestAccClickhouseService_settings(t *testing.T) {
	t.Parallel()

	tpl, err := template.ParseFiles("testdata/resource_clickhouse.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	fullResourceName := "exoscale_database.test"
	data := TemplateModelClickhouse{
		ResourceName:          "test",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "standard-1",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
		ClickhouseSettings:    strconv.Quote(`{"vector_similarity_index_cache_size": 0.1}`),
	}

	buf := &bytes.Buffer{}
	err = tpl.Execute(buf, &data)
	if err != nil {
		t.Fatal(err)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             CheckServiceDestroy("clickhouse", data.Name),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: buf.String(),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(fullResourceName, "clickhouse.0.clickhouse_settings"),
				),
			},
		},
	})
}