package config

import (
	"context"
	"errors"
	"time"

	egoscale "github.com/exoscale/egoscale/v2"
	v3 "github.com/exoscale/egoscale/v3"
)

const (
	// FIXME: defaultZone is used for global resources management, as at the
	//  time of this implementation the Exoscale public API V2 doesn't
	//  expose a global endpoint – only zone-local endpoints.
	//  This should be removed once the Exoscale public API V2 exposes a
	//  global endpoint.
	DefaultZone = "ch-gva-2"

	DefaultEnvironment = "api"
	DefaultTimeout     = 60 * time.Minute

	ComputeMaxUserDataLength = 32768
)

var Zones = []string{
	"ch-gva-2",
	"ch-dk-2",
	"at-vie-1",
	"at-vie-2",
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

// GetClientV3 builds egoscale v3 client from configuration parameters in meta field
func GetClientV3(meta interface{}) (*v3.Client, error) {
	c := meta.(map[string]interface{})
	if client, ok := c["clientV3"]; ok {
		return client.(*v3.Client), nil
	}
	return nil, errors.New("API client not found")
}

func GetClientV3WithZone(ctx context.Context, meta interface{}, zone string) (*v3.Client, error) {
	client, err := GetClientV3(meta)
	if err != nil {
		return nil, err
	}

	endpoint, err := client.GetZoneAPIEndpoint(ctx, v3.ZoneName(zone))
	if err != nil {
		return nil, err
	}

	return client.WithEndpoint(endpoint), nil
}

// GetEnvironment returns current environment
func GetEnvironment(meta interface{}) string {
	c := meta.(map[string]interface{})
	if env, ok := c["environment"]; ok {
		return env.(string)
	}
	return DefaultEnvironment
}
