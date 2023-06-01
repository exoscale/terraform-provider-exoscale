package exoscale

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	yaml "gopkg.in/yaml.v3"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
)

const (
	resSKSKubeconfigAttrClusterID           = "cluster_id"
	resSKSKubeconfigAttrEarlyRenewalSeconds = "early_renewal_seconds"
	resSKSKubeconfigAttrGroups              = "groups"
	resSKSKubeconfigAttrKubeconfig          = "kubeconfig"
	resSKSKubeconfigAttrReadyForRenewal     = "ready_for_renewal"
	resSKSKubeconfigAttrTTLSeconds          = "ttl_seconds"
	resSKSKubeconfigAttrUser                = "user"
	resSKSKubeconfigAttrZone                = "zone"
)

func resourceSKSKubeconfigIDString(d general.ResourceIDStringer) string {
	return general.ResourceIDString(d, "exoscale_sks_kubeconfig")
}

func resourceSKSKubeconfig() *schema.Resource {
	s := map[string]*schema.Schema{
		resSKSKubeconfigAttrClusterID: {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "The parent [exoscale_sks_cluster](./sks_cluster.md) ID.",
		},
		resSKSKubeconfigAttrEarlyRenewalSeconds: {
			Type:        schema.TypeInt,
			Optional:    true,
			Default:     0,
			Description: "If set, the resource will consider the Kubeconfig to have expired the given number of seconds before its actual CA certificate or client certificate expiry time. This can be useful to deploy an updated Kubeconfig in advance of the expiration of its internal current certificate. Note however that the old certificate remains valid until its true expiration time since this resource does not (and cannot) support revocation. Also note this advance update can only take place if the Terraform configuration is applied during the early renewal period (seconds; default: 0).",
		},
		resSKSKubeconfigAttrGroups: {
			Type:        schema.TypeSet,
			Required:    true,
			ForceNew:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "Group names in the generated Kubeconfig. The certificate present in the Kubeconfig will have these roles set in the Organization field.",
		},
		resSKSKubeconfigAttrKubeconfig: {
			Type:        schema.TypeString,
			Computed:    true,
			Sensitive:   true,
			Description: "The generated Kubeconfig (YAML content).",
		},
		resSKSKubeconfigAttrReadyForRenewal: {
			Type:     schema.TypeBool,
			Computed: true,
		},
		resSKSKubeconfigAttrTTLSeconds: {
			Type:        schema.TypeFloat,
			Optional:    true,
			ForceNew:    true,
			Default:     30 * 24 * 3600,
			Description: "The Time-to-Live of the Kubeconfig, after which it will expire / become invalid (seconds; default: 2592000 = 30 days).",
		},
		resSKSKubeconfigAttrUser: {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "User name in the generated Kubeconfig. The certificate present in the Kubeconfig will also have this name set for the CN field.",
		},
		resSKSKubeconfigAttrZone: {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
		},
	}

	return &schema.Resource{
		Schema: s,

		Description: "Manage Exoscale Scalable Kubernetes Service (SKS) Credentials (Kubeconfig).",

		CreateContext: resourceSKSKubeconfigCreate,
		ReadContext:   resourceSKSKubeconfigRead,
		UpdateContext: resourceSKSKubeconfigUpdate,
		DeleteContext: resourceSKSKubeconfigDelete,

		CustomizeDiff: resourceSKSKubeconfigDiff,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(config.DefaultTimeout),
			Read:   schema.DefaultTimeout(config.DefaultTimeout),
			Update: schema.DefaultTimeout(config.DefaultTimeout),
			Delete: schema.DefaultTimeout(config.DefaultTimeout),
		},
	}
}

func resourceSKSKubeconfigCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning create", map[string]interface{}{
		"id": resourceSKSKubeconfigIDString(d),
	})

	zone := d.Get(resSKSKubeconfigAttrZone).(string)
	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	cluster, err := client.GetSKSCluster(ctx, zone, d.Get(resSKSKubeconfigAttrClusterID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	user := d.Get(resSKSKubeconfigAttrUser).(string)
	groups := []string{}
	if set, ok := d.Get(resSKSKubeconfigAttrGroups).(*schema.Set); ok {
		groups = schemaSetToStringArray(set)
	}

	duration := time.Duration(int64(d.Get(resSKSKubeconfigAttrTTLSeconds).(float64)) * int64(time.Second))

	b64Kubeconfig, err := client.GetSKSClusterKubeconfig(ctx, zone, cluster, user, groups, duration)
	if err != nil {
		return diag.FromErr(err)
	}

	kubeconfig, err := base64.StdEncoding.DecodeString(b64Kubeconfig)
	if err != nil {
		return diag.Errorf("error decoding kubeconfig content: %s", err)
	}

	if err := d.Set(resSKSKubeconfigAttrReadyForRenewal, false); err != nil {
		return diag.Errorf("error setting value on key '%s': %s", resSKSKubeconfigAttrReadyForRenewal, err)
	}
	if err := d.Set(resSKSKubeconfigAttrKubeconfig, string(kubeconfig)); err != nil {
		return diag.Errorf("error setting value on key '%s': %s", resSKSKubeconfigAttrKubeconfig, err)
	}

	id, err := kubeconfigToID(string(kubeconfig))
	if err != nil {
		return diag.Errorf("error generating kubeconfig ID: %s", err)
	}

	d.SetId(*id)

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": resourceSKSKubeconfigIDString(d),
	})

	return resourceSKSKubeconfigRead(ctx, d, meta)
}

func resourceSKSKubeconfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceSKSKubeconfigUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceSKSKubeconfigRead(ctx, d, meta)
}

func resourceSKSKubeconfigDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning delete", map[string]interface{}{
		"id": resourceSKSKubeconfigIDString(d),
	})

	// no revocation: we rely on client certificate expiration
	// So let's just remove the kubeconfig from the state.
	d.SetId("")

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": resourceSKSKubeconfigIDString(d),
	})
	return nil
}

func resourceSKSKubeconfigDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	kubeconfig := d.Get(resSKSKubeconfigAttrKubeconfig).(string)

	clusterCerts, clientCerts, err := KubeconfigExtractCertificates(kubeconfig)
	if err != nil {
		return err
	}

	readyForRenewal := len(kubeconfig) == 0
	if !readyForRenewal {
		now := time.Now()
		earlyRenewalSeconds := d.Get(resSKSKubeconfigAttrEarlyRenewalSeconds).(int)
		earlyRenewalPeriod := time.Duration(-earlyRenewalSeconds) * time.Second

		for _, certificate := range append(clusterCerts, clientCerts...) {
			if certificate.NotAfter.Add(earlyRenewalPeriod).Sub(now) <= 0 {
				readyForRenewal = true
			}
		}
	}

	if readyForRenewal {
		if err := d.SetNew(resSKSKubeconfigAttrReadyForRenewal, true); err != nil {
			return err
		}

		if err := d.ForceNew(resSKSKubeconfigAttrReadyForRenewal); err != nil {
			return err
		}
	}

	return nil
}

func KubeconfigExtractCertificates(kubeconfig string) ([]*x509.Certificate, []*x509.Certificate, error) {
	if len(kubeconfig) == 0 {
		return []*x509.Certificate{}, []*x509.Certificate{}, nil
	}

	var kubeconfigData struct {
		Clusters []struct {
			Cluster struct {
				CertificateAuthorityData string `yaml:"certificate-authority-data"`
			} `yaml:"cluster"`
		} `yaml:"clusters"`
		Users []struct {
			User struct {
				ClientCertificateData string `yaml:"client-certificate-data"`
			} `yaml:"user"`
		} `yaml:"users"`
	}

	if err := yaml.Unmarshal([]byte(kubeconfig), &kubeconfigData); err != nil {
		return nil, nil, fmt.Errorf("error decoding kubeconfig certificates: %w", err)
	}

	clusterCertificates := make([]*x509.Certificate, 0, len(kubeconfigData.Clusters))
	for _, cluster := range kubeconfigData.Clusters {
		parsedCertificate, err := kubeconfigRawPEMDataToCertificate(cluster.Cluster.CertificateAuthorityData)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to read cluster CA certificate: %w", err)
		}

		clusterCertificates = append(clusterCertificates, parsedCertificate)
	}

	clientCertificates := make([]*x509.Certificate, 0, len(kubeconfigData.Users))
	for _, user := range kubeconfigData.Users {
		parsedCertificate, err := kubeconfigRawPEMDataToCertificate(user.User.ClientCertificateData)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to read client certificate: %w", err)
		}

		clientCertificates = append(clientCertificates, parsedCertificate)
	}

	return clusterCertificates, clientCertificates, nil
}

func kubeconfigRawPEMDataToCertificate(b64PEMData string) (*x509.Certificate, error) {
	rawPEMData, err := base64.StdEncoding.DecodeString(b64PEMData)
	if err != nil {
		return nil, fmt.Errorf("error decoding base64 kubeconfig certificate: %w", err)
	}

	parsedPEMData, _ := pem.Decode(rawPEMData)
	parsedCertificate, err := x509.ParseCertificate(parsedPEMData.Bytes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse kubeconfig x509 certificate: %w", err)
	}

	return parsedCertificate, nil
}

func kubeconfigToID(kubeconfig string) (*string, error) {
	tflog.Debug(context.Background(), "kubeconfigToID", map[string]interface{}{
		"kubeconfig": kubeconfig,
	})

	clusterCertificates, clientCertificates, err := KubeconfigExtractCertificates(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("unable to extract certificates from kubeconfig: %w", err)
	}

	certificateIDs := []string{}
	for _, cert := range append(clusterCertificates, clientCertificates...) {
		certificateIDs = append(certificateIDs, cert.SerialNumber.String())
	}

	kubeconfigID := strings.Join(certificateIDs, ":")
	return &kubeconfigID, nil
}
