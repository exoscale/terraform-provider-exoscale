package exoscale

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("EXOSCALE_KEY", nil),
				Description: "Exoscale API key",
			},
			"secret": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("EXOSCALE_SECRET", nil),
				Description: "Exoscale API secret",
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"exoscale_compute": 		computeResource(),
			"exoscale_ssh":     		sshResource(),
			"exoscale_affinity":		affinityResource(),
			"exoscale_securitygroup":	securityGroupResource(),
			"exoscale_dns":				dnsResource(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	baseConfig := Client{
		token:  d.Get("token").(string),
		secret: d.Get("secret").(string),
	}

	return baseConfig, nil
}
