package exoscale

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	defaultSKSClusterCNI          = "calico"
	defaultSKSClusterServiceLevel = "pro"

	sksClusterAddonExoscaleCCM = "exoscale-cloud-controller"
	sksClusterAddonMS          = "metrics-server"

	resSKSClusterAttrAddons             = "addons"
	resSKSClusterAttrAutoUpgrade        = "auto_upgrade"
	resSKSClusterAttrCNI                = "cni"
	resSKSClusterAttrCreatedAt          = "created_at"
	resSKSClusterAttrDescription        = "description"
	resSKSClusterAttrEndpoint           = "endpoint"
	resSKSClusterAttrKubeconfig         = "kubeconfig"
	resSKSClusterAttrExoscaleCCM        = "exoscale_ccm"
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
		resSKSClusterAttrAutoUpgrade: {
			Type:     schema.TypeBool,
			Optional: true,
		},
		resSKSClusterAttrCNI: {
			Type:     schema.TypeString,
			Optional: true,
			Default:  defaultSKSClusterCNI,
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
		resSKSClusterAttrKubeconfig: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resSKSClusterAttrExoscaleCCM: {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  true,
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
	log.Printf("[DEBUG] %s: beginning create", resourceSKSClusterIDString(d))

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

	log.Printf("[DEBUG] %s: create finished successfully", resourceSKSClusterIDString(d))

	return resourceSKSClusterRead(ctx, d, meta)
}

func resourceSKSClusterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning read", resourceSKSClusterIDString(d))

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

	sksKubeconfig, err := client.GetSKSClusterKubeconfig(
		ctx,
		zone,
		sksCluster,
		"kube-admin",
		[]string{"system:masters"},
		30*24*time.Hour,
	)

	log.Printf("[DEBUG] %s: read finished successfully", resourceSKSClusterIDString(d))

	return diag.FromErr(resourceSKSClusterApply(ctx, d, sksCluster, sksKubeconfig))
}

func resourceSKSClusterUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning update", resourceSKSClusterIDString(d))

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

	log.Printf("[DEBUG] %s: update finished successfully", resourceSKSClusterIDString(d))

	return resourceSKSClusterRead(ctx, d, meta)
}

func resourceSKSClusterDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning delete", resourceSKSClusterIDString(d))

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

	log.Printf("[DEBUG] %s: delete finished successfully", resourceSKSClusterIDString(d))

	return nil
}

func resourceSKSClusterApply(_ context.Context, d *schema.ResourceData, sksCluster *egoscale.SKSCluster, sksKubeconfig string) error {
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

	if err := d.Set(resSKSClusterAttrAutoUpgrade, defaultBool(sksCluster.AutoUpgrade, false)); err != nil {
		return err
	}

	if err := d.Set(resSKSClusterAttrCNI, defaultString(sksCluster.CNI, "")); err != nil {
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

	if err := d.Set(resSKSClusterAttrKubeconfig, sksKubeconfig); err != nil {
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
func resSKSClusterAttrOIDC(a string) string { return fmt.Sprintf("oidc.0.%s", a) }
