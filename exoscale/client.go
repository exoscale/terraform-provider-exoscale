package exoscale

import (
	"time"

	"github.com/exoscale/egoscale"
)

const defaultConfig = "cloudstack.ini"
const defaultProfile = "cloudstack"
const defaultComputeEndpoint = "https://api.exoscale.ch/compute"
const defaultDNSEndpoint = "https://api.exoscale.ch/dns"
const defaultTimeout = time.Duration(60) * time.Second
const defaultGzipUserData = true

// BaseConfig represents the provider structure
type BaseConfig struct {
	key             string
	secret          string
	timeout         time.Duration
	computeEndpoint string
	dnsEndpoint     string
	s3Endpoint      string
	gzipUserData    bool
}

func getClient(endpoint string, meta interface{}) *egoscale.Client {
	config := meta.(BaseConfig)
	timeout := config.timeout
	return egoscale.NewClientWithTimeout(endpoint, config.key, config.secret, timeout)
}

// GetComputeClient builds a CloudStack client
func GetComputeClient(meta interface{}) *egoscale.Client {
	config := meta.(BaseConfig)
	return getClient(config.computeEndpoint, meta)
}

// GetDNSClient builds a DNS client
func GetDNSClient(meta interface{}) *egoscale.Client {
	config := meta.(BaseConfig)
	return getClient(config.dnsEndpoint, meta)
}
