package exoscale

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/ini.v1"

	"github.com/exoscale/egoscale"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	tfmeta "github.com/hashicorp/terraform-plugin-sdk/v2/meta"

	"github.com/exoscale/terraform-provider-exoscale/version"
)

const (
	legacyAPIVersion = "compute"
	apiVersion       = "v1"

	// FIXME: defaultZone is used for global resources management, as at the
	//  time of this implementation the Exoscale public API V2 doesn't
	//  expose a global endpoint – only zone-local endpoints.
	//  This should be removed once the Exoscale public API V2 exposes a
	//  global endpoint.
	defaultZone = "ch-gva-2"
)

func init() {
	userAgent = fmt.Sprintf("Exoscale-Terraform-Provider/%s (%s) Terraform-SDK/%s %s",
		version.Version,
		version.Commit,
		tfmeta.SDKVersionString(),
		egoscale.UserAgent)
}

// Provider returns an Exoscale Provider.
func Provider() *schema.Provider {
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
					"EXOSCALE_API_ENDPOINT",
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
			"exoscale_affinity":            dataSourceAffinity(),
			"exoscale_anti_affinity_group": dataSourceAntiAffinityGroup(),
			"exoscale_compute":             dataSourceCompute(),
			"exoscale_compute_instance":    dataSourceComputeInstance(),
			"exoscale_instance_pool":       dataSourceInstancePool(),
			"exoscale_compute_ipaddress":   dataSourceComputeIPAddress(),
			"exoscale_compute_template":    dataSourceComputeTemplate(),
			"exoscale_domain":              dataSourceDomain(),
			"exoscale_domain_record":       dataSourceDomainRecord(),
			"exoscale_elastic_ip":          dataSourceElasticIP(),
			"exoscale_network":             dataSourceNetwork(),
			"exoscale_nlb":                 dataSourceNLB(),
			"exoscale_private_network":     dataSourcePrivateNetwork(),
			"exoscale_security_group":      dataSourceSecurityGroup(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"exoscale_affinity":             resourceAffinity(),
			"exoscale_anti_affinity_group":  resourceAntiAffinityGroup(),
			"exoscale_compute":              resourceCompute(),
			"exoscale_compute_instance":     resourceComputeInstance(),
			"exoscale_database":             resourceDatabase(),
			"exoscale_domain":               resourceDomain(),
			"exoscale_domain_record":        resourceDomainRecord(),
			"exoscale_elastic_ip":           resourceElasticIP(),
			"exoscale_instance_pool":        resourceInstancePool(),
			"exoscale_ipaddress":            resourceIPAddress(),
			"exoscale_network":              resourceNetwork(),
			"exoscale_nic":                  resourceNIC(),
			"exoscale_nlb":                  resourceNLB(),
			"exoscale_nlb_service":          resourceNLBService(),
			"exoscale_private_network":      resourcePrivateNetwork(),
			"exoscale_secondary_ipaddress":  resourceSecondaryIPAddress(),
			"exoscale_security_group":       resourceSecurityGroup(),
			"exoscale_security_group_rule":  resourceSecurityGroupRule(),
			"exoscale_security_group_rules": resourceSecurityGroupRules(),
			"exoscale_sks_cluster":          resourceSKSCluster(),
			"exoscale_sks_kubeconfig":       resourceSKSKubeconfig(),
			"exoscale_sks_nodepool":         resourceSKSNodepool(),
			"exoscale_ssh_key":              resourceSSHKey(),
			"exoscale_ssh_keypair":          resourceSSHKeypair(),
		},

		ConfigureContextFunc: providerConfigure,
	}
}

func providerConfigure(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

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
			return nil, diag.Errorf(
				"key (%#v) and secret (%#v) must be set",
				key.(string),
				secret.(string),
			)
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
			return nil, diag.Errorf(
				"key (%s), secret are missing, or config file not found within: %s",
				key,
				strings.Join(inis, ", "),
			)
		}

		cfg, err := ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true}, config)
		if err != nil {
			return nil, diag.Errorf("config file not loaded: %s", err)
		}

		section, err := cfg.GetSection(region.(string))
		if err != nil {
			sections := strings.Join(cfg.SectionStrings(), ", ")
			return nil, diag.Errorf("%s. Existing sections: %s", err, sections)
		}

		t, err := section.GetKey("key")
		if err != nil {
			return nil, diag.FromErr(err)
		}
		key = t.String()

		s, err := section.GetKey("secret")
		if err != nil {
			return nil, diag.FromErr(err)
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

	return baseConfig, diags
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
	} else if errors.Is(err, egoscale.ErrNotFound) || errors.Is(err, exoapi.ErrNotFound) {
		d.SetId("")
		return nil
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

// zonedStateContextFunc is an alternative resource importer function to be
// used for importing zone-local resources, where the resource ID is expected
// to be suffixed with "@ZONE" (e.g. "c01af84d-6ac6-4784-98bb-127c98be8258@ch-gva-2").
// Upon successful execution, the returned resource state contains the ID of the
// resource and the "zone" attribute set to the value parsed from the import ID.
func zonedStateContextFunc(_ context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
	parts := strings.SplitN(d.Id(), "@", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf(`invalid ID %q, expected format "<ID>@<ZONE>"`, d.Id())
	}

	d.SetId(parts[0])

	if err := d.Set("zone", parts[1]); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}
