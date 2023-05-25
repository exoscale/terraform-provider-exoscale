package exoscale

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_getClient(t *testing.T) {
	var (
		testEndpoint = DefaultComputeEndpoint
		testConfig   = BaseConfig{
			Key:     "x",
			Secret:  "x",
			Timeout: DefaultTimeout,
		}
	)

	client := getClient(testEndpoint, map[string]interface{}{"config": testConfig})
	require.Equal(t, testEndpoint, client.Endpoint)
	require.Equal(t, testConfig.Timeout, client.Timeout)
}
