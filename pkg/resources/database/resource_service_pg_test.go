package database_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/egoscale/v2/oapi"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

type TemplateModelPg struct {
	ResourceName string

	Name string
	Plan string
	Zone string

	MaintenanceDow        string
	MaintenanceTime       string
	TerminationProtection bool

	AdminPassword           string
	AdminUsername           string
	BackupSchedule          string
	IpFilter                []string
	PgSettings              string
	PgbouncerSettings       string
	PglookoutSettings       string
	Version                 string
	SharedBuffersPercentage int64
	TimescaledbSettings     string
	Variant                 string
	WorkMem                 int64
	RecoveryBackupTime      string
}

type TemplateModelPgUser struct {
	ResourceName string

	Username string
	Service  string
	Zone     string
}

type TemplateModelPgDb struct {
	ResourceName string

	DatabaseName string
	Service      string
	Zone         string
}

type TemplateModelPgConnectionPool struct {
	ResourceName string

	Name         string
	DatabaseName string
	Username     string
	Mode         string
	Size         int64
	Service      string
	Zone         string
}

func testResourcePg(t *testing.T) {
	t.Parallel()

	serviceTpl, err := template.ParseFiles("testdata/resource_pg.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	userTpl, err := template.ParseFiles("testdata/resource_user_pg.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	dbTpl, err := template.ParseFiles("testdata/resource_database_pg.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	poolTpl, err := template.ParseFiles("testdata/resource_connection_pool_pg.tmpl")
	if err != nil {
		t.Fatal(err)
	}

	serviceFullResourceName := "exoscale_dbaas.test"
	serviceDataBase := TemplateModelPg{
		ResourceName:          "test",
		Name:                  acctest.RandomWithPrefix(testutils.Prefix),
		Plan:                  "hobbyist-2",
		Zone:                  testutils.TestZoneName,
		TerminationProtection: false,
		Version:               "15",
	}

	userFullResourceName := "exoscale_dbaas_pg_user.test_user"
	userDataBase := TemplateModelPgUser{
		ResourceName: "test_user",
		Username:     "foo",
		Zone:         serviceDataBase.Zone,
		Service:      fmt.Sprintf("%s.name", serviceFullResourceName),
	}

	dbFullResourceName := "exoscale_dbaas_pg_database.test_database"
	dbDataBase := TemplateModelPgDb{
		ResourceName: "test_database",
		DatabaseName: "foo_db",
		Zone:         serviceDataBase.Zone,
		Service:      fmt.Sprintf("%s.name", serviceFullResourceName),
	}

	// Keep pool sizing conservative: this test replaces one pool while a second
	// defaulted pool exists, and the hobbyist-2 service plan has a tight
	// connection budget.
	poolFullResourceName := "exoscale_dbaas_pg_connection_pool.test_pool"
	poolDataBase := TemplateModelPgConnectionPool{
		ResourceName: "test_pool",
		Name:         "foo-pool",
		DatabaseName: fmt.Sprintf("%s.database_name", dbFullResourceName),
		Username:     fmt.Sprintf("%s.username", userFullResourceName),
		Mode:         "session",
		Size:         3,
		Zone:         serviceDataBase.Zone,
		Service:      fmt.Sprintf("%s.name", serviceFullResourceName),
	}

	defaultPoolFullResourceName := "exoscale_dbaas_pg_connection_pool.default_pool"
	defaultPoolDataBase := TemplateModelPgConnectionPool{
		ResourceName: "default_pool",
		Name:         "foo-default-pool",
		DatabaseName: fmt.Sprintf("%s.database_name", dbFullResourceName),
		Zone:         serviceDataBase.Zone,
		Service:      fmt.Sprintf("%s.name", serviceFullResourceName),
	}

	serviceDataCreate := serviceDataBase
	serviceDataCreate.MaintenanceDow = "monday"
	serviceDataCreate.MaintenanceTime = "01:23:00"
	serviceDataCreate.BackupSchedule = "01:23"
	serviceDataCreate.IpFilter = []string{"1.2.3.4/32"}
	serviceDataCreate.PgSettings = strconv.Quote(`{"timezone":"Europe/Zurich"}`)
	serviceDataCreate.PgbouncerSettings = strconv.Quote(`{"min_pool_size":2}`)
	serviceDataCreate.SharedBuffersPercentage = 25
	serviceDataCreate.WorkMem = 4
	serviceDataCreate.Variant = "aiven"

	testRecoveryBackupTime := os.Getenv("EXOSCALE_TEST_PG_RECOVERY_BACKUP_TIME")
	if testRecoveryBackupTime != "" {
		serviceDataCreate.RecoveryBackupTime = testRecoveryBackupTime
	}

	userDataCreate := userDataBase
	dbDataCreate := dbDataBase
	poolDataCreate := poolDataBase
	defaultPoolDataCreate := defaultPoolDataBase
	poolDataCreateExpected := TemplateModelPgConnectionPool{
		Name:         poolDataBase.Name,
		DatabaseName: dbDataCreate.DatabaseName,
		Username:     userDataCreate.Username,
		Mode:         poolDataCreate.Mode,
		Size:         poolDataCreate.Size,
	}

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
	err = poolTpl.Execute(buf, &poolDataCreate)
	if err != nil {
		t.Fatal(err)
	}
	err = poolTpl.Execute(buf, &defaultPoolDataCreate)
	if err != nil {
		t.Fatal(err)
	}
	configCreate := buf.String()

	serviceDataUpdate := serviceDataBase
	serviceDataUpdate.MaintenanceDow = "tuesday"
	serviceDataUpdate.MaintenanceTime = "02:34:00"
	serviceDataUpdate.BackupSchedule = "23:45"
	serviceDataUpdate.IpFilter = nil
	serviceDataUpdate.PgSettings = strconv.Quote(`{"max_worker_processes":10,"timezone":"Europe/Zurich"}`)
	serviceDataUpdate.PgbouncerSettings = strconv.Quote(`{"autodb_pool_size":3,"min_pool_size":2}`)
	serviceDataUpdate.PglookoutSettings = strconv.Quote(`{"max_failover_replication_time_lag":30}`)
	serviceDataUpdate.SharedBuffersPercentage = 30
	serviceDataUpdate.WorkMem = 8
	serviceDataUpdate.Variant = "aiven"

	userDataUpdate := userDataBase
	userDataUpdate.Username = "bar"

	dbDataUpdate := dbDataBase
	dbDataUpdate.DatabaseName = "bar_db"

	poolDataUpdate := poolDataBase
	defaultPoolDataUpdate := defaultPoolDataBase
	poolDataUpdate.DatabaseName = fmt.Sprintf("%s.database_name", dbFullResourceName)
	poolDataUpdate.Username = fmt.Sprintf("%s.username", userFullResourceName)
	poolDataUpdate.Mode = "transaction"
	poolDataUpdate.Size = 4
	poolDataUpdateExpected := TemplateModelPgConnectionPool{
		Name:         poolDataBase.Name,
		DatabaseName: dbDataUpdate.DatabaseName,
		Username:     userDataUpdate.Username,
		Mode:         poolDataUpdate.Mode,
		Size:         poolDataUpdate.Size,
	}

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
	err = poolTpl.Execute(buf, &poolDataUpdate)
	if err != nil {
		t.Fatal(err)
	}
	err = poolTpl.Execute(buf, &defaultPoolDataUpdate)
	if err != nil {
		t.Fatal(err)
	}
	configUpdate := buf.String()

	serviceDataScale := serviceDataUpdate
	serviceDataScale.Plan = "startup-4"

	buf = &bytes.Buffer{}
	err = serviceTpl.Execute(buf, &serviceDataScale)
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
	err = poolTpl.Execute(buf, &poolDataUpdate)
	if err != nil {
		t.Fatal(err)
	}
	err = poolTpl.Execute(buf, &defaultPoolDataUpdate)
	if err != nil {
		t.Fatal(err)
	}
	configScale := buf.String()

	namedCheck := func(name string, f resource.TestCheckFunc) resource.TestCheckFunc {
		return func(s *terraform.State) error {
			ok := t.Run(name, func(t *testing.T) {
				if err := f(s); err != nil {
					t.Fatal(err)
				}
			})
			if !ok {
				return fmt.Errorf("check %q failed", name)
			}
			return nil
		}
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testutils.AccPreCheck(t) },
		CheckDestroy:             CheckServiceDestroy("pg", serviceDataBase.Name),
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
					namedCheck("create/shared_buffers_percentage", resource.TestCheckResourceAttr(serviceFullResourceName, "pg.shared_buffers_percentage", "25")),
					namedCheck("create/work_mem", resource.TestCheckResourceAttr(serviceFullResourceName, "pg.work_mem", "4")),
					namedCheck("create/variant", resource.TestCheckResourceAttr(serviceFullResourceName, "pg.variant", "aiven")),
					func(s *terraform.State) error {
						if testRecoveryBackupTime != "" {
							return namedCheck("create/recovery_backup_time", resource.TestCheckResourceAttr(serviceFullResourceName, "pg.recovery_backup_time", testRecoveryBackupTime))(s)
						}
						return nil
					},
					func(s *terraform.State) error {
						err := CheckExistsPg(serviceDataBase.Name, &serviceDataCreate)
						if err != nil {
							return err
						}

						return nil
					},
					// User
					resource.TestCheckResourceAttrSet(userFullResourceName, "password"),
					resource.TestCheckResourceAttrSet(userFullResourceName, "type"),
					func(s *terraform.State) error {
						err := CheckExistsPgUser(serviceDataBase.Name, userDataBase.Username, &userDataCreate)
						if err != nil {
							return err
						}

						return nil
					},

					// Database
					func(s *terraform.State) error {
						err := CheckExistsPgDatabase(serviceDataBase.Name, dbDataCreate.DatabaseName, &dbDataCreate)
						if err != nil {
							return err
						}

						return nil
					},
					// Connection pool
					resource.TestCheckResourceAttrSet(poolFullResourceName, "connection_uri"),
					func(s *terraform.State) error {
						return CheckExistsPgConnectionPool(serviceDataBase.Name, poolDataCreateExpected.Name, &poolDataCreateExpected)
					},
					resource.TestCheckResourceAttrSet(defaultPoolFullResourceName, "connection_uri"),
					resource.TestCheckResourceAttrSet(defaultPoolFullResourceName, "mode"),
					resource.TestCheckResourceAttrSet(defaultPoolFullResourceName, "size"),
					resource.TestCheckResourceAttr(defaultPoolFullResourceName, "username", ""),
					func(s *terraform.State) error {
						return CheckExistsPgConnectionPoolAny(serviceDataBase.Name, defaultPoolDataCreate.Name)
					},
				),
			},
			{
				// Update
				Config: configUpdate,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Service
					namedCheck("update/shared_buffers_percentage", resource.TestCheckResourceAttr(serviceFullResourceName, "pg.shared_buffers_percentage", "30")),
					namedCheck("update/work_mem", resource.TestCheckResourceAttr(serviceFullResourceName, "pg.work_mem", "8")),
					namedCheck("update/variant", resource.TestCheckResourceAttr(serviceFullResourceName, "pg.variant", "aiven")),
					func(s *terraform.State) error {
						err := CheckExistsPg(serviceDataBase.Name, &serviceDataUpdate)
						if err != nil {
							return err
						}

						return nil
					},

					// Connection pool
					resource.TestCheckResourceAttrSet(poolFullResourceName, "connection_uri"),
					func(s *terraform.State) error {
						return CheckExistsPgConnectionPool(serviceDataBase.Name, poolDataUpdateExpected.Name, &poolDataUpdateExpected)
					},
					resource.TestCheckResourceAttrSet(defaultPoolFullResourceName, "connection_uri"),
					resource.TestCheckResourceAttrSet(defaultPoolFullResourceName, "mode"),
					resource.TestCheckResourceAttrSet(defaultPoolFullResourceName, "size"),
					resource.TestCheckResourceAttr(defaultPoolFullResourceName, "username", ""),
					func(s *terraform.State) error {
						return CheckExistsPgConnectionPoolAny(serviceDataBase.Name, defaultPoolDataUpdate.Name)
					},

					// User
					func(s *terraform.State) error {
						// Check the old user was deleted after the connection pool replacement.
						err := CheckNotExistsPgUser(serviceDataBase.Name, userDataBase.Username)
						if err != nil {
							return err
						}

						// Check the new user exists
						err = CheckExistsPgUser(serviceDataBase.Name, userDataUpdate.Username, &userDataUpdate)
						if err != nil {
							return err
						}

						return nil
					},

					// Database
					func(s *terraform.State) error {
						// Check the old database was deleted after the connection pool replacement.
						err := CheckNotExistsPgDatabase(serviceDataBase.Name, dbDataBase.DatabaseName)
						if err != nil {
							return err
						}

						// Check the new user exists
						err = CheckExistsPgDatabase(serviceDataBase.Name, dbDataUpdate.DatabaseName, &dbDataUpdate)
						if err != nil {
							return err
						}
						return nil
					},
				),
			},
			{
				// Scale
				Config: configScale,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Service
					func(s *terraform.State) error {
						err := CheckExistsPg(serviceDataBase.Name, &serviceDataScale)
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
				ImportState: true,
				// NOTE: ImportStateVerify doesn't work when there are optional attributes.
				//ImportStateVerify: true
			},
			{
				ResourceName: userFullResourceName,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s/%s@%s", serviceDataBase.Name, userDataUpdate.Username, userDataBase.Zone), nil
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
			{
				ResourceName: defaultPoolFullResourceName,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s/%s@%s", serviceDataBase.Name, defaultPoolDataUpdate.Name, defaultPoolDataBase.Zone), nil
					}
				}(),
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName: poolFullResourceName,
				ImportStateIdFunc: func() resource.ImportStateIdFunc {
					return func(*terraform.State) (string, error) {
						return fmt.Sprintf("%s/%s@%s", serviceDataBase.Name, poolDataUpdateExpected.Name, poolDataBase.Zone), nil
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

	if !cmp.Equal(data.IpFilter, *service.IpFilter, cmpopts.EquateEmpty()) {
		return fmt.Errorf("pg.ip_filter: expected %q, got %q", data.IpFilter, *service.IpFilter)
	}

	if v := string(service.Maintenance.Dow); data.MaintenanceDow != v {
		return fmt.Errorf("pg.maintenance_dow: expected %q, got %q", data.MaintenanceDow, v)
	}

	if data.MaintenanceTime != service.Maintenance.Time {
		return fmt.Errorf("pg.maintenance_time: expected %q, got %q", data.MaintenanceTime, service.Maintenance.Time)
	}

	serviceMajVersion := strings.Split(*service.Version, ".")[0]

	if data.Version != serviceMajVersion {
		return fmt.Errorf("pg.version: expected %q, got %q", data.Version, serviceMajVersion)
	}

	//  NOTE: Due to default values setup by Aiven, we won't validate settings.

	if data.SharedBuffersPercentage != 0 {
		if service.SharedBuffersPercentage == nil {
			return fmt.Errorf("pg.shared_buffers_percentage: expected %d, got nil", data.SharedBuffersPercentage)
		}
		if data.SharedBuffersPercentage != *service.SharedBuffersPercentage {
			return fmt.Errorf("pg.shared_buffers_percentage: expected %d, got %d", data.SharedBuffersPercentage, *service.SharedBuffersPercentage)
		}
	}

	if data.WorkMem != 0 {
		if service.WorkMem == nil {
			return fmt.Errorf("pg.work_mem: expected %d, got nil", data.WorkMem)
		}
		if data.WorkMem != *service.WorkMem {
			return fmt.Errorf("pg.work_mem: expected %d, got %d", data.WorkMem, *service.WorkMem)
		}
	}

	return nil
}

func CheckExistsPgUser(service, username string, data *TemplateModelPgUser) error {

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
		res, err := client.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(service))
		if err != nil {
			return err
		}
		if res.StatusCode() != http.StatusOK {
			return fmt.Errorf("API request error: unexpected status %s", res.Status())
		}
		svc := res.JSON200

		if svc.Users != nil {
			for _, u := range *svc.Users {
				serviceUsernames = append(serviceUsernames, u.Username)
				if u.Username == username {
					return nil
				}
			}
		}
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("could not find user %s for service %s, found %v", username, service, serviceUsernames)
}

func CheckNotExistsPgUser(service, username string) error {
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
		res, err := client.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(service))
		if err != nil {
			return err
		}
		if res.StatusCode() != http.StatusOK {
			return fmt.Errorf("API request error: unexpected status %s", res.Status())
		}
		svc := res.JSON200

		serviceUsernames = serviceUsernames[:0]
		found := false
		if svc.Users != nil {
			for _, u := range *svc.Users {
				serviceUsernames = append(serviceUsernames, u.Username)
				if u.Username == username {
					found = true
				}
			}
		}

		if !found {
			return nil
		}

		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("could still find user %s for service %s, found %v", username, service, serviceUsernames)
}

func CheckExistsPgDatabase(service, databaseName string, data *TemplateModelPgDb) error {

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

		res, err := client.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(service))
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

func CheckNotExistsPgDatabase(service, databaseName string) error {
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
		res, err := client.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(service))
		if err != nil {
			return err
		}
		if res.StatusCode() != http.StatusOK {
			return fmt.Errorf("API request error: unexpected status %s", res.Status())
		}
		svc := res.JSON200

		serviceDbs = serviceDbs[:0]
		found := false
		if svc.Databases != nil {
			for _, db := range *svc.Databases {
				serviceDbs = append(serviceDbs, string(db))
				if string(db) == databaseName {
					found = true
				}
			}
		}

		if !found {
			return nil
		}

		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("could still find database %s for service %s, found %v", databaseName, service, serviceDbs)
}

func CheckExistsPgConnectionPool(service, poolName string, data *TemplateModelPgConnectionPool) error {
	defaultClientV3, err := testutils.APIClientV3()
	if err != nil {
		return err
	}

	client, err := utils.SwitchClientZone(context.Background(), defaultClientV3, testutils.TestZoneName)
	if err != nil {
		return err
	}

	servicePools := make([]string, 0)

	ch := make(chan any, 1)
	go func() {
		time.Sleep(60 * time.Second)
		ch <- "timeout!"
	}()
	for len(ch) == 0 {
		svc, err := client.GetDBAASServicePG(context.Background(), service)
		if err != nil {
			return err
		}

		for _, pool := range svc.ConnectionPools {
			servicePools = append(servicePools, string(pool.Name))
			if string(pool.Name) != poolName {
				continue
			}

			if string(pool.Database) != data.DatabaseName {
				return fmt.Errorf("pool database_name: expected %q, got %q", data.DatabaseName, string(pool.Database))
			}
			if string(pool.Username) != data.Username {
				return fmt.Errorf("pool username: expected %q, got %q", data.Username, string(pool.Username))
			}
			if string(pool.Mode) != data.Mode {
				return fmt.Errorf("pool mode: expected %q, got %q", data.Mode, string(pool.Mode))
			}
			if int64(pool.Size) != data.Size {
				return fmt.Errorf("pool size: expected %d, got %d", data.Size, int64(pool.Size))
			}
			if pool.ConnectionURI == "" {
				return fmt.Errorf("pool connection_uri is empty")
			}

			return nil
		}

		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("could not find connection pool %s for service %s, found %v", poolName, service, servicePools)
}

func CheckExistsPgConnectionPoolAny(service, poolName string) error {
	defaultClientV3, err := testutils.APIClientV3()
	if err != nil {
		return err
	}

	client, err := utils.SwitchClientZone(context.Background(), defaultClientV3, testutils.TestZoneName)
	if err != nil {
		return err
	}

	servicePools := make([]string, 0)

	ch := make(chan any, 1)
	go func() {
		time.Sleep(60 * time.Second)
		ch <- "timeout!"
	}()
	for len(ch) == 0 {
		svc, err := client.GetDBAASServicePG(context.Background(), service)
		if err != nil {
			return err
		}

		servicePools = servicePools[:0]
		for _, pool := range svc.ConnectionPools {
			servicePools = append(servicePools, string(pool.Name))
			if string(pool.Name) != poolName {
				continue
			}

			if pool.ConnectionURI == "" {
				return fmt.Errorf("pool connection_uri is empty")
			}

			return nil
		}

		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("could not find connection pool %s for service %s, found %v", poolName, service, servicePools)
}
