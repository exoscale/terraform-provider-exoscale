package config

import (
	"errors"
	"fmt"
	"time"

	egoscale "github.com/exoscale/egoscale/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/meta"

	"github.com/exoscale/terraform-provider-exoscale/pkg/version"
)

const (
	// FIXME: defaultZone is used for global resources management, as at the
	//  time of this implementation the Exoscale public API V2 doesn't
	//  expose a global endpoint – only zone-local endpoints.
	//  This should be removed once the Exoscale public API V2 exposes a
	//  global endpoint.
	DefaultZone = "ch-gva-2"

	DefaultConfig          = "cloudstack.ini"
	DefaultProfile         = "cloudstack"
	DefaultComputeEndpoint = "https://api.exoscale.com/v1"
	DefaultDNSEndpoint     = "https://api.exoscale.com/dns"
	DefaultEnvironment     = "api"
	DefaultTimeout         = 5 * time.Minute
	DefaultGzipUserData    = true

	ComputeMaxUserDataLength = 32768
)

// userAgent represents the User Agent to advertise in outgoing HTTP requests.
var UserAgent = fmt.Sprintf("Exoscale-Terraform-Provider/%s (%s) Terraform-SDK/%s %s",
	version.Version,
	version.Commit,
	meta.SDKVersionString(),
	egoscale.UserAgent)

var Zones = []string{
	"ch-gva-2",
	"ch-dk-2",
	"at-vie-1",
	"de-fra-1",
	"bg-sof-1",
	"de-muc-1",
}

// GetClient builds egoscale client from configuration parameters in meta field
func GetClient(meta interface{}) (*egoscale.Client, error) {
	c := meta.(map[string]interface{})
	if client, ok := c["client"]; ok {
		return client.(*egoscale.Client), nil
	}
	return nil, errors.New("API client not found")
}

// GetEnvironment returns current environment
func GetEnvironment(meta interface{}) string {
	c := meta.(map[string]interface{})
	if env, ok := c["environment"]; ok {
		return env.(string)
	}
	return DefaultEnvironment
}
