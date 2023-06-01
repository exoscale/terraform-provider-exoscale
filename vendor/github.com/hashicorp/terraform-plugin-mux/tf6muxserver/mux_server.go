package tf6muxserver

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/internal/logging"
)

var _ tfprotov6.ProviderServer = muxServer{}

// muxServer is a gRPC server implementation that stands in front of other
// gRPC servers, routing requests to them as if they were a single server. It
// should always be instantiated by calling NewMuxServer().
type muxServer struct {
	// Routing for data source types
	dataSources map[string]tfprotov6.ProviderServer

	// Routing for resource types
	resources map[string]tfprotov6.ProviderServer

	// Underlying servers for requests that should be handled by all servers
	servers []tfprotov6.ProviderServer

	// Server errors are cached during server creation and deferred until
	// the GetProviderSchema call. This is to prevent confusing Terraform CLI
	// errors about the plugin not starting properly, which do not display the
	// stderr output from the plugin.
	//
	// Reference: https://github.com/hashicorp/terraform-plugin-mux/issues/77
	// Reference: https://github.com/hashicorp/terraform/issues/31363
	serverDataSourceSchemaDuplicates    []string
	serverProviderSchemaDifferences     []string
	serverProviderMetaSchemaDifferences []string
	serverResourceSchemaDuplicates      []string

	// Schemas are cached during server creation
	dataSourceSchemas    map[string]*tfprotov6.Schema
	providerMetaSchema   *tfprotov6.Schema
	providerSchema       *tfprotov6.Schema
	resourceCapabilities map[string]*tfprotov6.ServerCapabilities
	resourceSchemas      map[string]*tfprotov6.Schema
}

// ProviderServer is a function compatible with tf6server.Serve.
func (s muxServer) ProviderServer() tfprotov6.ProviderServer {
	return s
}

// NewMuxServer returns a muxed server that will route gRPC requests between
// tfprotov6.ProviderServers specified. The GetProviderSchema method of each
// is called to verify that the overall muxed server is compatible by ensuring:
//
//   - All provider schemas exactly match
//   - All provider meta schemas exactly match
//   - Only one provider implements each managed resource
//   - Only one provider implements each data source
//
// The various schemas are cached and used to respond to the GetProviderSchema
// method of the muxed server.
func NewMuxServer(ctx context.Context, servers ...func() tfprotov6.ProviderServer) (muxServer, error) {
	ctx = logging.InitContext(ctx)
	result := muxServer{
		dataSources:          make(map[string]tfprotov6.ProviderServer),
		dataSourceSchemas:    make(map[string]*tfprotov6.Schema),
		resources:            make(map[string]tfprotov6.ProviderServer),
		resourceCapabilities: make(map[string]*tfprotov6.ServerCapabilities),
		resourceSchemas:      make(map[string]*tfprotov6.Schema),
	}

	for _, serverFunc := range servers {
		server := serverFunc()

		ctx = logging.Tfprotov6ProviderServerContext(ctx, server)
		logging.MuxTrace(ctx, "calling downstream server")

		resp, err := server.GetProviderSchema(ctx, &tfprotov6.GetProviderSchemaRequest{})

		if err != nil {
			return result, fmt.Errorf("error retrieving schema for %T: %w", server, err)
		}

		for _, diag := range resp.Diagnostics {
			if diag == nil {
				continue
			}
			if diag.Severity != tfprotov6.DiagnosticSeverityError {
				continue
			}
			return result, fmt.Errorf("error retrieving schema for %T:\n\n\tAttribute: %s\n\tSummary: %s\n\tDetail: %s", server, diag.Attribute, diag.Summary, diag.Detail)
		}

		if resp.Provider != nil {
			if result.providerSchema != nil && !schemaEquals(resp.Provider, result.providerSchema) {
				result.serverProviderSchemaDifferences = append(result.serverProviderSchemaDifferences, schemaDiff(resp.Provider, result.providerSchema))
			} else {
				result.providerSchema = resp.Provider
			}
		}

		if resp.ProviderMeta != nil {
			if result.providerMetaSchema != nil && !schemaEquals(resp.ProviderMeta, result.providerMetaSchema) {
				result.serverProviderMetaSchemaDifferences = append(result.serverProviderMetaSchemaDifferences, schemaDiff(resp.ProviderMeta, result.providerMetaSchema))
			} else {
				result.providerMetaSchema = resp.ProviderMeta
			}
		}

		for resourceType, schema := range resp.ResourceSchemas {
			if _, ok := result.resources[resourceType]; ok {
				result.serverResourceSchemaDuplicates = append(result.serverResourceSchemaDuplicates, resourceType)
			} else {
				result.resources[resourceType] = server
				result.resourceSchemas[resourceType] = schema
			}

			result.resourceCapabilities[resourceType] = resp.ServerCapabilities
		}

		for dataSourceType, schema := range resp.DataSourceSchemas {
			if _, ok := result.dataSources[dataSourceType]; ok {
				result.serverDataSourceSchemaDuplicates = append(result.serverDataSourceSchemaDuplicates, dataSourceType)
			} else {
				result.dataSources[dataSourceType] = server
				result.dataSourceSchemas[dataSourceType] = schema
			}
		}

		result.servers = append(result.servers, server)
	}

	return result, nil
}
