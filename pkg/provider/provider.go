package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	exov2 "github.com/exoscale/egoscale/v2"

	"github.com/exoscale/terraform-provider-exoscale/exoscale"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/resources/zones"
)

const (
	KeyAttrName             = "key"
	TokenAttrName           = "token"
	SecretAttrName          = "secret"
	ConfigAttrName          = "config"
	ProfileAttrName         = "profile"
	RegionAttrName          = "region"
	ComputeEndpointAttrName = "compute_endpoint"
	DnsEndpointAttrName     = "dns_endpoint"
	EnvironmentAttrName     = "environment"
	TimeoutAttrName         = "timeout"
	DelayAttrName           = "delay"
)

var _ provider.Provider = &ExoscaleProvider{}

type ExoscaleProvider struct{}

type ExoscaleProviderModel struct {
	Key             types.String  `tfsdk:"key"`
	Token           types.String  `tfsdk:"token"`
	Secret          types.String  `tfsdk:"secret"`
	Config          types.String  `tfsdk:"config"`
	Profile         types.String  `tfsdk:"profile"`
	Region          types.String  `tfsdk:"region"`
	ComputeEndpoint types.String  `tfsdk:"compute_endpoint"`
	DnsEndpoint     types.String  `tfsdk:"dns_endpoint"`
	Environment     types.String  `tfsdk:"environment"`
	Timeout         types.Float64 `tfsdk:"timeout"`
	Delay           types.Int64   `tfsdk:"delay"`
}

func (p *ExoscaleProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "exoscale"
}

func (p *ExoscaleProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			KeyAttrName: schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Exoscale API key",
			},
			TokenAttrName: schema.StringAttribute{
				Optional:           true,
				DeprecationMessage: "Use key instead",
			},
			SecretAttrName: schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Exoscale API secret",
			},
			ConfigAttrName: schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: fmt.Sprintf("CloudStack ini configuration filename (by default: %s)", exoscale.DefaultConfig),
			},
			ProfileAttrName: schema.StringAttribute{
				Optional:           true,
				DeprecationMessage: "Use region instead",
			},
			RegionAttrName: schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: fmt.Sprintf("CloudStack ini configuration section name (by default: %s)", exoscale.DefaultProfile),
			},
			ComputeEndpointAttrName: schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: fmt.Sprintf("Exoscale CloudStack API endpoint (by default: %s)", exoscale.DefaultComputeEndpoint),
			},
			DnsEndpointAttrName: schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: fmt.Sprintf("Exoscale DNS API endpoint (by default: %s)", exoscale.DefaultDNSEndpoint),
			},
			EnvironmentAttrName: schema.StringAttribute{
				Optional: true,
			},
			TimeoutAttrName: schema.Float64Attribute{
				Optional: true,
				MarkdownDescription: fmt.Sprintf(
					"Timeout in seconds for waiting on compute resources to become available (by default: %.0f)",
					config.DefaultTimeout.Seconds()),
			},
			DelayAttrName: schema.Int64Attribute{
				Optional:           true,
				DeprecationMessage: "Does nothing",
			},
		},
	}
}

func (p *ExoscaleProviderModel) GetRegion() string {
	if !p.Profile.IsNull() {
		return p.Profile.ValueString()
	}

	if p.Region.IsNull() {
		return providerConfig.GetMultiEnvDefault([]string{
			"EXOSCALE_PROFILE",
			"EXOSCALE_REGION",
			"CLOUDSTACK_PROFILE",
			"CLOUDSTACK_REGION",
		}, exoscale.DefaultProfile)
	}

	return p.Region.ValueString()
}

func (p *ExoscaleProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ExoscaleProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	var key string
	if data.Key.IsNull() {
		key = providerConfig.GetMultiEnvDefault([]string{
			"EXOSCALE_KEY",
			"EXOSCALE_API_KEY",
			"CLOUDSTACK_KEY",
			"CLOUDSTACK_API_KEY",
		}, "")
	} else {
		key = data.Key.ValueString()
	}

	token := data.Token.ValueString()
	if token != "" && key == "" {
		key = token
	}

	var secret string
	if data.Secret.IsNull() {
		secret = providerConfig.GetMultiEnvDefault([]string{
			"EXOSCALE_SECRET",
			"EXOSCALE_SECRET_KEY",
			"EXOSCALE_API_SECRET",
			"CLOUDSTACK_SECRET",
			"CLOUDSTACK_SECRET_KEY",
		}, "")
	} else {
		secret = data.Secret.ValueString()
	}

	var endpoint string
	if data.ComputeEndpoint.IsNull() {
		endpoint = providerConfig.GetMultiEnvDefault([]string{
			"EXOSCALE_ENDPOINT",
			"EXOSCALE_API_ENDPOINT",
			"EXOSCALE_COMPUTE_ENDPOINT",
			"CLOUDSTACK_ENDPOINT",
		}, exoscale.DefaultComputeEndpoint)
	} else {
		endpoint = data.ComputeEndpoint.ValueString()
	}

	var dnsEndpoint string
	if data.DnsEndpoint.IsNull() {
		dnsEndpoint = providerConfig.GetEnvDefault("EXOSCALE_DNS_ENDPOINT", exoscale.DefaultDNSEndpoint)
	} else {
		dnsEndpoint = data.Key.ValueString()
	}

	var configuration *exoscale.Configuration

	if key != "" || secret != "" {
		if key == "" || secret == "" {
			resp.Diagnostics.AddError(
				fmt.Sprintf("key (%#v) and secret (%#v) must be set",
					key,
					secret,
				), "")

			return
		}
	} else {
		var config string
		if data.Config.IsNull() {
			config = providerConfig.GetMultiEnvDefault([]string{
				"EXOSCALE_CONFIG",
				"CLOUDSTACK_CONFIG",
			}, exoscale.DefaultConfig)
		} else {
			config = data.Config.ValueString()
		}

		region := data.GetRegion()

		var err error
		configuration, err = exoscale.ParseConfig(config, key, region)
		if err != nil {
			resp.Diagnostics.AddError(err.Error(), "")

			return
		}
	}

	if configuration != nil {
		if configuration.Endpoint != "" {
			endpoint = configuration.Endpoint
		}

		if configuration.DNSEndpoint != "" {
			dnsEndpoint = configuration.DNSEndpoint
		}

		if configuration.Key != "" {
			key = configuration.Key
		}

		if configuration.Secret != "" {
			secret = configuration.Secret
		}
	}

	var environment string
	if data.Environment.IsNull() {
		environment = providerConfig.GetMultiEnvDefault([]string{
			"EXOSCALE_API_ENVIRONMENT",
		}, exoscale.DefaultEnvironment)
	} else {
		environment = data.Environment.ValueString()
	}

	var timeout float64
	if data.Timeout.IsNull() {
		var err error
		timeout, err = providerConfig.GetTimeout()

		if err != nil {
			resp.Diagnostics.AddError(err.Error(), "")
		}
	} else {
		timeout = data.Timeout.ValueFloat64()
	}

	exov2.UserAgent = exoscale.UserAgent

	baseConfig := providerConfig.BaseConfig{
		Key:             key,
		Secret:          secret,
		Timeout:         exoscale.ConvertTimeout(timeout),
		ComputeEndpoint: endpoint,
		DNSEndpoint:     dnsEndpoint,
		Environment:     environment,
	}

	clv1 := exoscale.GetComputeClient(map[string]interface{}{
		"config": baseConfig,
	})

	clv2, err := exoscale.CreateClient(&baseConfig)
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "")

		return
	}

	_ = clv2

	resp.DataSourceData = &providerConfig.ExoscaleProviderConfig{
		Config:      baseConfig,
		ClientV1:    clv1,
		ClientV2:    clv2,
		Environment: environment,
	}
}

func (p *ExoscaleProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		func() datasource.DataSource {
			return &zones.ZonesDataSource{}
		},
	}
}

func (p *ExoscaleProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{}
}
