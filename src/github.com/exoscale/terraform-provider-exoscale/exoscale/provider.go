package exoscale

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"EXOSCALE_KEY", "CLOUDSTACK_API_KEY"}, nil),
				Description: "Exoscale API key",
			},
			"secret": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{"EXOSCALE_SECRET", "CLOUDSTACK_SECRET_KEY"}, nil),
				Description: "Exoscale API secret",
			},
			"timeout": {
				Type:        schema.TypeInt,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("EXOSCALE_TIMEOUT", defaultTimeout),
				Description: fmt.Sprintf(
					"Timeout in seconds for waiting on compute resources to become available (by default: %d)",
					defaultTimeout),
			},
			"compute_endpoint": {
				Type:     schema.TypeString,
				Required: true,
				DefaultFunc: schema.MultiEnvDefaultFunc(
					[]string{"EXOSCALE_COMPUTE_ENDPOINT", "CLOUDSTACK_ENDPOINT"},
					defaultComputeEndpoint),
				Description: fmt.Sprintf("Exoscale CloudStack API endpoint (by default: %s)", defaultComputeEndpoint),
			},
			"dns_endpoint": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("EXOSCALE_DNS_ENDPOINT", defaultDnsEndpoint),
				Description: fmt.Sprintf("Exoscale DNS API endpoint (by default: %s)", defaultDnsEndpoint),
			},
			"s3_endpoint": {
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("EXOSCALE_S3_ENDPOINT", defaultS3Endpoint),
				Description: fmt.Sprintf("Exoscale DNS API endpoint (by default: %s)", defaultS3Endpoint),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"exoscale_compute":       computeResource(),
			"exoscale_ssh":           sshResource(),
			"exoscale_affinity":      affinityResource(),
			"exoscale_securitygroup": securityGroupResource(),
			"exoscale_dns":           dnsResource(),
			"exoscale_s3bucket":      s3BucketResource(),
			"exoscale_s3object":      s3ObjectResource(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	baseConfig := BaseConfig{
		token:            d.Get("token").(string),
		secret:           d.Get("secret").(string),
		timeout:          d.Get("timeout").(int),
		compute_endpoint: d.Get("compute_endpoint").(string),
		dns_endpoint:     d.Get("dns_endpoint").(string),
		s3_endpoint:      d.Get("s3_endpoint").(string),
	}

	return baseConfig, nil
}
