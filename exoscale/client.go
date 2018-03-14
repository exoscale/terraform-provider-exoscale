package exoscale

import (
	"time"

	"github.com/exoscale/egoscale"
)

const defaultConfig = "cloudstack.ini"
const defaultProfile = "cloudstack"
const defaultComputeEndpoint = "https://api.exoscale.ch/compute"
const defaultDNSEndpoint = "https://api.exoscale.ch/dns"
const defaultTimeout = 60 // seconds

// BaseConfig represents the provider structure
type BaseConfig struct {
	key             string
	secret          string
	timeout         int
	computeEndpoint string
	dnsEndpoint     string
	s3Endpoint      string
}

func getClient(endpoint string, meta interface{}) *egoscale.Client {
	config := meta.(BaseConfig)
	timeout := time.Duration(config.timeout) * time.Second
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
