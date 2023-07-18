package exoscale

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	ini "gopkg.in/ini.v1"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/resources/anti_affinity_group"
	"github.com/exoscale/terraform-provider-exoscale/pkg/resources/instance"
	"github.com/exoscale/terraform-provider-exoscale/pkg/resources/instance_pool"

	"github.com/exoscale/egoscale"
	exov2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
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
	schema.DescriptionKind = schema.StringMarkdown

	schema.SchemaDescriptionBuilder = func(s *schema.Schema) string {
		if s.ForceNew {
			// we add an indication that modifying this attribute, will force the creation of a new resource.
			return fmt.Sprintf("❗ %s", s.Description)
		}

		return s.Description
	}
}

// Provider returns an Exoscale Provider.
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"key": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Exoscale API key",
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
			},
			"config": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: fmt.Sprintf("CloudStack ini configuration filename (by default: %s)", DefaultConfig),
			},
			"profile": {
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: "Use region instead",
			},
			"region": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: fmt.Sprintf("CloudStack ini configuration section name (by default: %s)", DefaultProfile),
			},
			"compute_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: fmt.Sprintf("Exoscale CloudStack API endpoint (by default: %s)", DefaultComputeEndpoint),
			},
			"dns_endpoint": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: fmt.Sprintf("Exoscale DNS API endpoint (by default: %s)", DefaultDNSEndpoint),
			},
			"environment": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"timeout": {
				Type:     schema.TypeFloat,
				Optional: true,
				Description: fmt.Sprintf(
					"Timeout in seconds for waiting on compute resources to become available (by default: %.0f)",
					config.DefaultTimeout.Seconds()),
			},
			"delay": {
				Type:       schema.TypeInt,
				Optional:   true,
				Deprecated: "Does nothing",
			},
		},

		DataSourcesMap: map[string]*schema.Resource{
			"exoscale_affinity":              dataSourceAffinity(),
			"exoscale_anti_affinity_group":   anti_affinity_group.DataSource(),
			"exoscale_compute":               dataSourceCompute(),
			"exoscale_compute_instance":      instance.DataSource(),
			"exoscale_compute_instance_list": instance.DataSourceList(),
			"exoscale_compute_ipaddress":     dataSourceComputeIPAddress(),
			"exoscale_compute_template":      dataSourceComputeTemplate(),
			"exoscale_domain":                dataSourceDomain(),
			"exoscale_domain_record":         dataSourceDomainRecord(),
			"exoscale_elastic_ip":            dataSourceElasticIP(),
			"exoscale_instance_pool":         instance_pool.DataSource(),
			"exoscale_instance_pool_list":    instance_pool.DataSourceList(),
			"exoscale_network":               dataSourceNetwork(),
			"exoscale_nlb":                   dataSourceNLB(),
			"exoscale_private_network":       dataSourcePrivateNetwork(),
			"exoscale_security_group":        dataSourceSecurityGroup(),
			"exoscale_template":              dataSourceTemplate(),
			dsSKSClusterIdentifier:           dataSourceSKSCluster(),
			dsSKSClustersListIdentifier:      dataSourceSKSClusterList(),
			dsSKSNodepoolsListIdentifier:     dataSourceSKSNodepoolList(),
			dsSKSNodepoolIdentifier:          dataSourceSKSNodepool(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"exoscale_affinity":             resourceAffinity(),
			"exoscale_anti_affinity_group":  anti_affinity_group.Resource(),
			"exoscale_compute":              resourceCompute(),
			"exoscale_compute_instance":     instance.Resource(),
			"exoscale_domain":               resourceDomain(),
			"exoscale_domain_record":        resourceDomainRecord(),
			"exoscale_elastic_ip":           resourceElasticIP(),
			"exoscale_iam_access_key":       resourceIAMAccessKey(),
			"exoscale_instance_pool":        instance_pool.Resource(),
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

		ConfigureContextFunc: ProviderConfigure,
	}
}

type Configuration struct {
	Key         string
	Secret      string
	Endpoint    string
	DNSEndpoint string
}

func ParseConfig(configFile, key, region string) (*Configuration, error) {
	// Support `~/`
	usr, err := user.Current()
	if err == nil {
		if strings.HasPrefix(configFile, "~/") {
			configFile = filepath.Join(usr.HomeDir, configFile[2:])
		}
	}

	// Convert relative path to absolute
	configFile, _ = filepath.Abs(configFile)
	localConfig, _ := filepath.Abs("cloudstack.ini")

	inis := []string{
		configFile,
		localConfig,
	}

	if usr != nil {
		inis = append(inis, filepath.Join(usr.HomeDir, ".cloudstack.ini"))
	}

	// Stops at the first file that exists
	configFile = ""
	for _, i := range inis {
		if _, err := os.Stat(i); err != nil {
			continue
		}
		configFile = i
		break
	}

	if configFile == "" {
		return nil, fmt.Errorf(
			"key (%s), secret are missing, or config file not found within: %s",
			key,
			strings.Join(inis, ", "),
		)
	}

	cfg, err := ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true}, configFile)
	if err != nil {
		return nil, fmt.Errorf("config file not loaded: %s", err)
	}

	section, err := cfg.GetSection(region)
	if err != nil {
		sections := strings.Join(cfg.SectionStrings(), ", ")
		return nil, fmt.Errorf("%s. Existing sections: %s", err, sections)
	}

	t, err := section.GetKey("key")
	if err != nil {
		return nil, err
	}
	var configuration Configuration

	configuration.Key = t.String()

	s, err := section.GetKey("secret")
	if err != nil {
		return nil, err
	}

	configuration.Secret = s.String()

	e, err := section.GetKey("endpoint")
	if err == nil {
		configuration.Endpoint = e.String()
		configuration.DNSEndpoint = strings.Replace(configuration.Endpoint, "/"+apiVersion, "/dns", 1)
		if strings.Contains(configuration.DNSEndpoint, "/"+legacyAPIVersion) {
			configuration.DNSEndpoint = strings.Replace(configuration.Endpoint, "/"+legacyAPIVersion, "/dns", 1)
		}
	}

	return &configuration, nil
}

func ConvertTimeout(timeout float64) time.Duration {
	return time.Duration(int64(timeout) * int64(time.Second))
}

func CreateClient(baseConfig *providerConfig.BaseConfig) (*exov2.Client, error) {
	return exov2.NewClient(
		baseConfig.Key,
		baseConfig.Secret,
		exov2.ClientOptWithAPIEndpoint(baseConfig.ComputeEndpoint),
		exov2.ClientOptWithTimeout(baseConfig.Timeout),
		exov2.ClientOptWithHTTPClient(func() *http.Client {
			rc := retryablehttp.NewClient()
			rc.Logger = LeveledTFLogger{Verbose: logging.IsDebugOrHigher()}
			hc := rc.StandardClient()
			if logging.IsDebugOrHigher() {
				hc.Transport = logging.NewSubsystemLoggingHTTPTransport("exoscale", hc.Transport)
			}
			return hc
		}()))
}

func ProviderConfigure(_ context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	// we only need to set UserAgent once, so lets do it right away.
	exov2.UserAgent = UserAgent

	key, keyOK := d.GetOk("key")
	if !keyOK {
		key = providerConfig.GetMultiEnvDefault([]string{
			"EXOSCALE_KEY",
			"EXOSCALE_API_KEY",
			"CLOUDSTACK_KEY",
			"CLOUDSTACK_API_KEY",
		}, "")

		if key != "" {
			keyOK = true
		}
	}

	secret, secretOK := d.GetOk("secret")
	if !secretOK {
		secret = providerConfig.GetMultiEnvDefault([]string{
			"EXOSCALE_SECRET",
			"EXOSCALE_SECRET_KEY",
			"EXOSCALE_API_SECRET",
			"CLOUDSTACK_SECRET",
			"CLOUDSTACK_SECRET_KEY",
		}, "")

		if secret != "" {
			secretOK = true
		}
	}

	endpoint, endpointOK := d.GetOk("compute_endpoint")
	if !endpointOK {
		endpoint = providerConfig.GetMultiEnvDefault([]string{
			"EXOSCALE_ENDPOINT",
			"EXOSCALE_API_ENDPOINT",
			"EXOSCALE_COMPUTE_ENDPOINT",
			"CLOUDSTACK_ENDPOINT",
		}, DefaultComputeEndpoint)
	}

	dnsEndpoint, dnsEndpointOK := d.GetOk("dns_endpoint")
	if !dnsEndpointOK {
		dnsEndpoint = providerConfig.GetEnvDefault("EXOSCALE_DNS_ENDPOINT", DefaultDNSEndpoint)
	}

	environment, environmentOK := d.GetOk("environment")
	if !environmentOK {
		environment = providerConfig.GetEnvDefault(
			"EXOSCALE_API_ENVIRONMENT",
			DefaultEnvironment)
	}

	// deprecation support
	token, tokenOK := d.GetOk("token")
	if tokenOK && !keyOK {
		keyOK = tokenOK
		key = token
	}

	var configuration *Configuration

	if keyOK || secretOK {
		if !keyOK || !secretOK {
			return nil, diag.Errorf(
				"key (%#v) and secret (%#v) must be set",
				key.(string),
				secret.(string),
			)
		}
	} else {
		configFile, configFileOK := d.Get("config").(string)
		if !configFileOK {
			configFile = providerConfig.GetMultiEnvDefault(
				[]string{
					"EXOSCALE_CONFIG",
					"CLOUDSTACK_CONFIG",
				}, DefaultConfig)
		}

		region, regionOK := d.GetOk("region")
		if !regionOK {
			region = providerConfig.GetMultiEnvDefault([]string{
				"EXOSCALE_PROFILE",
				"EXOSCALE_REGION",
				"CLOUDSTACK_PROFILE",
				"CLOUDSTACK_REGION",
			}, DefaultProfile)
		}

		// deprecation support
		profile, profileOK := d.GetOk("profile")
		if profileOK && profile.(string) != "" {
			region = profile
		}

		var err error
		configuration, err = ParseConfig(configFile, key.(string), region.(string))
		if err != nil {
			return nil, diag.FromErr(err)
		}
	}

	if configuration != nil {
		if configuration.Endpoint != "" {
			endpoint = configuration.Endpoint
		}

		if configuration.DNSEndpoint != "" {
			dnsEndpoint = configuration.DNSEndpoint
		}

		if configuration.Key != "" {
			key = configuration.Key
		}

		if configuration.Secret != "" {
			secret = configuration.Secret
		}
	}

	var timeout float64
	timeoutRaw, timeoutOk := d.GetOk("timeout")
	if timeoutOk {
		timeout = timeoutRaw.(float64)
	} else {
		var err error
		timeout, err = providerConfig.GetTimeout()

		if err != nil {
			return nil, diag.FromErr(err)
		}
	}

	baseConfig := providerConfig.BaseConfig{
		Key:             key.(string),
		Secret:          secret.(string),
		Timeout:         ConvertTimeout(timeout),
		ComputeEndpoint: endpoint.(string),
		DNSEndpoint:     dnsEndpoint.(string),
		Environment:     environment.(string),
	}

	clv2, err := CreateClient(&baseConfig)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return map[string]interface{}{
			"config":      baseConfig,
			"client":      clv2,
			"environment": environment,
		},
		diags
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
