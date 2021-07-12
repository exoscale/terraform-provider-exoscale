package exoscale

import (
	"context"
	"errors"
	"log"

	exov2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	defaultSKSClusterCNI          = "calico"
	defaultSKSClusterServiceLevel = "pro"

	sksClusterAddonExoscaleCCM = "exoscale-cloud-controller"
	sksClusterAddonMS          = "metrics-server"

	resSKSClusterAttrAddons        = "addons"
	resSKSClusterAttrAutoUpgrade   = "auto_upgrade"
	resSKSClusterAttrCNI           = "cni"
	resSKSClusterAttrCreatedAt     = "created_at"
	resSKSClusterAttrDescription   = "description"
	resSKSClusterAttrEndpoint      = "endpoint"
	resSKSClusterAttrExoscaleCCM   = "exoscale_ccm"
	resSKSClusterAttrMetricsServer = "metrics_server"
	resSKSClusterAttrName          = "name"
	resSKSClusterAttrNodepools     = "nodepools"
	resSKSClusterAttrServiceLevel  = "service_level"
	resSKSClusterAttrState         = "state"
	resSKSClusterAttrVersion       = "version"
	resSKSClusterAttrZone          = "zone"
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

	sksCluster := new(exov2.SKSCluster)

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

	sksCluster, err := client.CreateSKSCluster(ctx, zone, sksCluster)
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

	log.Printf("[DEBUG] %s: read finished successfully", resourceSKSClusterIDString(d))

	return resourceSKSClusterApply(ctx, d, sksCluster)
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

	err := client.DeleteSKSCluster(ctx, zone, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceSKSClusterIDString(d))

	return nil
}

func resourceSKSClusterApply(_ context.Context, d *schema.ResourceData, sksCluster *exov2.SKSCluster) diag.Diagnostics {
	if sksCluster.AddOns != nil {
		if err := d.Set(resSKSClusterAttrAddons, *sksCluster.AddOns); err != nil {
			return diag.FromErr(err)
		}

		if err := d.Set(resSKSClusterAttrExoscaleCCM, in(*sksCluster.AddOns, sksClusterAddonExoscaleCCM)); err != nil {
			return diag.FromErr(err)
		}

		if err := d.Set(resSKSClusterAttrMetricsServer, in(*sksCluster.AddOns, sksClusterAddonMS)); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(resSKSClusterAttrAutoUpgrade, defaultBool(sksCluster.AutoUpgrade, false)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resSKSClusterAttrCNI, defaultString(sksCluster.CNI, "")); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resSKSClusterAttrCreatedAt, sksCluster.CreatedAt.String()); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resSKSClusterAttrDescription, defaultString(sksCluster.Description, "")); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resSKSClusterAttrEndpoint, *sksCluster.Endpoint); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resSKSClusterAttrName, *sksCluster.Name); err != nil {
		return diag.FromErr(err)
	}

	nodepools := make([]string, len(sksCluster.Nodepools))
	for i, nodepool := range sksCluster.Nodepools {
		nodepools[i] = *nodepool.ID
	}
	if err := d.Set(resSKSClusterAttrNodepools, nodepools); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resSKSClusterAttrServiceLevel, *sksCluster.ServiceLevel); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resSKSClusterAttrState, *sksCluster.State); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resSKSClusterAttrVersion, *sksCluster.Version); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
