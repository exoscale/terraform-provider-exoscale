package exoscale

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
)

func Test_getClient(t *testing.T) {
	var (
		testEndpoint = DefaultComputeEndpoint
		testConfig   = providerConfig.BaseConfig{
			Key:     "x",
			Secret:  "x",
			Timeout: config.DefaultTimeout,
		}
	)

	client := getClient(testEndpoint, map[string]interface{}{"config": testConfig})
	require.Equal(t, testEndpoint, client.Endpoint)
	require.Equal(t, testConfig.Timeout, client.Timeout)
}
