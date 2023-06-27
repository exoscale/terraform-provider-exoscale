package exoscale

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/egoscale/v2/oapi"
)

// Note: at the moment we can't test updating the `plan` attribute, as the Aiven API
// prevents us from upgrading the plan of a service that hasn't been backed up at least once.
// (which is the case of a service that has just been created by the TF provider).

var (
	testAccResourceDatabaseMaintenanceDOW         = "monday"
	testAccResourceDatabaseMaintenanceDOWUpdated  = "tuesday"
	testAccResourceDatabaseMaintenanceTime        = "01:23:00"
	testAccResourceDatabaseMaintenanceTimeUpdated = "02:34:00"
	testAccResourceDatabaseName                   = acctest.RandomWithPrefix(testPrefix)
)

func testAccCheckResourceDatabaseExists(r string, d *interface{}) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[r]
		if !ok {
			return errors.New("resource not found in the state")
		}

		if rs.Primary.ID == "" {
			return errors.New("resource ID not set")
		}

		client := GetComputeClient(testAccProvider.Meta())

		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testZoneName),
		)

		switch rs.Primary.Attributes[resDatabaseAttrType] {
		case "kafka":
			res, err := client.GetDbaasServiceKafkaWithResponse(ctx, oapi.DbaasServiceName(rs.Primary.ID))
			if err != nil {
				return err
			}
			if res.StatusCode() != http.StatusOK {
				return fmt.Errorf("API request error: unexpected status %s", res.Status())
			}
			*d = res.JSON200
			return nil

		case "mysql":
			res, err := client.GetDbaasServiceMysqlWithResponse(ctx, oapi.DbaasServiceName(rs.Primary.ID))
			if err != nil {
				return err
			}
			if res.StatusCode() != http.StatusOK {
				return fmt.Errorf("API request error: unexpected status %s", res.Status())
			}
			*d = res.JSON200
			return nil

		case "pg":
			res, err := client.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(rs.Primary.ID))
			if err != nil {
				return err
			}
			if res.StatusCode() != http.StatusOK {
				return fmt.Errorf("API request error: unexpected status %s", res.Status())
			}
			*d = res.JSON200
			return nil

		case "redis":
			res, err := client.GetDbaasServiceRedisWithResponse(ctx, oapi.DbaasServiceName(rs.Primary.ID))
			if err != nil {
				return err
			}
			if res.StatusCode() != http.StatusOK {
				return fmt.Errorf("API request error: unexpected status %s", res.Status())
			}
			*d = res.JSON200
			return nil

		case "opensearch":
			res, err := client.GetDbaasServiceOpensearchWithResponse(ctx, oapi.DbaasServiceName(rs.Primary.ID))
			if err != nil {
				return err
			}
			if res.StatusCode() != http.StatusOK {
				return fmt.Errorf("API request error: unexpected status %s", res.Status())
			}
			*d = res.JSON200
			return nil

		default:
			return fmt.Errorf(
				"unsupported database service type %q",
				rs.Primary.Attributes[resDatabaseAttrType],
			)
		}
	}
}

func testAccCheckResourceDatabaseDestroy(dbType, dbName string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := GetComputeClient(testAccProvider.Meta())
		ctx := exoapi.WithEndpoint(
			context.Background(),
			exoapi.NewReqEndpoint(testEnvironment, testZoneName),
		)

		var err error
		switch dbType {
		case "kafka":
			_, err = client.GetDbaasServiceKafkaWithResponse(ctx, oapi.DbaasServiceName(dbName))
		case "mysql":
			_, err = client.GetDbaasServiceMysqlWithResponse(ctx, oapi.DbaasServiceName(dbName))
		case "pg":
			_, err = client.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(dbName))
		case "redis":
			_, err = client.GetDbaasServiceRedisWithResponse(ctx, oapi.DbaasServiceName(dbName))
		case "opensearch":
			_, err = client.GetDbaasServiceOpensearchWithResponse(ctx, oapi.DbaasServiceName(dbName))
		default:
			return fmt.Errorf("unsupported database service type %q", dbType)
		}

		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				return nil
			}
			return err
		}

		return errors.New("database service still exists")
	}
}
