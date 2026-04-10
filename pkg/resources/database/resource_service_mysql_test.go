package database_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
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

	Integrations []TemplateModelMysqlIntegration

	// DependsOn renders an explicit `depends_on = [...]` meta-argument.
	// Each entry is emitted as-is in the HCL, so typical values look
	// like "exoscale_dbaas.primary" (bare references, no quotes).
	DependsOn []string
}

// TemplateModelMysqlIntegration renders a single `integrations` block entry
// in the mysql service template. SourceService is rendered as-is (so it may
// be either a quoted literal or a resource reference).
type TemplateModelMysqlIntegration struct {
	Type          string
	SourceService string
}

type TemplateModelMysqlUser struct {
	ResourceName string

	Username string
	Zone     string
	Service  string

	Type     string
	Password string

	Authentication string
}

type TemplateModelMysqlDb struct {
	ResourceName string

	DatabaseName string
	Service      string
	Zone         string
}

func testResourceMysql(t *testing.T) {
	t.Parallel()

	serviceTpl, err := template.ParseFiles("testdata/resource_mysql.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	userTpl, err := template.ParseFiles("testdata/resource_user_mysql.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	dbTpl, err := template.ParseFiles("testdata/resource_database_mysql.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	serviceFullResourceName := "exoscale_dbaas.test"
	serviceDataBase := TemplateModelMysql{
		ResourceName:          "test",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "hobbyist-2",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
		Version:               "8",
	}

	userFullResourceName := "exoscale_dbaas_mysql_user.test_user"
	userDataBase := TemplateModelMysqlUser{
		ResourceName: "test_user",
		Username:     "foo",
		Zone:         serviceDataBase.Zone,
		Service:      fmt.Sprintf("%s.name", serviceFullResourceName),
	}

	dbFullResourceName := "exoscale_dbaas_mysql_database.test_database"
	dbDataBase := TemplateModelMysqlDb{
		ResourceName: "test_database",
		DatabaseName: "foo_db",
		Zone:         serviceDataBase.Zone,
		Service:      fmt.Sprintf("%s.name", serviceFullResourceName),
	}
	serviceDataCreate := serviceDataBase
	serviceDataCreate.MaintenanceDow = "monday"
	serviceDataCreate.MaintenanceTime = "01:23:00"
	serviceDataCreate.BackupSchedule = "01:23"
	serviceDataCreate.IpFilter = []string{"1.2.3.4/32"}
	serviceDataCreate.MysqlSettings = strconv.Quote(`{"log_output":"INSIGHTS","long_query_time":1,"slow_query_log":true,"sql_mode":"ANSI,TRADITIONAL","sql_require_primary_key":true}`)

	userDataCreate := userDataBase
	userDataCreate.Authentication = "caching_sha2_password"

	dbDataCreate := dbDataBase

	buf := &bytes.Buffer{}
	err = serviceTpl.Execute(buf, &serviceDataCreate)
	if err != nil {
		t.Fatal(err)
	}
	err = userTpl.Execute(buf, &userDataCreate)
	if err != nil {
		t.Fatal(err)
	}
	err = dbTpl.Execute(buf, &dbDataCreate)
	if err != nil {
		t.Fatal(err)
	}
	configCreate := buf.String()

	serviceDataUpdate := serviceDataBase
	serviceDataUpdate.MaintenanceDow = "tuesday"
	serviceDataUpdate.MaintenanceTime = "02:34:00"
	serviceDataUpdate.BackupSchedule = "23:45"
	serviceDataUpdate.IpFilter = nil
	serviceDataUpdate.MysqlSettings = strconv.Quote(`{"log_output":"INSIGHTS","long_query_time":5,"slow_query_log":true,"sql_mode":"ANSI,TRADITIONAL","sql_require_primary_key":true}`)

	userDataUpdate := userDataBase
	userDataUpdate.Authentication = "mysql_native_password"

	dbDataUpdate := dbDataBase
	dbDataUpdate.DatabaseName = "bar_db"

	buf = &bytes.Buffer{}
	err = serviceTpl.Execute(buf, &serviceDataUpdate)
	if err != nil {
		t.Fatal(err)
	}
	err = userTpl.Execute(buf, &userDataUpdate)
	if err != nil {
		t.Fatal(err)
	}
	err = dbTpl.Execute(buf, &dbDataUpdate)
	if err != nil {
		t.Fatal(err)
	}
	configUpdate := buf.String()

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             CheckServiceDestroy("mysql", serviceDataBase.Name),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create
				Config: configCreate,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Service
					resource.TestCheckResourceAttrSet(serviceFullResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(serviceFullResourceName, "disk_size"),
					resource.TestCheckResourceAttrSet(serviceFullResourceName, "node_cpus"),
					resource.TestCheckResourceAttrSet(serviceFullResourceName, "node_memory"),
					resource.TestCheckResourceAttrSet(serviceFullResourceName, "nodes"),
					resource.TestCheckResourceAttrSet(serviceFullResourceName, "ca_certificate"),
					resource.TestCheckResourceAttrSet(serviceFullResourceName, "updated_at"),
					func(s *terraform.State) error {
						err := CheckExistsMysql(serviceDataBase.Name, &serviceDataCreate)
						if err != nil {
							return err
						}

						return nil
					},

					// User
					resource.TestCheckResourceAttrSet(userFullResourceName, "password"),
					resource.TestCheckResourceAttrSet(userFullResourceName, "type"),
					resource.TestCheckResourceAttrSet(userFullResourceName, "authentication"),
					func(s *terraform.State) error {
						err := CheckExistsMysqlUser(serviceDataBase.Name, userDataBase.Username, &userDataCreate)
						if err != nil {
							return err
						}

						return nil
					},

					// Database
					func(s *terraform.State) error {
						err := CheckExistsMysqlDatabase(serviceDataBase.Name, dbDataCreate.DatabaseName, &dbDataCreate)
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
					// Service
					func(s *terraform.State) error {
						err := CheckExistsMysql(serviceDataBase.Name, &serviceDataUpdate)
						if err != nil {
							return err
						}

						return nil
					},

					// User
					func(s *terraform.State) error {
						// Check the new user exists
						err = CheckExistsMysqlUser(serviceDataBase.Name, userDataUpdate.Username, &userDataUpdate)
						if err != nil {
							return err
						}

						return nil

					},

					// Database
					func(s *terraform.State) error {
						// Check the old database was deleted
						err := CheckExistsMysqlDatabase(serviceDataBase.Name, dbDataBase.DatabaseName, &dbDataUpdate)
						if err == nil {
							return fmt.Errorf("expected to not find database %s", dbDataBase.DatabaseName)
						}

						// Check the new user exists
						err = CheckExistsMysqlDatabase(serviceDataBase.Name, dbDataUpdate.DatabaseName, &dbDataUpdate)
						if err != nil {
							return err
						}
						return nil
					},
				),
			},
			{
				// Import
				ResourceName: serviceFullResourceName,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", serviceDataBase.Name, serviceDataBase.Zone), nil
					}
				}(),
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: strings.Fields("updated_at state"),
			},
			{
				ResourceName: userFullResourceName,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s/%s@%s", serviceDataBase.Name, userDataBase.Username, userDataBase.Zone), nil
					}
				}(),
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName: dbFullResourceName,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s/%s@%s", serviceDataBase.Name, dbDataUpdate.DatabaseName, dbDataBase.Zone), nil
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

	majVersion := strings.Split(*service.Version, ".")[0]

	if data.Version != majVersion {
		return fmt.Errorf("mysql.version: expected %q, got %q", data.Version, majVersion)
	}

	return nil
}

func CheckExistsMysqlUser(service, username string, data *TemplateModelMysqlUser) error {

	client, err := testutils.APIClient()
	if err != nil {
		return err
	}

	ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testutils.TestEnvironment(), testutils.TestZoneName))
	serviceUsernames := make([]string, 0)

	ch := make(chan any, 1)
	go func() {
		time.Sleep(60 * time.Second)
		ch <- "timeout!"
	}()
	for len(ch) == 0 {

		res, err := client.GetDbaasServiceMysqlWithResponse(ctx, oapi.DbaasServiceName(service))
		if err != nil {
			return err
		}
		if res.StatusCode() != http.StatusOK {
			return fmt.Errorf("API request error: unexpected status %s", res.Status())
		}
		svc := res.JSON200

		if svc.Users != nil {
			for _, u := range *svc.Users {
				if u.Username != nil {
					serviceUsernames = append(serviceUsernames, *u.Username)
					if *u.Username == username {
						if *u.Authentication == data.Authentication {
							return nil
						}
					}
				}
			}
		}

		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("could not find user %s for service %s, found %v", username, service, serviceUsernames)
}

func CheckExistsMysqlDatabase(service, databaseName string, data *TemplateModelMysqlDb) error {

	client, err := testutils.APIClient()
	if err != nil {
		return err
	}

	ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testutils.TestEnvironment(), testutils.TestZoneName))
	serviceDbs := make([]string, 0)

	ch := make(chan any, 1)
	go func() {
		time.Sleep(60 * time.Second)
		ch <- "timeout!"
	}()

	for len(ch) == 0 {
		res, err := client.GetDbaasServiceMysqlWithResponse(ctx, oapi.DbaasServiceName(service))
		if err != nil {
			return err
		}
		if res.StatusCode() != http.StatusOK {
			return fmt.Errorf("API request error: unexpected status %s", res.Status())
		}
		svc := res.JSON200

		if svc.Databases != nil {

			for _, db := range *svc.Databases {
				serviceDbs = append(serviceDbs, string(db))
				if string(db) == databaseName {
					return nil
				}
			}
		}

		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("could not find database %s for service %s, found %v", databaseName, service, serviceDbs)
}

// testResourceMysqlIntegrations exercises creating a MySQL service declared
// as a read replica of another MySQL service via the `integrations`
// attribute, and verifies that modifying the integration triggers a full
// replace of the replica resource.
func testResourceMysqlIntegrations(t *testing.T) {
	t.Parallel()

	serviceTpl, err := template.ParseFiles("testdata/resource_mysql.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	// Plan choice note: Exoscale DBaaS read replicas require the primary
	// service to be on a Business or Premium plan (Hobbyist and Startup
	// are not sufficient). The replica itself can run on any Startup or
	// larger plan. `business-4` / `startup-4` is the cheapest authorized
	// combination that actually supports the read_replica integration.
	primary := TemplateModelMysql{
		ResourceName:          "primary",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "business-4",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
		Version:               "8",
	}
	replica := TemplateModelMysql{
		ResourceName:          "replica",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "startup-4",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
		Version:               "8",
		Integrations: []TemplateModelMysqlIntegration{
			{
				Type:          "read_replica",
				SourceService: "exoscale_dbaas.primary.name",
			},
		},
	}

	renderMysqlConfig := func(services ...TemplateModelMysql) string {
		t.Helper()
		buf := &bytes.Buffer{}
		for _, s := range services {
			if err := serviceTpl.Execute(buf, &s); err != nil {
				t.Fatal(err)
			}
		}
		return buf.String()
	}

	configCreate := renderMysqlConfig(primary, replica)

	// Second variant: add a second primary and swap the replica's
	// source_service to point at it — must force replacement.
	primary2 := TemplateModelMysql{
		ResourceName:          "primary2",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "business-4",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
		Version:               "8",
	}
	replicaSwapped := replica
	// Rotate the replica name on the swap step to avoid any 409
	// eventual-consistency race on destroy+create against the same name.
	replicaSwapped.Name = acctest.RandomWithPrefix(testutils.Prefix)
	replicaSwapped.Integrations = []TemplateModelMysqlIntegration{
		{
			Type:          "read_replica",
			SourceService: "exoscale_dbaas.primary2.name",
		},
	}
	configSwap := renderMysqlConfig(primary, primary2, replicaSwapped)

	// Variant for the P1 regression step: same topology as configSwap
	// but the replica's HCL omits the `integrations` block entirely.
	// Mirrors the pg test's P1 step — see the pg test for full
	// rationale. depends_on is required because removing the
	// source_service reference also removes the implicit dependency
	// edge in Terraform's graph; without it, post-test destroy races
	// primary2 against the replica.
	replicaSwappedNoIntegrations := replicaSwapped
	replicaSwappedNoIntegrations.Integrations = nil
	replicaSwappedNoIntegrations.DependsOn = []string{"exoscale_dbaas.primary2"}
	configSwapNoIntegrations := renderMysqlConfig(primary, primary2, replicaSwappedNoIntegrations)

	primaryFullResourceName := "exoscale_dbaas.primary"
	primary2FullResourceName := "exoscale_dbaas.primary2"
	replicaFullResourceName := "exoscale_dbaas.replica"

	integrationContains := func(primaryResource string) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			primaryRes, ok := s.RootModule().Resources[primaryResource]
			if !ok {
				return fmt.Errorf("resource %s not found in state", primaryResource)
			}
			primaryName := primaryRes.Primary.Attributes["name"]
			if primaryName == "" {
				return fmt.Errorf("resource %s has no `name` attribute in state", primaryResource)
			}
			return resource.TestCheckTypeSetElemNestedAttrs(
				replicaFullResourceName,
				"mysql.integrations.*",
				map[string]string{
					"type":           "read_replica",
					"source_service": primaryName,
				},
			)(s)
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck: func() { testutils.AccPreCheck(t) },
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			CheckServiceDestroy("mysql", primary.Name),
			CheckServiceDestroy("mysql", primary2.Name),
			CheckServiceDestroy("mysql", replica.Name),
			CheckServiceDestroy("mysql", replicaSwapped.Name),
		),
		ProtoV6ProviderFactories: testutils.TestAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: configCreate,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(primaryFullResourceName, "created_at"),
					resource.TestCheckResourceAttrSet(replicaFullResourceName, "created_at"),
					resource.TestCheckResourceAttr(replicaFullResourceName, "mysql.integrations.#", "1"),
					integrationContains(primaryFullResourceName),
					func(s *terraform.State) error {
						return CheckMysqlIntegrationExists(replica.Name, primary.Name, "read_replica")
					},
				),
			},
			{
				// Verify the import round-trip of the integrations
				// attribute. See the pg test for rationale; uses the
				// shared assertImportedIntegrations helper.
				ResourceName: replicaFullResourceName,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s@%s", replica.Name, replica.Zone), nil
					}
				}(),
				ImportState: true,
				ImportStateCheck: assertImportedIntegrations(
					"mysql",
					"read_replica",
					&primary.Name,
					1,
				),
			},
			{
				// Swapping source_service must force the replica to be
				// replaced — verified with plancheck before apply.
				Config: configSwap,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(replicaFullResourceName, plancheck.ResourceActionReplace),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet(primary2FullResourceName, "created_at"),
					resource.TestCheckResourceAttr(replicaFullResourceName, "mysql.integrations.#", "1"),
					integrationContains(primary2FullResourceName),
					func(s *terraform.State) error {
						return CheckMysqlIntegrationExists(replicaSwapped.Name, primary2.Name, "read_replica")
					},
				),
			},
			{
				// P1 regression: omit `integrations` from the
				// replica's config entirely. Mirrors the equivalent
				// pg step — see testResourcePgIntegrations for full
				// rationale, including the caveat that
				// `depends_on` is only a partial mitigation for
				// destroy ordering and does not handle source
				// replacement. Verifies the Optional+Computed+
				// UseStateForUnknown fix applies symmetrically to
				// mysql: the plan action on the replica is Update
				// (not Replace), and the post-apply state still
				// carries the integration.
				//
				// ExpectNonEmptyPlan: true accounts for pre-existing
				// drift on other Optional+Computed mysql attributes
				// that lack UseStateForUnknown modifiers. The
				// PreApply plancheck asserts Update (not Replace).
				Config: configSwapNoIntegrations,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(replicaFullResourceName, plancheck.ResourceActionUpdate),
					},
				},
				ExpectNonEmptyPlan: true,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(replicaFullResourceName, "mysql.integrations.#", "1"),
					integrationContains(primary2FullResourceName),
					func(s *terraform.State) error {
						return CheckMysqlIntegrationExists(replicaSwapped.Name, primary2.Name, "read_replica")
					},
				),
			},
		},
	})
}

// CheckMysqlIntegrationExists verifies that the DBaaS API reports an
// integration of the given type between source and dest (dest being the
// service currently declared with the integration in its spec).
func CheckMysqlIntegrationExists(dest, source, integrationType string) error {
	client, err := testutils.APIClient()
	if err != nil {
		return err
	}

	// terraform-plugin-testing TestCheckFunc has no context, so we make a fresh one.
	ctx := exoapi.WithEndpoint(context.Background(), exoapi.NewReqEndpoint(testutils.TestEnvironment(), testutils.TestZoneName))

	res, err := client.GetDbaasServiceMysqlWithResponse(ctx, oapi.DbaasServiceName(dest))
	if err != nil {
		return err
	}
	if res.StatusCode() != http.StatusOK {
		return fmt.Errorf("API request error: unexpected status %s", res.Status())
	}
	service := res.JSON200

	if service.Integrations == nil {
		return fmt.Errorf("no integrations reported for service %q", dest)
	}
	for _, integration := range *service.Integrations {
		if integration.Dest == nil || integration.Source == nil || integration.Type == nil {
			continue
		}
		if *integration.Dest == dest && *integration.Source == source && *integration.Type == integrationType {
			return nil
		}
	}
	return fmt.Errorf("integration %q from %q to %q not found on service %q", integrationType, source, dest, dest)
}
