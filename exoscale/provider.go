package exoscale

import (
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
	"gopkg.in/ini.v1"

	"github.com/terraform-providers/terraform-provider-exoscale/version"
)

const (
	legacyAPIVersion = "compute"
	apiVersion       = "v1"
)

func init() {
	egoscale.UserAgent = fmt.Sprintf("Exoscale-Terraform-Provider/%s %s",
		version.ProviderVersion,
		egoscale.UserAgent)
}

// Provider returns a terraform.ResourceProvider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"key": {
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
			"token": {
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: "Use key instead",
			},
			"secret": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "Exoscale API secret",
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"EXOSCALE_SECRET",
					"EXOSCALE_SECRET_KEY",
					"EXOSCALE_API_SECRET",
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
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: "Use region instead",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: fmt.Sprintf("CloudStack ini configuration section name (by default: %s)", defaultProfile),
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"EXOSCALE_PROFILE",
					"EXOSCALE_REGION",
					"CLOUDSTACK_PROFILE",
					"CLOUDSTACK_REGION",
				}, defaultProfile),
			},
			"compute_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: fmt.Sprintf("Exoscale CloudStack API endpoint (by default: %s)", defaultComputeEndpoint),
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"EXOSCALE_ENDPOINT",
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
			"environment": {
				Type:     schema.TypeString,
				Optional: true,
				DefaultFunc: schema.MultiEnvDefaultFunc([]string{
					"EXOSCALE_API_ENVIRONMENT",
				}, defaultEnvironment),
			},
			"timeout": {
				Type:     schema.TypeFloat,
				Required: true,
				Description: fmt.Sprintf(
					"Timeout in seconds for waiting on compute resources to become available (by default: %.0f)",
					defaultTimeout.Seconds()),
				DefaultFunc: schema.EnvDefaultFunc("EXOSCALE_TIMEOUT", defaultTimeout.Seconds()),
			},
			"gzip_user_data": {
				Type:     schema.TypeBool,
				Optional: true,
				Description: fmt.Sprintf(
					"Defines if the user-data of compute instances should be gzipped (by default: %t)",
					defaultGzipUserData),
				DefaultFunc: schema.EnvDefaultFunc("EXOSCALE_GZIP_USER_DATA", defaultGzipUserData),
			},
			"delay": {
				Type:       schema.TypeInt,
				Optional:   true,
				Deprecated: "Does nothing",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"exoscale_affinity":          dataSourceAffinity(),
			"exoscale_compute_ipaddress": dataSourceComputeIPAddress(),
			"exoscale_compute_template":  dataSourceComputeTemplate(),
			"exoscale_domain":            dataSourceDomain(),
			"exoscale_domain_record":     dataSourceDomainRecord(),
			"exoscale_compute":           dataSourceCompute(),
			"exoscale_network":           dataSourceNetwork(),
			"exoscale_security_group":    dataSourceSecurityGroup(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"exoscale_affinity":             resourceAffinity(),
			"exoscale_compute":              resourceCompute(),
			"exoscale_instance_pool":        resourceInstancePool(),
			"exoscale_domain_record":        resourceDomainRecord(),
			"exoscale_domain":               resourceDomain(),
			"exoscale_ipaddress":            resourceIPAddress(),
			"exoscale_network":              resourceNetwork(),
			"exoscale_nic":                  resourceNIC(),
			"exoscale_nlb":                  resourceNLB(),
			"exoscale_nlb_service":          resourceNLBService(),
			"exoscale_secondary_ipaddress":  resourceSecondaryIPAddress(),
			"exoscale_security_group_rule":  resourceSecurityGroupRule(),
			"exoscale_security_group_rules": resourceSecurityGroupRules(),
			"exoscale_security_group":       resourceSecurityGroup(),
			"exoscale_ssh_keypair":          resourceSSHKeypair(),
		},

		ConfigureFunc: providerConfigure,
	}
}

func providerConfigure(d *schema.ResourceData) (interface{}, error) {
	key, keyOK := d.GetOk("key")
	secret, secretOK := d.GetOk("secret")
	endpoint := d.Get("compute_endpoint").(string)
	dnsEndpoint := d.Get("dns_endpoint").(string)
	environment := d.Get("environment").(string)

	// deprecation support
	token, tokenOK := d.GetOk("token")
	if tokenOK && !keyOK {
		keyOK = tokenOK
		key = token
	}

	if keyOK || secretOK {
		if !keyOK || !secretOK {
			return nil, fmt.Errorf("key (%#v) and secret (%#v) must be set", key.(string), secret.(string))
		}
	} else {
		config := d.Get("config").(string)
		region := d.Get("region")

		// deprecation support
		profile, profileOK := d.GetOk("profile")
		if profileOK && profile.(string) != "" {
			region = profile
		}

		// Support `~/`
		usr, err := user.Current()
		if err == nil {
			if strings.HasPrefix(config, "~/") {
				config = filepath.Join(usr.HomeDir, config[2:])
			}
		}

		// Convert relative path to absolute
		config, _ = filepath.Abs(config)
		localConfig, _ := filepath.Abs("cloudstack.ini")

		inis := []string{
			config,
			localConfig,
		}

		if usr != nil {
			inis = append(inis, filepath.Join(usr.HomeDir, ".cloudstack.ini"))
		}

		// Stops at the first file that exists
		config = ""
		for _, i := range inis {
			if _, err := os.Stat(i); err != nil {
				continue
			}
			config = i
			break
		}

		if config == "" {
			return nil, fmt.Errorf("key (%s), secret are missing, or config file not found within: %s", key, strings.Join(inis, ", "))
		}

		cfg, err := ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true}, config)
		if err != nil {
			return nil, fmt.Errorf("Config file not loaded: %s", err)
		}

		section, err := cfg.GetSection(region.(string))
		if err != nil {
			sections := strings.Join(cfg.SectionStrings(), ", ")
			return nil, fmt.Errorf("%s. Existing sections: %s", err, sections)
		}

		t, err := section.GetKey("key")
		if err != nil {
			return nil, err
		}
		key = t.String()

		s, err := section.GetKey("secret")
		if err != nil {
			return nil, err
		}
		secret = s.String()

		e, err := section.GetKey("endpoint")
		if err == nil {
			endpoint = e.String()
			dnsEndpoint = strings.Replace(endpoint, "/"+apiVersion, "/dns", 1)
			if strings.Contains(dnsEndpoint, "/"+legacyAPIVersion) {
				dnsEndpoint = strings.Replace(endpoint, "/"+legacyAPIVersion, "/dns", 1)
			}
		}
	}

	baseConfig := BaseConfig{
		key:             key.(string),
		secret:          secret.(string),
		timeout:         time.Duration(int64(d.Get("timeout").(float64)) * int64(time.Second)),
		computeEndpoint: endpoint,
		dnsEndpoint:     dnsEndpoint,
		environment:     environment,
		gzipUserData:    d.Get("gzip_user_data").(bool),
	}

	return baseConfig, nil
}

func getZoneByName(ctx context.Context, client *egoscale.Client, zoneName string) (*egoscale.Zone, error) {
	zone := &egoscale.Zone{}

	id, err := egoscale.ParseUUID(zoneName)
	if err != nil {
		zone.Name = zoneName
	} else {
		zone.ID = id
	}

	resp, err := client.GetWithContext(ctx, zone)
	if err != nil {
		return nil, err
	}

	return resp.(*egoscale.Zone), nil
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

type resourceIDStringer interface {
	Id() string
}

func resourceIDString(d resourceIDStringer, name string) string {
	id := d.Id()
	if id == "" {
		id = "<new resource>"
	}

	return fmt.Sprintf("%s (ID = %s)", name, id)
}
