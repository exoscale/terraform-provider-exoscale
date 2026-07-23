package database_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
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
	t.Parallel()

	t.Run("ResourcePg", testResourcePg)
	t.Run("ResourceMysql", testResourceMysql)
	t.Run("ResourceValkey", testResourceValkey)
	t.Run("ResourceValkeyUser", testResourceValkeyUser)
	t.Run("ResourceKafka", testResourceKafka)
	t.Run("ResourceOpensearch", testResourceOpensearch)
	t.Run("ResourceGrafana", testResourceGrafana)
	t.Run("DataSourceURI", testDataSourceURI)
	t.Run("ResourceDBAASExternalEndpointPrometheus", testResourceExternalEndpointPrometheus)
	t.Run("ResourceDBAASExternalEndpointDatadog", testResourceExternalEndpointDatadog)
	t.Run("ResourceDBAASExternalEndpointOpensearch", testResourceExternalEndpointOpensearch)
	t.Run("ResourceDBAASExternalEndpointElasticsearch", testResourceExternalEndpointElasticsearch)
	t.Run("ResourceDBAASExternalEndpointRsyslog", testResourceExternalEndpointRsyslog)
	t.Run("ResourceDBAASExternalIntegration", testResourceExternalIntegration)
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

func checkURIWellFormed(resourceFullName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceFullName]
		if !ok {
			return fmt.Errorf("resource not found: %s", resourceFullName)
		}

		uri, ok := rs.Primary.Attributes["uri"]
		if !ok || uri == "" {
			return fmt.Errorf("%s: uri attribute is not set", resourceFullName)
		}

		if !strings.Contains(uri, "://") {
			// url.Parse without scheme put the host in the URL.Scheme field and not in the URL.Host.
			uri = "//" + uri
		}

		parsed, err := url.Parse(uri)
		if err != nil {
			return fmt.Errorf("%s: uri %q does not parse as a URL: %w", resourceFullName, uri, err)
		}

		if parsed.User != nil {
			return fmt.Errorf("%s: uri %q unexpectedly still contains credentials", resourceFullName, uri)
		}

		if parsed.Hostname() == "" || parsed.Port() == "" {
			return fmt.Errorf("%s: uri %q is missing a host or port", resourceFullName, uri)
		}

		return nil
	}
}
