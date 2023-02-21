package testutils

import (
	"os"

	egoscale "github.com/exoscale/egoscale/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/exoscale/terraform-provider-exoscale/exoscale"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
)

const (
	Prefix                         = "test-terraform-exoscale"
	TestDescription                = "Created by the terraform-exoscale provider"
	TestZoneName                   = "ch-dk-2"
	TestInstanceTemplateName       = "Linux Ubuntu 20.04 LTS 64-bit"
	TestInstanceTemplateUsername   = "ubuntu"
	TestInstanceTemplateFilter     = "featured"
	TestInstanceTemplateVisibility = "public"

	TestInstanceTypeIDTiny   = "b6cd1ff5-3a2f-4e9d-a4d1-8988c1191fe8"
	TestInstanceTypeIDSmall  = "21624abb-764e-4def-81d7-9fc54b5957fb"
	TestInstanceTypeIDMedium = "b6e9d1e8-89fc-4db3-aaa4-9b4c5b1d0844"
)

// Providers returns all providers used during acceptance testing.
func Providers() map[string]func() (*schema.Provider, error) {
	testAccProvider := exoscale.Provider()
	return map[string]func() (*schema.Provider, error){
		"exoscale": func() (*schema.Provider, error) {
			return testAccProvider, nil
		},
	}
}

func APIClient() (*egoscale.Client, error) {
	client, err := egoscale.NewClient(
		os.Getenv("EXOSCALE_API_KEY"),
		os.Getenv("EXOSCALE_API_SECRET"),
	)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func TestEnvironment() string {
	env := os.Getenv("EXOSCALE_API_ENVIRONMENT")
	if env == "" {
		env = config.DefaultEnvironment
	}

	return env
}
