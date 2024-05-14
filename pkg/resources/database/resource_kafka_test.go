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

type TemplateModelKafka struct {
	ResourceName string

	Name string
	Plan string
	Zone string

	MaintenanceDow        string
	MaintenanceTime       string
	TerminationProtection bool

	EnableCertAuth         bool
	EnableKafkaConnect     bool
	EnableKafkaREST        bool
	EnableSASLAuth         bool
	EnableSchemaRegistry   bool
	IpFilter               []string
	KafkaSettings          string
	ConnectSettings        string
	RestSettings           string
	SchemaRegistrySettings string
	Version                string
}

func testResourceKafka(t *testing.T) {
	tpl, err := template.ParseFiles("testdata/resource_kafka.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	fullResourceName := "exoscale_database.test"
	dataBase := TemplateModelKafka{
		ResourceName:          "test",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "business-4",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
		Version:               "3.6",
	}

	dataCreate := dataBase
	dataCreate.MaintenanceDow = "monday"
	dataCreate.MaintenanceTime = "01:23:00"
	dataCreate.EnableCertAuth = true
	dataCreate.IpFilter = []string{"1.2.3.4/32"}
	dataCreate.KafkaSettings = strconv.Quote(`{"num_partitions":10}`)
	buf := &bytes.Buffer{}
	err = tpl.Execute(buf, &dataCreate)
	if err != nil {
		t.Fatal(err)
	}
	configCreate := buf.String()

	dataUpdate := dataBase
	dataUpdate.MaintenanceDow = "tuesday"
	dataUpdate.MaintenanceTime = "02:34:00"
	dataUpdate.EnableCertAuth = false
	dataUpdate.EnableSASLAuth = true
	dataUpdate.EnableKafkaREST = true
	dataUpdate.EnableKafkaConnect = true
	dataUpdate.IpFilter = nil
	dataUpdate.KafkaSettings = strconv.Quote(`{"compression_type":"gzip","num_partitions":10}`)
	dataUpdate.RestSettings = strconv.Quote(`{"consumer_request_max_bytes":100000}`)
	dataUpdate.ConnectSettings = strconv.Quote(`{"session_timeout_ms":6000}`)
	buf = &bytes.Buffer{}
	err = tpl.Execute(buf, &dataUpdate)
	if err != nil {
		t.Fatal(err)
	}
	configUpdate := buf.String()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             CheckDestroy("kafka", dataBase.Name),
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
						err := CheckExistsKafka(dataBase.Name, &dataCreate)
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
						err := CheckExistsKafka(dataBase.Name, &dataUpdate)
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

func CheckExistsKafka(name string, data *TemplateModelKafka) error {
	client, err := testutils.APIClient()
	if err != nil {
		return err
	}

	ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testutils.TestEnvironment(), testutils.TestZoneName))

	res, err := client.GetDbaasServiceKafkaWithResponse(ctx, oapi.DbaasServiceName(name))
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
		return fmt.Errorf("kafka.ip_filter: expected %q, got %q", data.IpFilter, *service.IpFilter)
	}

	if v := string(service.Maintenance.Dow); data.MaintenanceDow != v {
		return fmt.Errorf("kafka.maintenance_dow: expected %q, got %q", data.MaintenanceDow, v)
	}

	if data.MaintenanceTime != service.Maintenance.Time {
		return fmt.Errorf("kafka.maintenance_time: expected %q, got %q", data.MaintenanceTime, service.Maintenance.Time)
	}

	if data.EnableKafkaConnect != *service.KafkaConnectEnabled {
		return fmt.Errorf("kafka.enable_kafka_connect: expected %v, got %v", data.EnableKafkaConnect, *service.KafkaConnectEnabled)
	}

	if data.EnableKafkaREST != *service.KafkaRestEnabled {
		return fmt.Errorf("kafka.enable_kafka_rest: expected %v, got %v", data.EnableKafkaREST, *service.KafkaRestEnabled)
	}

	if data.EnableSchemaRegistry != *service.SchemaRegistryEnabled {
		return fmt.Errorf("kafka.enable_schema_registry: expected %v, got %v", data.EnableSchemaRegistry, *service.SchemaRegistryEnabled)
	}

	if data.EnableCertAuth != *service.AuthenticationMethods.Certificate {
		return fmt.Errorf("kafka.enable_cert_auth: expected %v, got %v", data.EnableCertAuth, *service.AuthenticationMethods.Certificate)
	}

	if data.EnableSASLAuth != *service.AuthenticationMethods.Sasl {
		return fmt.Errorf("kafka.enable_sasl_auth: expected %v, got %v", data.EnableSASLAuth, *service.AuthenticationMethods.Sasl)
	}

	if data.KafkaSettings != "" {
		obj := map[string]interface{}{}
		s, err := strconv.Unquote(data.KafkaSettings)
		if err != nil {
			return err
		}
		err = json.Unmarshal([]byte(s), &obj)
		if err != nil {
			return err
		}
		if !cmp.Equal(
			obj,
			*service.KafkaSettings,
		) {
			return fmt.Errorf("kafka.kafka_settings: expected %q, got %q", obj, *service.KafkaSettings)
		}
	}

	if data.ConnectSettings != "" {
		obj := map[string]interface{}{}
		s, err := strconv.Unquote(data.ConnectSettings)
		if err != nil {
			return err
		}
		err = json.Unmarshal([]byte(s), &obj)
		if err != nil {
			return err
		}
		if !cmp.Equal(
			obj,
			*service.KafkaConnectSettings,
		) {
			return fmt.Errorf("kafka.kafka_connect_settings: expected %q, got %q", obj, *service.KafkaConnectSettings)
		}
	}

	if data.RestSettings != "" {
		obj := map[string]interface{}{}
		s, err := strconv.Unquote(data.RestSettings)
		if err != nil {
			return err
		}
		err = json.Unmarshal([]byte(s), &obj)
		if err != nil {
			return err
		}
		if !cmp.Equal(
			obj,
			*service.KafkaRestSettings,
		) {
			return fmt.Errorf("kafka.kafka_rest_settings: expected %q, got %q", obj, *service.KafkaRestSettings)
		}
	}

	if data.SchemaRegistrySettings != "" {
		obj := map[string]interface{}{}
		s, err := strconv.Unquote(data.SchemaRegistrySettings)
		if err != nil {
			return err
		}
		err = json.Unmarshal([]byte(s), &obj)
		if err != nil {
			return err
		}
		if !cmp.Equal(
			obj,
			*service.SchemaRegistrySettings,
		) {
			return fmt.Errorf("kafka.schema_registry_settings: expected %q, got %q", obj, *service.SchemaRegistrySettings)
		}
	}

	if data.Version != *service.Version {
		return fmt.Errorf("kafka.version: expected %q, got %q", data.Version, *service.Version)
	}

	return nil
}
