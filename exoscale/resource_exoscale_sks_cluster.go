package exoscale

import (
	"context"
	"errors"
	"fmt"
	"log"

	exov2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	defaultSKSClusterCNI          = "calico"
	defaultSKSClusterServiceLevel = "pro"

	sksClusterAddonExoscaleCCM = "exoscale-cloud-controller"
	sksClusterAddonMS          = "metrics-server"
)

func resourceSKSClusterIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_sks_cluster")
}

func resourceSKSCluster() *schema.Resource {
	s := map[string]*schema.Schema{
		"addons": {
			Type:     schema.TypeSet,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Optional: true,
			Computed: true,
			Deprecated: "This attribute has been replaced by `exoscale_ccm`/`metrics_server` " +
				"attributes, it will be removed in a future release.",
		},
		"cni": {
			Type:     schema.TypeString,
			Optional: true,
			Default:  defaultSKSClusterCNI,
		},
		"created_at": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"description": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"endpoint": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"exoscale_ccm": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  true,
		},
		"metrics_server": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  true,
		},
		"name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"nodepools": {
			Type:     schema.TypeSet,
			Computed: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		"service_level": {
			Type:     schema.TypeString,
			Optional: true,
			Default:  defaultSKSClusterServiceLevel,
		},
		"state": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"version": {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		"zone": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
	}

	return &schema.Resource{
		Schema: s,

		Create: resourceSKSClusterCreate,
		Read:   resourceSKSClusterRead,
		Update: resourceSKSClusterUpdate,
		Delete: resourceSKSClusterDelete,
		Exists: resourceSKSClusterExists,

		Importer: &schema.ResourceImporter{
			StateContext: resourceSKSClusterImport,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceSKSClusterCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning create", resourceSKSClusterIDString(d))

	zone := d.Get("zone").(string)

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))

	client := GetComputeClient(meta)

	var addOns []string
	if addonsSet, ok := d.Get("addons").(*schema.Set); ok && addonsSet.Len() > 0 {
		addOns = make([]string, addonsSet.Len())
		for i, a := range addonsSet.List() {
			addOns[i] = a.(string)
		}
	}

	if enableCCM := d.Get("exoscale_ccm").(bool); enableCCM && !in(addOns, sksClusterAddonExoscaleCCM) {
		addOns = append(addOns, sksClusterAddonExoscaleCCM)
	}

	if enableMS := d.Get("metrics_server").(bool); enableMS && !in(addOns, sksClusterAddonMS) {
		addOns = append(addOns, sksClusterAddonMS)
	}

	version := d.Get("version").(string)
	if version == "" {
		versions, err := client.ListSKSClusterVersions(ctx)
		if err != nil || len(versions) == 0 {
			if len(versions) == 0 {
				err = errors.New("no version returned by the API")
			}
			return fmt.Errorf("unable to retrieve SKS versions: %s", err)
		}
		version = versions[0]
	}

	cluster, err := client.CreateSKSCluster(
		ctx,
		zone,
		&exov2.SKSCluster{
			Name:         d.Get("name").(string),
			Description:  d.Get("description").(string),
			Version:      version,
			ServiceLevel: d.Get("service_level").(string),
			CNI:          d.Get("cni").(string),
			AddOns:       addOns,
		},
	)
	if err != nil {
		return err
	}

	d.SetId(cluster.ID)

	log.Printf("[DEBUG] %s: create finished successfully", resourceSKSClusterIDString(d))

	return resourceSKSClusterRead(d, meta)
}

func resourceSKSClusterRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning read", resourceSKSClusterIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	cluster, err := findSKSCluster(ctx, d, meta)
	if err != nil {
		return err
	}

	if cluster == nil {
		return fmt.Errorf("SKS cluster %q not found", d.Id())
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceSKSClusterIDString(d))

	return resourceSKSClusterApply(d, cluster)
}

func resourceSKSClusterExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	cluster, err := findSKSCluster(ctx, d, meta)
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			return false, nil
		}
		return false, err
	}

	if cluster == nil {
		return false, nil
	}

	return true, nil
}

func resourceSKSClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning update", resourceSKSClusterIDString(d))

	zone := d.Get("zone").(string)

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutUpdate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	cluster, err := findSKSCluster(ctx, d, meta)
	if err != nil {
		return err
	}

	if cluster == nil {
		return fmt.Errorf("SKS cluster %q not found", d.Id())
	}

	var (
		updated     bool
		resetFields = make([]interface{}, 0)
	)

	if d.HasChange("name") {
		cluster.Name = d.Get("name").(string)
		updated = true
	}

	if d.HasChange("description") {
		if v := d.Get("description").(string); v == "" {
			cluster.Description = ""
			resetFields = append(resetFields, &cluster.Description)
		} else {
			cluster.Description = d.Get("description").(string)
			updated = true
		}
	}

	if updated {
		if err = client.UpdateSKSCluster(ctx, zone, cluster); err != nil {
			return err
		}
	}

	for _, f := range resetFields {
		if err = cluster.ResetField(ctx, f); err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] %s: update finished successfully", resourceSKSClusterIDString(d))

	return resourceSKSClusterRead(d, meta)
}

func resourceSKSClusterDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning delete", resourceSKSClusterIDString(d))

	zone := d.Get("zone").(string)

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	err := client.DeleteSKSCluster(ctx, zone, d.Id())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceSKSClusterIDString(d))

	return nil
}

func resourceSKSClusterImport(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	log.Printf("[DEBUG] %s: beginning import", resourceSKSClusterIDString(d))

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancel()

	cluster, err := findSKSCluster(ctx, d, meta)
	if err != nil {
		return nil, err
	}

	if cluster == nil {
		return nil, fmt.Errorf("SKS cluster %q not found", d.Id())
	}

	if err := resourceSKSClusterApply(d, cluster); err != nil {
		return nil, err
	}

	resources := []*schema.ResourceData{d}
	for _, nodepool := range cluster.Nodepools {
		resource := resourceSKSNodepool()
		d := resource.Data(nil)
		d.SetType("exoscale_sks_nodepool")
		d.SetId(nodepool.ID)
		err := resourceSKSNodepoolApply(d, meta, nodepool)
		if err != nil {
			return nil, err
		}

		resources = append(resources, d)
	}

	log.Printf("[DEBUG] %s: import finished successfully", resourceSKSClusterIDString(d))

	return resources, nil
}

func resourceSKSClusterApply(d *schema.ResourceData, cluster *exov2.SKSCluster) error {
	if err := d.Set("addons", cluster.AddOns); err != nil {
		return err
	}

	if err := d.Set("cni", cluster.CNI); err != nil {
		return err
	}

	if err := d.Set("created_at", cluster.CreatedAt.String()); err != nil {
		return err
	}

	if err := d.Set("description", cluster.Description); err != nil {
		return err
	}

	if err := d.Set("endpoint", cluster.Endpoint); err != nil {
		return err
	}

	if err := d.Set("exoscale_ccm", in(cluster.AddOns, sksClusterAddonExoscaleCCM)); err != nil {
		return err
	}

	if err := d.Set("metrics_server", in(cluster.AddOns, sksClusterAddonMS)); err != nil {
		return err
	}

	if err := d.Set("name", cluster.Name); err != nil {
		return err
	}

	nodepools := make([]string, len(cluster.Nodepools))
	for i, nodepool := range cluster.Nodepools {
		nodepools[i] = nodepool.ID
	}
	if err := d.Set("nodepools", nodepools); err != nil {
		return err
	}

	if err := d.Set("service_level", cluster.ServiceLevel); err != nil {
		return err
	}

	if err := d.Set("state", cluster.State); err != nil {
		return err
	}

	if err := d.Set("version", cluster.Version); err != nil {
		return err
	}

	return nil
}

func findSKSCluster(ctx context.Context, d *schema.ResourceData, meta interface{}) (*exov2.SKSCluster, error) {
	client := GetComputeClient(meta)

	if zone, ok := d.GetOk("zone"); ok {
		ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone.(string)))
		cluster, err := client.GetSKSCluster(ctx, zone.(string), d.Id())
		if err != nil {
			return nil, err
		}

		return cluster, nil
	}

	zones, err := client.ListZones(exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), defaultZone)))
	if err != nil {
		return nil, err
	}

	var cluster *exov2.SKSCluster
	for _, zone := range zones {
		c, err := client.GetSKSCluster(
			exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone)),
			zone,
			d.Id())
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				continue
			}

			return nil, err
		}

		if c != nil {
			cluster = c
			if err := d.Set("zone", zone); err != nil {
				return nil, err
			}
			break
		}
	}

	return cluster, nil
}
