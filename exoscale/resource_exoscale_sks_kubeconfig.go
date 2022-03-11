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

	"gopkg.in/yaml.v3"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

	var clusterCertificates []*x509.Certificate
	for _, cluster := range kubeconfigData.Clusters {
		parsedCertificate, err := kubeconfigRawPEMDataToCertificate(cluster.Cluster.CertificateAuthorityData)
		if err != nil {
			return nil, nil, fmt.Errorf("unable to read cluster CA certificate: %w", err)
		}

		clusterCertificates = append(clusterCertificates, parsedCertificate)
	}

	var clientCertificates []*x509.Certificate
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
	log.Printf("[DEBUG] kubeconfigToID: kubeconfig= %s", kubeconfig)

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
