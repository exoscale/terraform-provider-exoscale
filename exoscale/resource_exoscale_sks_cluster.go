package exoscale

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	defaultSKSClusterCNI          = "calico"
	defaultSKSClusterServiceLevel = "pro"

	sksClusterAddonExoscaleCCM = "exoscale-cloud-controller"
	sksClusterAddonMS          = "metrics-server"

	resSKSClusterAttrAddons             = "addons"
	resSKSClusterAttrAggregationLayerCA = "aggregation_ca"
	resSKSClusterAttrAutoUpgrade        = "auto_upgrade"
	resSKSClusterAttrCNI                = "cni"
	resSKSClusterAttrControlPlaneCA     = "control_plane_ca"
	resSKSClusterAttrCreatedAt          = "created_at"
	resSKSClusterAttrDescription        = "description"
	resSKSClusterAttrEndpoint           = "endpoint"
	resSKSClusterAttrExoscaleCCM        = "exoscale_ccm"
	resSKSClusterAttrKubeletCA          = "kubelet_ca"
	resSKSClusterAttrLabels             = "labels"
	resSKSClusterAttrMetricsServer      = "metrics_server"
	resSKSClusterAttrName               = "name"
	resSKSClusterAttrNodepools          = "nodepools"
	resSKSClusterAttrOIDCClientID       = "client_id"
	resSKSClusterAttrOIDCGroupsClaim    = "groups_claim"
	resSKSClusterAttrOIDCGroupsPrefix   = "groups_prefix"
	resSKSClusterAttrOIDCIssuerURL      = "issuer_url"
	resSKSClusterAttrOIDCRequiredClaim  = "required_claim"
	resSKSClusterAttrOIDCUsernameClaim  = "username_claim"
	resSKSClusterAttrOIDCUsernamePrefix = "username_prefix"
	resSKSClusterAttrServiceLevel       = "service_level"
	resSKSClusterAttrState              = "state"
	resSKSClusterAttrVersion            = "version"
	resSKSClusterAttrZone               = "zone"
)

func resourceSKSClusterIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_sks_cluster")
}

func resourceSKSCluster() *schema.Resource {
	s := map[string]*schema.Schema{
		resSKSClusterAttrAddons: {
			Type:     schema.TypeSet,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Optional: true,
			Computed: true,
			Deprecated: "This attribute has been replaced by `exoscale_ccm`/`metrics_server` " +
				"attributes, it will be removed in a future release.",
		},

		resSKSClusterAttrAggregationLayerCA: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resSKSClusterAttrAutoUpgrade: {
			Type:     schema.TypeBool,
			Optional: true,
		},
		resSKSClusterAttrCNI: {
			Type:     schema.TypeString,
			Optional: true,
			Default:  defaultSKSClusterCNI,
		},
		resSKSClusterAttrControlPlaneCA: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resSKSClusterAttrCreatedAt: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resSKSClusterAttrDescription: {
			Type:     schema.TypeString,
			Optional: true,
		},
		resSKSClusterAttrEndpoint: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resSKSClusterAttrExoscaleCCM: {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  true,
		},
		resSKSClusterAttrKubeletCA: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resSKSClusterAttrMetricsServer: {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  true,
		},
		resSKSClusterAttrLabels: {
			Type:     schema.TypeMap,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Optional: true,
		},
		resSKSClusterAttrName: {
			Type:     schema.TypeString,
			Required: true,
		},
		resSKSClusterAttrNodepools: {
			Type:     schema.TypeSet,
			Computed: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		"oidc": {
			Type:     schema.TypeList,
			MaxItems: 1,
			Optional: true,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					resSKSClusterAttrOIDCClientID: {
						Type:     schema.TypeString,
						Required: true,
					},
					resSKSClusterAttrOIDCGroupsClaim: {
						Type:     schema.TypeString,
						Optional: true,
					},
					resSKSClusterAttrOIDCGroupsPrefix: {
						Type:     schema.TypeString,
						Optional: true,
					},
					resSKSClusterAttrOIDCIssuerURL: {
						Type:     schema.TypeString,
						Required: true,
					},
					resSKSClusterAttrOIDCRequiredClaim: {
						Type:     schema.TypeMap,
						Elem:     &schema.Schema{Type: schema.TypeString},
						Optional: true,
					},
					resSKSClusterAttrOIDCUsernameClaim: {
						Type:     schema.TypeString,
						Optional: true,
					},
					resSKSClusterAttrOIDCUsernamePrefix: {
						Type:     schema.TypeString,
						Optional: true,
					},
				},
			},
		},
		resSKSClusterAttrServiceLevel: {
			Type:     schema.TypeString,
			Optional: true,
			Default:  defaultSKSClusterServiceLevel,
		},
		resSKSClusterAttrState: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resSKSClusterAttrVersion: {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		resSKSClusterAttrZone: {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
	}

	return &schema.Resource{
		Schema: s,

		CreateContext: resourceSKSClusterCreate,
		ReadContext:   resourceSKSClusterRead,
		UpdateContext: resourceSKSClusterUpdate,
		DeleteContext: resourceSKSClusterDelete,

		Importer: &schema.ResourceImporter{
			StateContext: zonedStateContextFunc,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceSKSClusterCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning create", map[string]interface{}{
		"id": resourceSKSClusterIDString(d),
	})

	zone := d.Get(resSKSClusterAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	defer cancel()
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))

	client := GetComputeClient(meta)

	sksCluster := new(egoscale.SKSCluster)

	var addOns []string
	if addonsSet, ok := d.Get(resSKSClusterAttrAddons).(*schema.Set); ok && addonsSet.Len() > 0 {
		addOns = make([]string, addonsSet.Len())
		for i, a := range addonsSet.List() {
			addOns[i] = a.(string)
		}
	}
	if enableCCM := d.Get(resSKSClusterAttrExoscaleCCM).(bool); enableCCM && !in(addOns, sksClusterAddonExoscaleCCM) {
		addOns = append(addOns, sksClusterAddonExoscaleCCM)
	}
	if enableMS := d.Get(resSKSClusterAttrMetricsServer).(bool); enableMS && !in(addOns, sksClusterAddonMS) {
		addOns = append(addOns, sksClusterAddonMS)
	}
	if len(addOns) > 0 {
		sksCluster.AddOns = &addOns
	}

	if autoUpgrade := d.Get(resSKSClusterAttrAutoUpgrade).(bool); autoUpgrade {
		sksCluster.AutoUpgrade = &autoUpgrade
	}

	if v, ok := d.GetOk(resSKSClusterAttrCNI); ok {
		s := v.(string)
		sksCluster.CNI = &s
	}

	if v, ok := d.GetOk(resSKSClusterAttrDescription); ok {
		s := v.(string)
		sksCluster.Description = &s
	}

	if l, ok := d.GetOk(resSKSClusterAttrLabels); ok {
		labels := make(map[string]string)
		for k, v := range l.(map[string]interface{}) {
			labels[k] = v.(string)
		}
		sksCluster.Labels = &labels
	}

	if v, ok := d.GetOk(resSKSClusterAttrName); ok {
		s := v.(string)
		sksCluster.Name = &s
	}

	if v, ok := d.GetOk(resSKSClusterAttrServiceLevel); ok {
		s := v.(string)
		sksCluster.ServiceLevel = &s
	}

	version := d.Get(resSKSClusterAttrVersion).(string)
	if version == "" {
		versions, err := client.ListSKSClusterVersions(ctx)
		if err != nil || len(versions) == 0 {
			if len(versions) == 0 {
				err = errors.New("no version returned by the API")
			}
			return diag.Errorf("error retrieving SKS versions: %s", err)
		}
		version = versions[0]
	}
	sksCluster.Version = &version

	var opts []egoscale.CreateSKSClusterOpt

	if v, ok := d.GetOk(resSKSClusterAttrOIDC(resSKSClusterAttrOIDCClientID)); ok {
		sksClusterOIDCConfig := egoscale.SKSClusterOIDCConfig{ClientID: nonEmptyStringPtr(v.(string))}

		if v, ok := d.GetOk(resSKSClusterAttrOIDC(resSKSClusterAttrOIDCGroupsClaim)); ok {
			sksClusterOIDCConfig.GroupsClaim = nonEmptyStringPtr(v.(string))
		}

		if v, ok := d.GetOk(resSKSClusterAttrOIDC(resSKSClusterAttrOIDCIssuerURL)); ok {
			sksClusterOIDCConfig.IssuerURL = nonEmptyStringPtr(v.(string))
		}

		if v, ok := d.GetOk(resSKSClusterAttrOIDC(resSKSClusterAttrOIDCGroupsPrefix)); ok {
			sksClusterOIDCConfig.GroupsPrefix = nonEmptyStringPtr(v.(string))
		}

		if v, ok := d.GetOk(resSKSClusterAttrOIDC(resSKSClusterAttrOIDCIssuerURL)); ok {
			sksClusterOIDCConfig.IssuerURL = nonEmptyStringPtr(v.(string))
		}

		if c, ok := d.GetOk(resSKSClusterAttrOIDC(resSKSClusterAttrOIDCRequiredClaim)); ok {
			claims := make(map[string]string)
			for k, v := range c.(map[string]interface{}) {
				claims[k] = v.(string)
			}
			sksClusterOIDCConfig.RequiredClaim = &claims
		}

		if v, ok := d.GetOk(resSKSClusterAttrOIDC(resSKSClusterAttrOIDCUsernameClaim)); ok {
			sksClusterOIDCConfig.UsernameClaim = nonEmptyStringPtr(v.(string))
		}

		if v, ok := d.GetOk(resSKSClusterAttrOIDC(resSKSClusterAttrOIDCUsernamePrefix)); ok {
			sksClusterOIDCConfig.UsernamePrefix = nonEmptyStringPtr(v.(string))
		}

		opts = append(opts, egoscale.CreateSKSClusterWithOIDC(&sksClusterOIDCConfig))
	}

	sksCluster, err := client.CreateSKSCluster(ctx, zone, sksCluster, opts...)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*sksCluster.ID)

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": resourceSKSClusterIDString(d),
	})

	return resourceSKSClusterRead(ctx, d, meta)
}

func resourceSKSClusterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceSKSClusterIDString(d),
	})

	zone := d.Get(resSKSClusterAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	sksCluster, err := client.GetSKSCluster(ctx, zone, d.Id())
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceSKSClusterIDString(d),
	})

	certificates, err := readClusterCertificates(client.Client, ctx, zone, sksCluster)
	if err != nil {
		return diag.FromErr(err)
	}

	return diag.FromErr(resourceSKSClusterApply(ctx, d, sksCluster, certificates))
}

func resourceSKSClusterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning update", map[string]interface{}{
		"id": resourceSKSClusterIDString(d),
	})

	zone := d.Get(resSKSClusterAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutUpdate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	sksCluster, err := client.GetSKSCluster(ctx, zone, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var updated bool

	if d.HasChange(resSKSClusterAttrAutoUpgrade) {
		v := d.Get(resSKSClusterAttrAutoUpgrade).(bool)
		sksCluster.AutoUpgrade = &v
		updated = true
	}

	if d.HasChange(resSKSClusterAttrLabels) {
		labels := make(map[string]string)
		for k, v := range d.Get(resSKSClusterAttrLabels).(map[string]interface{}) {
			labels[k] = v.(string)
		}
		sksCluster.Labels = &labels
		updated = true
	}

	if d.HasChange(resSKSClusterAttrName) {
		v := d.Get(resSKSClusterAttrName).(string)
		sksCluster.Name = &v
		updated = true
	}

	if d.HasChange(resSKSClusterAttrDescription) {
		v := d.Get(resSKSClusterAttrDescription).(string)
		sksCluster.Description = &v
		updated = true
	}

	if updated {
		if err = client.UpdateSKSCluster(ctx, zone, sksCluster); err != nil {
			return diag.FromErr(err)
		}
	}

	tflog.Debug(ctx, "update finished successfully", map[string]interface{}{
		"id": resourceSKSClusterIDString(d),
	})

	return resourceSKSClusterRead(ctx, d, meta)
}

func resourceSKSClusterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning delete", map[string]interface{}{
		"id": resourceSKSClusterIDString(d),
	})

	zone := d.Get(resSKSClusterAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	clusterID := d.Id()
	err := client.DeleteSKSCluster(ctx, zone, &egoscale.SKSCluster{ID: &clusterID})
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": resourceSKSClusterIDString(d),
	})

	return nil
}

func resourceSKSClusterApply(_ context.Context, d *schema.ResourceData, sksCluster *egoscale.SKSCluster, certificates *SKSClusterCertificates) error {
	if sksCluster.AddOns != nil {
		if err := d.Set(resSKSClusterAttrAddons, *sksCluster.AddOns); err != nil {
			return err
		}

		if err := d.Set(resSKSClusterAttrExoscaleCCM, in(*sksCluster.AddOns, sksClusterAddonExoscaleCCM)); err != nil {
			return err
		}

		if err := d.Set(resSKSClusterAttrMetricsServer, in(*sksCluster.AddOns, sksClusterAddonMS)); err != nil {
			return err
		}
	}

	if err := d.Set(resSKSClusterAttrAggregationLayerCA, certificates.AggregationCA); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrAutoUpgrade, defaultBool(sksCluster.AutoUpgrade, false)); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrCNI, defaultString(sksCluster.CNI, "")); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrControlPlaneCA, certificates.ControlPlaneCA); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrCreatedAt, sksCluster.CreatedAt.String()); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrDescription, defaultString(sksCluster.Description, "")); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrEndpoint, *sksCluster.Endpoint); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrKubeletCA, certificates.KubeletCA); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrLabels, sksCluster.Labels); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrName, *sksCluster.Name); err != nil {
		return err
	}

	nodepools := make([]string, len(sksCluster.Nodepools))
	for i, nodepool := range sksCluster.Nodepools {
		nodepools[i] = *nodepool.ID
	}
	if err := d.Set(resSKSClusterAttrNodepools, nodepools); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrServiceLevel, *sksCluster.ServiceLevel); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrState, *sksCluster.State); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrVersion, *sksCluster.Version); err != nil {
		return err
	}

	return nil
}

// resSKSClusterAttrOIDC returns a sks_cluster resource attribute key formatted for an "oidc {}" block.
func resSKSClusterAttrOIDC(a string) string {
	return fmt.Sprintf("oidc.0.%s", a)
}

type SKSClusterCertificates struct {
	AggregationCA  string
	ControlPlaneCA string
	KubeletCA      string
}

// readClusterCertificates returns an SKS Cluster related CA certificates
func readClusterCertificates(client *egoscale.Client, ctx context.Context, zone string, cluster *egoscale.SKSCluster) (*SKSClusterCertificates, error) {
	encodedAggregationCertificate, err := client.GetSKSClusterAuthorityCert(ctx, zone, cluster, "aggregation")
	if err != nil {
		return nil, err
	}

	encodedControlPlaneCertificate, err := client.GetSKSClusterAuthorityCert(ctx, zone, cluster, "control-plane")
	if err != nil {
		return nil, err
	}

	encodedKubeletCertificate, err := client.GetSKSClusterAuthorityCert(ctx, zone, cluster, "kubelet")
	if err != nil {
		return nil, err
	}

	aggregationCertificate, err := base64.StdEncoding.DecodeString(encodedAggregationCertificate)
	if err != nil {
		return nil, err
	}

	controlPlaneCertificate, err := base64.StdEncoding.DecodeString(encodedControlPlaneCertificate)
	if err != nil {
		return nil, err
	}

	kubeletCertificate, err := base64.StdEncoding.DecodeString(encodedKubeletCertificate)
	if err != nil {
		return nil, err
	}

	return &SKSClusterCertificates{
		AggregationCA:  string(aggregationCertificate),
		ControlPlaneCA: string(controlPlaneCertificate),
		KubeletCA:      string(kubeletCertificate),
	}, nil
}
