package exoscale

import (
	"fmt"
	"net/http"
	"time"

	"github.com/exoscale/egoscale"
	exov2 "github.com/exoscale/egoscale/v2"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/terraform-plugin-sdk/helper/logging"
)

const (
	defaultConfig          = "cloudstack.ini"
	defaultProfile         = "cloudstack"
	defaultComputeEndpoint = "https://api.exoscale.com/v1"
	defaultDNSEndpoint     = "https://api.exoscale.com/dns"
	defaultEnvironment     = "api"
	defaultTimeout         = 5 * time.Minute
	defaultGzipUserData    = true
)

// userAgent represents the User Agent to advertise in outgoing HTTP requests.
var userAgent string

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

	httpClient := cleanhttp.DefaultPooledClient()
	httpClient.Transport = &defaultTransport{next: httpClient.Transport}
	if logging.IsDebugOrHigher() {
		httpClient.Transport = logging.NewTransport(
			"exoscale",
			httpClient.Transport,
		)
	}

	client := egoscale.NewClient(
		endpoint,
		config.key,
		config.secret,
		egoscale.WithHTTPClient(httpClient),
		egoscale.WithTimeout(config.timeout),
		egoscale.WithoutV2Client(),
	)

	// During the Exoscale API V1 -> V2 transition, we need to initialize the
	// V2 client independently from the V1 client because of HTTP middleware
	// (http.Transport) clashes.
	// This can be removed once the only API used is V2.
	clientExoV2, err := exov2.NewClient(
		config.key,
		config.secret,
		exov2.ClientOptWithAPIEndpoint(endpoint),
		exov2.ClientOptWithTimeout(config.timeout),
		exov2.ClientOptWithHTTPClient(func() *http.Client {
			hc := cleanhttp.DefaultPooledClient()
			hc.Transport = &defaultTransport{next: hc.Transport}
			if logging.IsDebugOrHigher() {
				hc.Transport = logging.NewTransport("exoscale", hc.Transport)
			}
			return hc
		}()),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to initialize Exoscale API V2 client: %v", err))
	}
	client.Client = clientExoV2

	return client
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

type defaultTransport struct {
	next http.RoundTripper
}

// RoundTrip executes a single HTTP transaction while augmenting requests with custom headers.
func (t *defaultTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("User-Agent", userAgent)

	resp, err := t.next.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
