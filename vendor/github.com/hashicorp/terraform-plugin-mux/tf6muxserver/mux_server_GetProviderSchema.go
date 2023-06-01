package tf6muxserver

import (
	"context"

	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-mux/internal/logging"
)

// GetProviderSchema merges the schemas returned by the
// tfprotov6.ProviderServers associated with muxServer into a single schema.
// Resources and data sources must be returned from only one server. Provider
// and ProviderMeta schemas must be identical between all servers.
func (s muxServer) GetProviderSchema(ctx context.Context, req *tfprotov6.GetProviderSchemaRequest) (*tfprotov6.GetProviderSchemaResponse, error) {
	rpc := "GetProviderSchema"
	ctx = logging.InitContext(ctx)
	ctx = logging.RpcContext(ctx, rpc)
	logging.MuxTrace(ctx, "serving cached schema information")

	resp := &tfprotov6.GetProviderSchemaResponse{
		Provider:          s.providerSchema,
		ResourceSchemas:   s.resourceSchemas,
		DataSourceSchemas: s.dataSourceSchemas,
		ProviderMeta:      s.providerMetaSchema,

		// Always announce all ServerCapabilities. Individual capabilities are
		// handled in their respective RPCs to protect downstream servers if
		// they are not compatible with a capability.
		ServerCapabilities: &tfprotov6.ServerCapabilities{
			PlanDestroy: true,
		},
	}

	for _, diff := range s.serverProviderSchemaDifferences {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov6.Diagnostic{
			Severity: tfprotov6.DiagnosticSeverityError,
			Summary:  "Invalid Provider Server Combination",
			Detail: "The combined provider has differing provider schema implementations across providers. " +
				"Provider schemas must be identical across providers. " +
				"This is always an issue in the provider implementation and should be reported to the provider developers.\n\n" +
				"Provider schema difference: " + diff,
		})
	}

	for _, diff := range s.serverProviderMetaSchemaDifferences {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov6.Diagnostic{
			Severity: tfprotov6.DiagnosticSeverityError,
			Summary:  "Invalid Provider Server Combination",
			Detail: "The combined provider has differing provider meta schema implementations across providers. " +
				"Provider meta schemas must be identical across providers. " +
				"This is always an issue in the provider implementation and should be reported to the provider developers.\n\n" +
				"Provider meta schema difference: " + diff,
		})
	}

	for _, dataSourceType := range s.serverDataSourceSchemaDuplicates {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov6.Diagnostic{
			Severity: tfprotov6.DiagnosticSeverityError,
			Summary:  "Invalid Provider Server Combination",
			Detail: "The combined provider has multiple implementations of the same data source type across providers. " +
				"Data source types must be implemented by only one provider. " +
				"This is always an issue in the provider implementation and should be reported to the provider developers.\n\n" +
				"Duplicate data source type: " + dataSourceType,
		})
	}

	for _, resourceType := range s.serverResourceSchemaDuplicates {
		resp.Diagnostics = append(resp.Diagnostics, &tfprotov6.Diagnostic{
			Severity: tfprotov6.DiagnosticSeverityError,
			Summary:  "Invalid Provider Server Combination",
			Detail: "The combined provider has multiple implementations of the same resource type across providers. " +
				"Resource types must be implemented by only one provider. " +
				"This is always an issue in the provider implementation and should be reported to the provider developers.\n\n" +
				"Duplicate resource type: " + resourceType,
		})
	}

	return resp, nil
}
