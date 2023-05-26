package config

import (
	exov1 "github.com/exoscale/egoscale"
	exov2 "github.com/exoscale/egoscale/v2"

	"github.com/exoscale/terraform-provider-exoscale/exoscale"
)

type ExoscaleProviderConfig struct {
	Config      exoscale.BaseConfig
	ClientV2    *exov2.Client
	ClientV1    *exov1.Client
	Environment string
}
