package exoscale

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_getClient(t *testing.T) {
	var (
		testEndpoint = defaultComputeEndpoint
		testConfig   = BaseConfig{
			key:     "x",
			secret:  "x",
			timeout: defaultTimeout,
		}
	)

	client := getClient(testEndpoint, testConfig)
	require.Equal(t, testEndpoint, client.Endpoint)
	require.Equal(t, testConfig.timeout, client.Timeout)
	require.IsType(t, &defaultTransport{}, client.HTTPClient.Transport)
}
