package exoscale

import (
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
				DefaultFunc: schema.EnvDefaultFunc("EXOSCALE_TIMEOUT", 60),
				Description: "Timeout in seconds for waiting on compute resources to become available",
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
		token:   d.Get("token").(string),
		secret:  d.Get("secret").(string),
		timeout: d.Get("timeout").(int),
	}

	return baseConfig, nil
}
