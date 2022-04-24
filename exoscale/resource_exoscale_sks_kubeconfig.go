package exoscale

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"strings"
	"time"

	"k8s.io/client-go/tools/clientcmd"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	resSKSKubeconfigAttrClusterID           = "cluster_id"
	resSKSKubeconfigAttrEarlyRenewalSeconds = "early_renewal_seconds"
	resSKSKubeconfigAttrGroups              = "groups"
	resSKSKubeconfigAttrKubeconfig          = "kubeconfig"
	resSKSKubeconfigAttrCACertificate       = "ca_certificate"
	resSKSKubeconfigAttrClientCertificate   = "client_certificate"
	resSKSKubeconfigAttrClientKey           = "client_key"
	resSKSKubeconfigAttrReadyForRenewal     = "ready_for_renewal"
	resSKSKubeconfigAttrTTLSeconds          = "ttl_seconds"
	resSKSKubeconfigAttrUser                = "user"
	resSKSKubeconfigAttrZone                = "zone"
)

func resourceSKSKubeconfigIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_sks_kubeconfig")
}

func resourceSKSKubeconfig() *schema.Resource {
	s := map[string]*schema.Schema{
		resSKSKubeconfigAttrClusterID: {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		resSKSKubeconfigAttrEarlyRenewalSeconds: {
			Type:     schema.TypeInt,
			Optional: true,
			Default:  0,
		},
		resSKSKubeconfigAttrGroups: {
			Type:     schema.TypeSet,
			Required: true,
			ForceNew: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		resSKSKubeconfigAttrKubeconfig: {
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
		resSKSKubeconfigAttrCACertificate: {
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
		resSKSKubeconfigAttrClientCertificate: {
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
		resSKSKubeconfigAttrClientKey: {
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
		resSKSKubeconfigAttrReadyForRenewal: {
			Type:     schema.TypeBool,
			Computed: true,
		},
		resSKSKubeconfigAttrTTLSeconds: {
			Type:     schema.TypeFloat,
			Optional: true,
			ForceNew: true,
			Default:  30 * 24 * 3600,
		},
		resSKSKubeconfigAttrUser: {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		resSKSKubeconfigAttrZone: {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
	}

	return &schema.Resource{
		Schema: s,

		CreateContext: resourceSKSKubeconfigCreate,
		ReadContext:   resourceSKSKubeconfigRead,
		UpdateContext: resourceSKSKubeconfigUpdate,
		DeleteContext: resourceSKSKubeconfigDelete,

		CustomizeDiff: resourceSKSKubeconfigDiff,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceSKSKubeconfigCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning create", resourceSKSKubeconfigIDString(d))

	zone := d.Get(resSKSKubeconfigAttrZone).(string)
	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	clusterId := d.Get(resSKSKubeconfigAttrClusterID).(string)

	cluster, err := client.GetSKSCluster(ctx, zone, clusterId)
	if err != nil {
		return diag.FromErr(err)
	}

	user := d.Get(resSKSKubeconfigAttrUser).(string)
	var groups []string
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

	kubeconfigData, err := clientcmd.Load(kubeconfig)
	if err != nil {
		return diag.Errorf("error loading kubeconfig: %s", err)
	}

	clusterConfig := kubeconfigData.Clusters[clusterId]
	certificateAuthorityData := clusterConfig.CertificateAuthorityData
	userAuthInfo := kubeconfigData.AuthInfos[user]
	clientCertificateData := userAuthInfo.ClientCertificateData
	clientKeyData := userAuthInfo.ClientKeyData

	if err := d.Set(resSKSKubeconfigAttrCACertificate, string(certificateAuthorityData)); err != nil {
		return diag.Errorf("error setting value on key '%s': %s", resSKSKubeconfigAttrCACertificate, err)
	}
	if err := d.Set(resSKSKubeconfigAttrClientCertificate, string(clientCertificateData)); err != nil {
		return diag.Errorf("error setting value on key '%s': %s", resSKSKubeconfigAttrCACertificate, err)
	}
	if err := d.Set(resSKSKubeconfigAttrClientKey, string(clientKeyData)); err != nil {
		return diag.Errorf("error setting value on key '%s': %s", resSKSKubeconfigAttrCACertificate, err)
	}

	id, err := certificatesToId(certificateAuthorityData, clientCertificateData)
	if err != nil {
		return diag.Errorf("error generating ID: %s", err)
	}

	d.SetId(*id)

	log.Printf("[DEBUG] %s: create finished successfully", resourceSKSKubeconfigIDString(d))

	return resourceSKSKubeconfigRead(ctx, d, meta)
}

func resourceSKSKubeconfigRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return nil
}

func resourceSKSKubeconfigUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return resourceSKSKubeconfigRead(ctx, d, meta)
}

func resourceSKSKubeconfigDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning delete", resourceSKSKubeconfigIDString(d))

	// no revocation: we rely on client certificate expiration
	// So let's just remove the kubeconfig from the state.
	d.SetId("")

	log.Printf("[DEBUG] %s: delete finished successfully", resourceSKSKubeconfigIDString(d))
	return nil
}

func resourceSKSKubeconfigDiff(ctx context.Context, d *schema.ResourceDiff, meta interface{}) error {
	caCertificate := d.Get(resSKSKubeconfigAttrCACertificate).(string)
	clientCertificate := d.Get(resSKSKubeconfigAttrClientCertificate).(string)

	readyForRenewal := len(caCertificate) == 0 || len(clientCertificate) == 0

	if !readyForRenewal {
		parsedCACertificate, err := rawPEMDataToCertificate([]byte(caCertificate))
		if err != nil {
			return fmt.Errorf("unable to read cluster CA certificate: %w", err)
		}

		parsedClientCertificate, err := rawPEMDataToCertificate([]byte(clientCertificate))
		if err != nil {
			return fmt.Errorf("unable to read client certificate: %w", err)
		}

		certificates := []*x509.Certificate{parsedCACertificate, parsedClientCertificate}

		now := time.Now()
		earlyRenewalSeconds := d.Get(resSKSKubeconfigAttrEarlyRenewalSeconds).(int)
		earlyRenewalPeriod := time.Duration(-earlyRenewalSeconds) * time.Second

		for _, certificate := range certificates {
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

func rawPEMDataToCertificate(rawPEMData []byte) (*x509.Certificate, error) {
	parsedPEMData, _ := pem.Decode(rawPEMData)
	parsedCertificate, err := x509.ParseCertificate(parsedPEMData.Bytes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse x509 certificate: %w", err)
	}

	return parsedCertificate, nil
}

func certificatesToId(certificates ...[]byte) (*string, error) {
	log.Printf("[DEBUG] certificatesToId: certificates = %s", certificates)

	var certificateIDs []string
	for _, certificate := range certificates {
		parsedCertificate, err := rawPEMDataToCertificate(certificate)
		if err != nil {
			return nil, fmt.Errorf("unable to read certificate: %s", err)
		}

		certificateIDs = append(certificateIDs, parsedCertificate.SerialNumber.String())
	}

	id := strings.Join(certificateIDs, ":")
	return &id, nil
}
