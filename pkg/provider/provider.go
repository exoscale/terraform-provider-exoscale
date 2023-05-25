package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// Ensure the implementation satisfies the provider.Provider interface.
var _ provider.Provider = &ExampleCloudProvider{}

type ExampleCloudProvider struct {
	// Version is an example field that can be set with an actual provider
	// version on release, "dev" when the provider is built and ran locally,
	// and "test" when running acceptance testing.
	Version string
}

func (p *ExampleCloudProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "exoscale"
}

// Schema satisfies the provider.Provider interface for ExampleCloudProvider.
func (p *ExampleCloudProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			// Provider specific implementation.
		},
	}
}

// Configure satisfies the provider.Provider interface for ExampleCloudProvider.
func (p *ExampleCloudProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	// Provider specific implementation.
}

// DataSources satisfies the provider.Provider interface for ExampleCloudProvider.
func (p *ExampleCloudProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// Provider specific implementation
	}
}

// Resources satisfies the provider.Provider interface for ExampleCloudProvider.
func (p *ExampleCloudProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// Provider specific implementation
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ExampleCloudProvider{
			Version: version,
		}
	}
}
