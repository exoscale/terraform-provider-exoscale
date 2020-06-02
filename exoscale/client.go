package exoscale

import (
	"time"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform-plugin-sdk/helper/logging"
)

const defaultConfig = "cloudstack.ini"
const defaultProfile = "cloudstack"
const defaultComputeEndpoint = "https://api.exoscale.com/v1"
const defaultDNSEndpoint = "https://api.exoscale.com/dns"
const defaultEnvironment = "api"
const defaultTimeout = 5 * time.Minute
const defaultGzipUserData = true

// BaseConfig represents the provider structure
type BaseConfig struct {
	key             string
	secret          string
	timeout         time.Duration
	computeEndpoint string
	dnsEndpoint     string
	environment     string
	gzipUserData    bool
	computeClient   *egoscale.Client
	dnsClient       *egoscale.Client
}

func getClient(endpoint string, meta interface{}) *egoscale.Client {
	config := meta.(BaseConfig)
	cs := egoscale.NewClient(endpoint, config.key, config.secret)

	cs.Timeout = config.timeout
	cs.HTTPClient = cleanhttp.DefaultPooledClient()
	cs.HTTPClient.Timeout = config.timeout

	if logging.IsDebugOrHigher() {
		cs.HTTPClient.Transport = logging.NewTransport(
			"exoscale",
			cs.HTTPClient.Transport,
		)
	}

	return cs
}

// GetComputeClient builds a CloudStack client
func GetComputeClient(meta interface{}) *egoscale.Client {
	config := meta.(BaseConfig)
	if config.computeClient == nil {
		config.computeClient = getClient(config.computeEndpoint, meta)
	}
	return config.computeClient
}

// GetDNSClient builds a DNS client
func GetDNSClient(meta interface{}) *egoscale.Client {
	config := meta.(BaseConfig)
	if config.dnsClient == nil {
		config.dnsClient = getClient(config.dnsEndpoint, meta)
	}
	return config.dnsClient
}

func getEnvironment(meta interface{}) string {
	config := meta.(BaseConfig)
	if config.environment == "" {
		return defaultEnvironment
	}
	return config.environment
}
