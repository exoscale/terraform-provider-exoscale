package database_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/egoscale/v2/oapi"

	"github.com/exoscale/terraform-provider-exoscale/pkg/testutils"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

func TestDatabase(t *testing.T) {
	t.Run("ResourcePg", testResourcePg)
	t.Run("ResourceMysql", testResourceMysql)
	t.Run("ResourceValkey", testResourceValkey)
	t.Run("ResourceKafka", testResourceKafka)
	t.Run("ResourceOpensearch", testResourceOpensearch)
	t.Run("ResourceGrafana", testResourceGrafana)
	t.Run("ResourceThanos", testResourceThanos)
	t.Run("DataSourceURI", testDataSourceURI)
}

func CheckServiceDestroy(dbType, name string) resource.TestCheckFunc {
	return func(_ *terraform.State) error {
		ctx := context.Background()

		client, err := testutils.APIClient()
		if err != nil {
			return err
		}

		ctx = api.WithEndpoint(ctx, api.NewReqEndpoint(testutils.TestEnvironment(), testutils.TestZoneName))

		ctxV3 := ctx

		defaultClientV3, err := testutils.APIClientV3()
		if err != nil {
			return err
		}

		clientV3, err := utils.SwitchClientZone(
			ctxV3,
			defaultClientV3,
			testutils.TestZoneName,
		)

		if err != nil {
			return err
		}

		var serviceErr error
		switch dbType {
		case "grafana":
			_, serviceErr = client.GetDbaasServiceGrafanaWithResponse(ctx, oapi.DbaasServiceName(name))
		case "kafka":
			_, serviceErr = client.GetDbaasServiceKafkaWithResponse(ctx, oapi.DbaasServiceName(name))
		case "mysql":
			_, serviceErr = client.GetDbaasServiceMysqlWithResponse(ctx, oapi.DbaasServiceName(name))
		case "pg":
			_, serviceErr = client.GetDbaasServicePgWithResponse(ctx, oapi.DbaasServiceName(name))
		case "valkey":
			_, serviceErr = clientV3.GetDBAASServiceValkey(ctxV3, name)
		case "opensearch":
			_, serviceErr = client.GetDbaasServiceOpensearchWithResponse(ctx, oapi.DbaasServiceName(name))
		case "thanos":
			_, serviceErr = clientV3.GetDBAASServiceThanos(ctxV3, name)
		default:
			return fmt.Errorf("unsupported database service type %q", dbType)
		}

		if serviceErr != nil {
			// For V2 API
			if errors.Is(serviceErr, api.ErrNotFound) {
				return nil
			}
			// For V3 API
			if strings.Contains(serviceErr.Error(), "Not Found: Service does not exist") {
				return nil
			}
			return serviceErr
		}

		return fmt.Errorf("database service %q not deleted", name)
	}
}
