package exoscale

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/logging"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/resources/anti_affinity_group"
	"github.com/exoscale/terraform-provider-exoscale/pkg/resources/instance"
	"github.com/exoscale/terraform-provider-exoscale/pkg/resources/instance_pool"

	exov2 "github.com/exoscale/egoscale/v2"
	exov3 "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/egoscale/v3/credentials"
	providerConfig "github.com/exoscale/terraform-provider-exoscale/pkg/provider/config"
)

const (
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
			"secret": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "Exoscale API secret",
			},
			"environment": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"sos_endpoint": {
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
			"exoscale_anti_affinity_group":   anti_affinity_group.DataSource(),
			"exoscale_compute_instance":      instance.DataSource(),
			"exoscale_compute_instance_list": instance.DataSourceList(),
			"exoscale_domain":                dataSourceDomain(),
			"exoscale_domain_record":         dataSourceDomainRecord(),
			"exoscale_elastic_ip":            dataSourceElasticIP(),
			"exoscale_instance_pool":         instance_pool.DataSource(),
			"exoscale_instance_pool_list":    instance_pool.DataSourceList(),
			"exoscale_nlb":                   dataSourceNLB(),
			"exoscale_private_network":       dataSourcePrivateNetwork(),
			"exoscale_template":              dataSourceTemplate(),
			dsSKSClusterIdentifier:           dataSourceSKSCluster(),
			dsSKSClustersListIdentifier:      dataSourceSKSClusterList(),
			dsSKSNodepoolsListIdentifier:     dataSourceSKSNodepoolList(),
			dsSKSNodepoolIdentifier:          dataSourceSKSNodepool(),
		},

		ResourcesMap: map[string]*schema.Resource{
			"exoscale_anti_affinity_group": anti_affinity_group.Resource(),
			"exoscale_compute_instance":    instance.Resource(),
			"exoscale_domain":              resourceDomain(),
			"exoscale_domain_record":       resourceDomainRecord(),
			"exoscale_elastic_ip":          resourceElasticIP(),
			"exoscale_iam_access_key":      resourceIAMAccessKey(),
			"exoscale_instance_pool":       instance_pool.Resource(),
			"exoscale_nlb":                 resourceNLB(),
			"exoscale_nlb_service":         resourceNLBService(),
			"exoscale_private_network":     resourcePrivateNetwork(),
			"exoscale_security_group_rule": resourceSecurityGroupRule(),
			"exoscale_sks_cluster":         resourceSKSCluster(),
			"exoscale_sks_kubeconfig":      resourceSKSKubeconfig(),
			"exoscale_sks_nodepool":        resourceSKSNodepool(),
			"exoscale_ssh_key":             resourceSSHKey(),
		},

		ConfigureContextFunc: ProviderConfigure,
	}
}

func ConvertTimeout(timeout float64) time.Duration {
	return time.Duration(int64(timeout) * int64(time.Second))
}

func CreateClient(baseConfig *providerConfig.BaseConfig) (*exov2.Client, error) {
	return exov2.NewClient(
		baseConfig.Key,
		baseConfig.Secret,
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

	environment, environmentOK := d.GetOk("environment")
	if !environmentOK {
		environment = providerConfig.GetEnvDefault(
			"EXOSCALE_API_ENVIRONMENT",
			DefaultEnvironment)
	}

	sosEndpoint, sosEndpointOK := d.GetOk("sos_endpoint")
	if !sosEndpointOK {
		sosEndpoint = providerConfig.GetEnvDefault(
			"EXOSCALE_SOS_ENDPOINT",
			providerConfig.GetEnvDefault("EXOSCALE_STORAGE_API_ENDPOINT", ""))
	}

	if keyOK || secretOK {
		if !keyOK || !secretOK {
			return nil, diag.Errorf(
				"key (%#v) and secret (%#v) must be set",
				key.(string),
				secret.(string),
			)
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
		Key:         key.(string),
		Secret:      secret.(string),
		Timeout:     ConvertTimeout(timeout),
		Environment: environment.(string),
		SOSEndpoint: sosEndpoint.(string),
	}

	clv2, err := CreateClient(&baseConfig)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	// Exoscale v3 client
	creds := credentials.NewStaticCredentials(
		key.(string),
		secret.(string),
	)

	opts := []exov3.ClientOpt{}
	if ep := os.Getenv("EXOSCALE_API_ENDPOINT"); ep != "" {
		opts = append(opts, exov3.ClientOptWithEndpoint(exov3.Endpoint(ep)), exov3.ClientOptWithUserAgent(UserAgent))
	}

	clv3, err := exov3.NewClient(creds, opts...)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return map[string]interface{}{
			"config":       baseConfig,
			"client":       clv2,
			"clientV3":     clv3,
			"environment":  environment,
			"sos_endpoint": sosEndpoint,
		},
		diags
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
