package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	exov2 "github.com/exoscale/egoscale/v2"

	"github.com/exoscale/terraform-provider-exoscale/exoscale"
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
	GzipUserDataAttrName    = "gzip_user_data"
	DelayAttrName           = "delay"
)

var _ provider.Provider = &ExoscaleProvider{}

type ExoscaleProvider struct {
	// Version is an example field that can be set with an actual provider
	// version on release, "dev" when the provider is built and ran locally,
	// and "test" when running acceptance testing.
	Version string
}

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
	GzipUserData    types.Bool    `tfsdk:"gzip_user_data"`
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
					exoscale.DefaultTimeout.Seconds()),
			},
			GzipUserDataAttrName: schema.BoolAttribute{
				Optional: true,
				MarkdownDescription: fmt.Sprintf(
					"Defines if the user-data of compute instances should be gzipped (by default: %t)",
					exoscale.DefaultGzipUserData),
			},
			DelayAttrName: schema.Int64Attribute{
				Optional:           true,
				DeprecationMessage: "Does nothing",
			},
		},
	}
}

type ExoscaleProviderConfig struct {
	Config      exoscale.BaseConfig
	Client      *exov2.Client
	Environment string
}

func multiEnvDefault(ks []string, dv string) string {
	for _, k := range ks {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}

	return dv
}

func envDefault(k string, dv string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}

	return dv
}

func (p *ExoscaleProviderModel) GetRegion() string {
	if !p.Profile.IsNull() {
		return p.Profile.ValueString()
	}

	if p.Region.IsNull() {
		return multiEnvDefault([]string{
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
		key = multiEnvDefault([]string{
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
		secret = multiEnvDefault([]string{
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
		endpoint = multiEnvDefault([]string{
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
		dnsEndpoint = envDefault("EXOSCALE_DNS_ENDPOINT", exoscale.DefaultDNSEndpoint)
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
			config = multiEnvDefault([]string{
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

		if configuration.Secret != "" {
			secret = configuration.Secret
		}
	}

	var environment string
	if data.Environment.IsNull() {
		environment = multiEnvDefault([]string{
			"EXOSCALE_API_ENVIRONMENT",
		}, exoscale.DefaultEnvironment)
	} else {
		environment = data.Environment.ValueString()
	}

	var timeout float64
	if data.Timeout.IsNull() {
		defaultTimeout := exoscale.DefaultTimeout.Seconds()

		timeoutRaw := envDefault("EXOSCALE_TIMEOUT", "")

		var err error
		timeout, err = strconv.ParseFloat(timeoutRaw, 64)
		if err != nil {
			resp.Diagnostics.AddError(err.Error(), "")

			timeout = defaultTimeout
		}
	}

	var gzipUserData bool
	if data.GzipUserData.IsNull() {
		gzipUserDataRaw := envDefault("EXOSCALE_GZIP_USER_DATA", "")
		if gzipUserDataRaw == "" {
			gzipUserData = exoscale.DefaultGzipUserData
		} else {
			var err error
			gzipUserData, err = strconv.ParseBool(gzipUserDataRaw)
			if err != nil {
				resp.Diagnostics.AddError(err.Error(), "")

				gzipUserData = exoscale.DefaultGzipUserData
			}
		}
	} else {
		gzipUserData = data.GzipUserData.ValueBool()
	}

	exov2.UserAgent = exoscale.UserAgent

	baseConfig := exoscale.BaseConfig{
		Key:             key,
		Secret:          secret,
		Timeout:         exoscale.ConvertTimeout(timeout),
		ComputeEndpoint: endpoint,
		DNSEndpoint:     dnsEndpoint,
		Environment:     environment,
		GZIPUserData:    gzipUserData,
	}

	clv2, err := exoscale.CreateClient(&baseConfig)
	if err != nil {
		resp.Diagnostics.AddError(err.Error(), "")

		return
	}

	resp.ResourceData = ExoscaleProviderConfig{
		Config:      baseConfig,
		Client:      clv2,
		Environment: environment,
	}
}

func (p *ExoscaleProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// Provider specific implementation
	}
}

func (p *ExoscaleProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		// Provider specific implementation
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ExoscaleProvider{
			Version: version,
		}
	}
}
