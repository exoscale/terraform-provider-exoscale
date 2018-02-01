package exoscale

import (
	"fmt"
	"log"
	"strings"

	"github.com/exoscale/egoscale"
	"github.com/go-ini/ini"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"token": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Exoscale API key",
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"EXOSCALE_KEY",
					"EXOSCALE_API_KEY",
					"CLOUDSTACK_KEY",
					"CLOUDSTACK_API_KEY",
				}, nil),
			},
			"secret": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Exoscale API secret",
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"EXOSCALE_SECRET",
					"EXOSCALE_SECRET_KEY",
					"CLOUDSTACK_SECRET",
					"CLOUDSTACK_SECRET_KEY",
				}, nil),
			},
			"config": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: fmt.Sprintf("CloudStack ini configuration filename (by default: %s)", defaultConfig),
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"EXOSCALE_CONFIG",
					"CLOUDSTACK_CONFIG",
				}, defaultConfig),
			},
			"profile": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: fmt.Sprintf("CloudStack ini configuration section name (by default: %s)", defaultProfile),
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"EXOSCALE_PROFILE",
					"CLOUDSTACK_PROFILE",
				}, defaultProfile),
			},
			"compute_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: fmt.Sprintf("Exoscale CloudStack API endpoint (by default: %s)", defaultComputeEndpoint),
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"EXOSCALE_COMPUTE_ENDPOINT",
					"CLOUDSTACK_ENDPOINT",
				}, defaultComputeEndpoint),
			},
			"dns_endpoint": {
				Type:        schema.TypeString,
				Required:    true,
				Description: fmt.Sprintf("Exoscale DNS API endpoint (by default: %s)", defaultDNSEndpoint),
				DefaultFunc: schema.EnvDefaultFunc("EXOSCALE_DNS_ENDPOINT", defaultDNSEndpoint),
			},
			"timeout": {
				Type:     schema.TypeInt,
				Required: true,
				Description: fmt.Sprintf(
					"Timeout in seconds for waiting on compute resources to become available (by default: %d)",
					defaultTimeout),
				DefaultFunc: schema.EnvDefaultFunc("EXOSCALE_TIMEOUT", defaultTimeout),
			},
			"delay": {
				Type:        schema.TypeInt,
				Required:    true,
				DefaultFunc: schema.EnvDefaultFunc("EXOSCALE_DELAY", defaultDelayBeforeRetry),
				Description: fmt.Sprintf(
					"Delay in seconds representing the polling time interval (by default: %d)",
					defaultDelayBeforeRetry),
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
	token, tokenOK := d.GetOk("token")
	secret, secretOK := d.GetOk("secret")
	endpoint := d.Get("compute_endpoint").(string)

	if tokenOK || secretOK {
		if !tokenOK || !secretOK {
			return nil, fmt.Errorf("token (%#v) and secret (%#v) must be set", token.(string), secret.(string))
		}
	} else {
		config := d.Get("config").(string)
		profile := d.Get("profile").(string)

		cfg, err := ini.LooseLoad(config, "cloudstack.ini", "~/.cloudstack.ini")
		if err != nil {
			return nil, err
		}
		section, err := cfg.GetSection(profile)
		if err != nil {
			return nil, err
		}

		t, err := section.GetKey("key")
		if err != nil {
			return nil, err
		}
		token = t.String()

		s, err := section.GetKey("secret")
		if err != nil {
			return nil, err
		}
		secret = s.String()

		e, err := section.GetKey("endpoint")
		if err == nil {
			endpoint = e.String()
		}
	}

	timeout := d.Get("timeout").(int)
	delay := d.Get("delay").(int)
	retries := timeout/delay - 1 // the first try is free
	log.Printf("Async calls: timeout %d - delay %d - retries %d\n", timeout, delay, retries)

	baseConfig := BaseConfig{
		token:           token.(string),
		secret:          secret.(string),
		timeout:         timeout,
		computeEndpoint: endpoint,
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
