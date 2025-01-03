package config

import (
	"os"
	"strconv"
	"time"

	exov2 "github.com/exoscale/egoscale/v2"
	exov3 "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
)

// BaseConfig represents the provider structure
type BaseConfig struct {
	Key         string
	Secret      string
	Timeout     time.Duration
	Environment string
	SOSEndpoint string
}

type ExoscaleProviderConfig struct {
	Config      BaseConfig
	ClientV2    *exov2.Client
	ClientV3    *exov3.Client
	Environment string
	SOSEndpoint string
}

func GetMultiEnvDefault(ks []string, dv string) string {
	for _, k := range ks {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}

	return dv
}

func GetEnvDefault(k string, dv string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}

	return dv
}

func GetTimeout() (float64, error) {
	defaultTimeout := config.DefaultTimeout.Seconds()

	timeoutRaw := GetEnvDefault("EXOSCALE_TIMEOUT", "")
	if timeoutRaw != "" {
		timeout, err := strconv.ParseFloat(timeoutRaw, 64)
		if err != nil {
			return defaultTimeout, err
		}

		return timeout, nil
	}

	return defaultTimeout, nil
}
