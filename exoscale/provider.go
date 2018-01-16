package exoscale

import (
	"fmt"
	"log"
	"strings"

	"github.com/exoscale/egoscale"
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
			"delay": {
				Type:        schema.TypeInt,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("EXOSCALE_DELAY", defaultDelayBeforeRetry),
				Description: fmt.Sprintf(
					"Delay in seconds representing the polling time interval (by default: %d)",
					defaultDelayBeforeRetry),
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
				DefaultFunc: schema.EnvDefaultFunc("EXOSCALE_DNS_ENDPOINT", defaultDNSEndpoint),
				Description: fmt.Sprintf("Exoscale DNS API endpoint (by default: %s)", defaultDNSEndpoint),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"exoscale_compute":             computeResource(),
			"exoscale_ssh_keypair":         sshResource(),
			"exoscale_affinity":            affinityGroupResource(),
			"exoscale_domain":              domainResource(),
			"exoscale_domain_record":       domainRecordResource(),
			"exoscale_security_group":      securityGroupResource(),
			"exoscale_security_group_rule": securityGroupRuleResource(),
			"exoscale_ipaddress":           elasticIPResource(),
			"exoscale_secondary_ipaddress": secondaryIPResource(),
			"exoscale_network":             networkResource(),
			"exoscale_nic":                 nicResource(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	timeout := d.Get("timeout").(int)
	delay := d.Get("delay").(int)
	retries := timeout/delay - 1 // the first try is free

	log.Printf("Async calls: timeout %d - delay %d - retries %d\n", timeout, delay, retries)

	baseConfig := BaseConfig{
		token:           d.Get("token").(string),
		secret:          d.Get("secret").(string),
		timeout:         timeout,
		computeEndpoint: d.Get("compute_endpoint").(string),
		dnsEndpoint:     d.Get("dns_endpoint").(string),
		async: egoscale.AsyncInfo{
			Retries: retries,
			Delay:   delay,
		},
	}

	return baseConfig, nil
}

func getZoneByName(client *egoscale.Client, zoneName string) (*egoscale.Zone, error) {
	resp, err := client.Request(&egoscale.ListZones{
		Name: strings.ToLower(zoneName),
	})

	if err != nil {
		return nil, err
	}

	zones := resp.(*egoscale.ListZonesResponse)
	if zones.Count == 0 {
		return nil, fmt.Errorf("Zone not found %s", zoneName)
	}

	return &(zones.Zone[0]), nil
}

func getNetworkOfferingByName(client *egoscale.Client, zoneName string) (*egoscale.NetworkOffering, error) {
	resp, err := client.Request(&egoscale.ListNetworkOfferings{
		Name: strings.ToLower(zoneName),
	})

	if err != nil {
		return nil, err
	}

	networks := resp.(*egoscale.ListNetworkOfferingsResponse)
	if networks.Count == 0 {
		return nil, fmt.Errorf("NetworkOffering not found %s", zoneName)
	}

	return &(networks.NetworkOffering[0]), nil
}

// handleNotFound inspects the CloudStack ErrorCode to guess if the resource is missing
// and then removes it (unsetting the ID) and succeeds.
func handleNotFound(d *schema.ResourceData, err error) error {
	if r, ok := err.(*egoscale.ErrorResponse); ok {
		if r.ErrorCode == egoscale.ParamError {
			d.SetId("")
			return nil
		}
		return r
	}
	return err
}
