package exoscale

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/exoscale/egoscale"
	exov2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

const (
	defaultSKSNodepoolDiskSize = 50
)

func resourceSKSNodepoolIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_sks_nodepool")
}

func resourceSKSNodepool() *schema.Resource {
	s := map[string]*schema.Schema{
		"cluster_id": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"created_at": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"description": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"disk_size": {
			Type:     schema.TypeInt,
			Optional: true,
			Default:  defaultSKSNodepoolDiskSize,
		},
		"instance_pool_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"instance_type": {
			Type:     schema.TypeString,
			Required: true,
			ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
				v := val.(string)
				if strings.ContainsAny(v, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
					errs = append(errs, fmt.Errorf("%q must be lowercase, got: %q", key, v))
				}
				return
			},
		},
		"name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"anti_affinity_group_ids": {
			Type:     schema.TypeSet,
			Optional: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		"security_group_ids": {
			Type:     schema.TypeSet,
			Optional: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		"size": {
			Type:     schema.TypeInt,
			Required: true,
		},
		"state": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"template_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"version": {
			Type:     schema.TypeString,
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

		Create: resourceSKSNodepoolCreate,
		Read:   resourceSKSNodepoolRead,
		Update: resourceSKSNodepoolUpdate,
		Delete: resourceSKSNodepoolDelete,
		Exists: resourceSKSNodepoolExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceSKSNodepoolCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning create", resourceSKSNodepoolIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()
	client := GetComputeClient(meta)

	zone := d.Get("zone").(string)

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	cluster, err := client.GetSKSCluster(ctx, zone, d.Get("cluster_id").(string))
	if err != nil {
		return err
	}

	resp, err := client.GetWithContext(ctx, &egoscale.ServiceOffering{
		Name: d.Get("instance_type").(string),
	})
	if err != nil {
		return err
	}
	instanceType := resp.(*egoscale.ServiceOffering)

	var antiAffinityGroupIDs []string
	if antiAffinityIDSet, ok := d.Get("anti_affinity_group_ids").(*schema.Set); ok {
		antiAffinityGroupIDs = make([]string, antiAffinityIDSet.Len())
		for i, id := range antiAffinityIDSet.List() {
			antiAffinityGroupIDs[i] = id.(string)
		}
	}

	var securityGroupIDs []string
	if securityIDSet, ok := d.Get("security_group_ids").(*schema.Set); ok {
		securityGroupIDs = make([]string, securityIDSet.Len())
		for i, id := range securityIDSet.List() {
			securityGroupIDs[i] = id.(string)
		}
	}

	nodepool, err := cluster.AddNodepool(
		ctx,
		&exov2.SKSNodepool{
			Name:                 d.Get("name").(string),
			Description:          d.Get("description").(string),
			InstanceTypeID:       instanceType.ID.String(),
			DiskSize:             int64(d.Get("disk_size").(int)),
			AntiAffinityGroupIDs: antiAffinityGroupIDs,
			SecurityGroupIDs:     securityGroupIDs,
			Size:                 int64(d.Get("size").(int)),
		},
	)
	if err != nil {
		return err
	}

	d.SetId(nodepool.ID)

	log.Printf("[DEBUG] %s: create finished successfully", resourceSKSNodepoolIDString(d))

	return resourceSKSNodepoolRead(d, meta)
}

func resourceSKSNodepoolRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning read", resourceSKSNodepoolIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	nodepool, err := findSKSNodepool(ctx, d, meta)
	if err != nil {
		return err
	}

	if nodepool == nil {
		return fmt.Errorf("SKS Nodepool %q not found", d.Id())
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceSKSNodepoolIDString(d))

	return resourceSKSNodepoolApply(d, meta, nodepool)
}

func resourceSKSNodepoolExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	nodepool, err := findSKSNodepool(ctx, d, meta)
	if err != nil {
		return false, err
	}

	if nodepool == nil {
		return false, nil
	}

	return true, nil
}

func resourceSKSNodepoolUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning update", resourceSKSNodepoolIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()
	client := GetComputeClient(meta)

	zone := d.Get("zone").(string)

	nodepool, err := findSKSNodepool(ctx, d, meta)
	if err != nil {
		return err
	}

	if nodepool == nil {
		return fmt.Errorf("SKS Nodepool %q not found", d.Id())
	}

	var (
		updated     bool
		resetFields = make([]interface{}, 0)
	)

	if d.HasChange("name") {
		nodepool.Name = d.Get("name").(string)
		updated = true
	}

	if d.HasChange("description") {
		if v := d.Get("description").(string); v == "" {
			nodepool.Description = ""
			resetFields = append(resetFields, &nodepool.Description)
		} else {
			nodepool.Description = d.Get("description").(string)
			updated = true
		}
	}

	if d.HasChange("disk_size") {
		nodepool.DiskSize = int64(d.Get("disk_size").(int))
		updated = true
	}

	if d.HasChange("instance_type") {
		resp, err := client.GetWithContext(ctx, &egoscale.ServiceOffering{
			Name: d.Get("instance_type").(string),
		})
		if err != nil {
			return err
		}
		nodepool.InstanceTypeID = resp.(*egoscale.ServiceOffering).ID.String()
		updated = true
	}

	if d.HasChange("anti_affinity_group_ids") {
		set := d.Get("anti_affinity_group_ids").(*schema.Set)
		if set.Len() == 0 {
			nodepool.AntiAffinityGroupIDs = nil
			resetFields = append(resetFields, &nodepool.AntiAffinityGroupIDs)
		} else {
			nodepool.AntiAffinityGroupIDs = func() []string {
				list := make([]string, set.Len())
				for i, v := range set.List() {
					list[i] = v.(string)
				}
				return list
			}()
			updated = true
		}
	}

	if d.HasChange("security_group_ids") {
		set := d.Get("security_group_ids").(*schema.Set)
		if set.Len() == 0 {
			nodepool.SecurityGroupIDs = nil
			resetFields = append(resetFields, &nodepool.SecurityGroupIDs)
		} else {
			nodepool.SecurityGroupIDs = func() []string {
				list := make([]string, set.Len())
				for i, v := range set.List() {
					list[i] = v.(string)
				}
				return list
			}()
			updated = true
		}
	}

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	cluster, err := client.GetSKSCluster(ctx, zone, d.Get("cluster_id").(string))
	if err != nil {
		return err
	}

	if updated {
		if err = cluster.UpdateNodepool(ctx, nodepool); err != nil {
			return err
		}
	}

	for _, f := range resetFields {
		if err = cluster.ResetNodepoolField(ctx, nodepool, f); err != nil {
			return err
		}
	}

	if d.HasChange("size") {
		if err = cluster.ScaleNodepool(ctx, nodepool, int64(d.Get("size").(int))); err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] %s: update finished successfully", resourceSKSNodepoolIDString(d))

	return resourceSKSNodepoolRead(d, meta)
}

func resourceSKSNodepoolDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning delete", resourceSKSNodepoolIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()
	client := GetComputeClient(meta)

	zone := d.Get("zone").(string)

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	cluster, err := client.GetSKSCluster(ctx, zone, d.Get("cluster_id").(string))
	if err != nil {
		return err
	}

	if err = cluster.DeleteNodepool(ctx, &exov2.SKSNodepool{ID: d.Id()}); err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceSKSNodepoolIDString(d))

	return nil
}

func resourceSKSNodepoolApply(d *schema.ResourceData, meta interface{}, nodepool *exov2.SKSNodepool) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()
	client := GetComputeClient(meta)

	if err := d.Set("created_at", nodepool.CreatedAt.String()); err != nil {
		return err
	}

	if err := d.Set("description", nodepool.Description); err != nil {
		return err
	}

	if err := d.Set("disk_size", nodepool.DiskSize); err != nil {
		return err
	}

	if err := d.Set("instance_pool_id", nodepool.InstancePoolID); err != nil {
		return err
	}

	v, err := client.GetWithContext(ctx, &egoscale.ServiceOffering{
		ID: egoscale.MustParseUUID(nodepool.InstanceTypeID),
	})
	if err != nil {
		return err
	}
	if err := d.Set("instance_type", strings.ToLower(v.(*egoscale.ServiceOffering).Name)); err != nil {
		return err
	}

	if err := d.Set("name", nodepool.Name); err != nil {
		return err
	}

	if err := d.Set("anti_affinity_group_ids", nodepool.AntiAffinityGroupIDs); err != nil {
		return err
	}

	if err := d.Set("security_group_ids", nodepool.SecurityGroupIDs); err != nil {
		return err
	}

	if err := d.Set("size", nodepool.Size); err != nil {
		return err
	}

	if err := d.Set("state", nodepool.State); err != nil {
		return err
	}

	if err := d.Set("template_id", nodepool.TemplateID); err != nil {
		return err
	}

	if err := d.Set("version", nodepool.Version); err != nil {
		return err
	}

	return nil
}

func findSKSNodepool(ctx context.Context, d *schema.ResourceData, meta interface{}) (*exov2.SKSNodepool, error) {
	client := GetComputeClient(meta)

	zone, okZone := d.GetOk("zone")
	clusterID, okClusterID := d.GetOk("cluster_id")
	if okZone && okClusterID {
		ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone.(string)))
		cluster, err := client.GetSKSCluster(ctx, zone.(string), clusterID.(string))
		if err != nil {
			return nil, err
		}

		for _, np := range cluster.Nodepools {
			if np.ID == d.Id() {
				return np, nil
			}
		}
	}

	zones, err := client.ListZones(exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), defaultZone)))
	if err != nil {
		return nil, err
	}

	for _, zone := range zones {
		clusters, err := client.ListSKSClusters(
			exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone)),
			zone)
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				continue
			}

			return nil, err
		}

		for _, cluster := range clusters {
			for _, np := range cluster.Nodepools {
				if np.ID == d.Id() {
					if err := d.Set("zone", zone); err != nil {
						return nil, err
					}

					if err := d.Set("cluster_id", cluster.ID); err != nil {
						return nil, err
					}

					return np, nil
				}
			}
		}
	}

	return nil, nil
}
