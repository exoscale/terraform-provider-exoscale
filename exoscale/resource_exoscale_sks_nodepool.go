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
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	defaultSKSNodepoolDiskSize       int64 = 50
	defaultSKSNodepoolInstancePrefix       = "pool"

	resSKSNodepoolAttrAntiAffinityGroupIDs = "anti_affinity_group_ids"
	resSKSNodepoolAttrClusterID            = "cluster_id"
	resSKSNodepoolAttrCreatedAt            = "created_at"
	resSKSNodepoolAttrDeployTargetID       = "deploy_target_id"
	resSKSNodepoolAttrDescription          = "description"
	resSKSNodepoolAttrDiskSize             = "disk_size"
	resSKSNodepoolAttrInstancePoolID       = "instance_pool_id"
	resSKSNodepoolAttrInstancePrefix       = "instance_prefix"
	resSKSNodepoolAttrInstanceType         = "instance_type"
	resSKSNodepoolAttrName                 = "name"
	resSKSNodepoolAttrPrivateNetworkIDs    = "private_network_ids"
	resSKSNodepoolAttrSecurityGroupIDs     = "security_group_ids"
	resSKSNodepoolAttrSize                 = "size"
	resSKSNodepoolAttrState                = "state"
	resSKSNodepoolAttrTemplateID           = "template_id"
	resSKSNodepoolAttrVersion              = "version"
	resSKSNodepoolAttrZone                 = "zone"
)

func resourceSKSNodepoolIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_sks_nodepool")
}

func resourceSKSNodepool() *schema.Resource {
	s := map[string]*schema.Schema{
		resSKSNodepoolAttrAntiAffinityGroupIDs: {
			Type:     schema.TypeSet,
			Optional: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		resSKSNodepoolAttrClusterID: {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		resSKSNodepoolAttrCreatedAt: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resSKSNodepoolAttrDeployTargetID: {
			Type:     schema.TypeString,
			Optional: true,
		},
		resSKSNodepoolAttrDescription: {
			Type:     schema.TypeString,
			Optional: true,
		},
		resSKSNodepoolAttrDiskSize: {
			Type:     schema.TypeInt,
			Optional: true,
			Default:  defaultSKSNodepoolDiskSize,
		},
		resSKSNodepoolAttrInstancePoolID: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resSKSNodepoolAttrInstancePrefix: {
			Type:     schema.TypeString,
			Optional: true,
			Default:  defaultSKSNodepoolInstancePrefix,
		},
		resSKSNodepoolAttrInstanceType: {
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
		resSKSNodepoolAttrName: {
			Type:     schema.TypeString,
			Required: true,
		},
		resSKSNodepoolAttrPrivateNetworkIDs: {
			Type:     schema.TypeSet,
			Optional: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		resSKSNodepoolAttrSecurityGroupIDs: {
			Type:     schema.TypeSet,
			Optional: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		resSKSNodepoolAttrSize: {
			Type:     schema.TypeInt,
			Required: true,
		},
		resSKSNodepoolAttrState: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resSKSNodepoolAttrTemplateID: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resSKSNodepoolAttrVersion: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resSKSNodepoolAttrZone: {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
	}

	return &schema.Resource{
		Schema: s,

		CreateContext: resourceSKSNodepoolCreate,
		ReadContext:   resourceSKSNodepoolRead,
		UpdateContext: resourceSKSNodepoolUpdate,
		DeleteContext: resourceSKSNodepoolDelete,

		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
				zonedRes, err := zonedStateContextFunc(ctx, d, nil)
				if err != nil {
					return nil, err
				}
				d = zonedRes[0]

				parts := strings.SplitN(d.Id(), "/", 2)
				if len(parts) != 2 {
					return nil, fmt.Errorf(`invalid ID %q, expected format "<CLUSTER-ID>/<NODEPOOL-ID>@<ZONE>"`, d.Id())
				}

				d.SetId(parts[1])
				if err := d.Set(resSKSNodepoolAttrClusterID, parts[0]); err != nil {
					return nil, err
				}

				return []*schema.ResourceData{d}, nil
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceSKSNodepoolCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning create", resourceSKSNodepoolIDString(d))

	zone := d.Get(resSKSNodepoolAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	sksCluster, err := client.GetSKSCluster(ctx, zone, d.Get(resSKSNodepoolAttrClusterID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	sksNodepool := new(exov2.SKSNodepool)

	if set, ok := d.Get(resSKSNodepoolAttrAntiAffinityGroupIDs).(*schema.Set); ok {
		sksNodepool.AntiAffinityGroupIDs = func() (v *[]string) {
			if l := set.Len(); l > 0 {
				list := make([]string, l)
				for i, v := range set.List() {
					list[i] = v.(string)
				}
				v = &list
			}
			return
		}()
	}

	if v, ok := d.GetOk(resSKSNodepoolAttrDeployTargetID); ok {
		s := v.(string)
		sksNodepool.DeployTargetID = &s
	}

	if v, ok := d.GetOk(resSKSNodepoolAttrDescription); ok {
		s := v.(string)
		sksNodepool.Description = &s
	}

	if v, ok := d.GetOk(resSKSNodepoolAttrDiskSize); ok {
		i := int64(v.(int))
		sksNodepool.DiskSize = &i
	}

	if v, ok := d.GetOk(resSKSNodepoolAttrInstancePrefix); ok {
		s := v.(string)
		sksNodepool.InstancePrefix = &s
	}

	instanceType, err := client.FindInstanceType(ctx, zone, d.Get(resSKSNodepoolAttrInstanceType).(string))
	if err != nil {
		return diag.Errorf("error retrieving instance type: %s", err)
	}
	sksNodepool.InstanceTypeID = instanceType.ID

	if v, ok := d.GetOk(resSKSNodepoolAttrName); ok {
		s := v.(string)
		sksNodepool.Name = &s
	}

	if set, ok := d.Get(resSKSNodepoolAttrPrivateNetworkIDs).(*schema.Set); ok {
		sksNodepool.PrivateNetworkIDs = func() (v *[]string) {
			if l := set.Len(); l > 0 {
				list := make([]string, l)
				for i, v := range set.List() {
					list[i] = v.(string)
				}
				v = &list
			}
			return
		}()
	}

	if set, ok := d.Get(resSKSNodepoolAttrSecurityGroupIDs).(*schema.Set); ok {
		sksNodepool.SecurityGroupIDs = func() (v *[]string) {
			if l := set.Len(); l > 0 {
				list := make([]string, l)
				for i, v := range set.List() {
					list[i] = v.(string)
				}
				v = &list
			}
			return
		}()
	}

	if v, ok := d.GetOk(resSKSNodepoolAttrSize); ok {
		i := int64(v.(int))
		sksNodepool.Size = &i
	}

	sksNodepool, err = sksCluster.AddNodepool(ctx, sksNodepool)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*sksNodepool.ID)

	log.Printf("[DEBUG] %s: create finished successfully", resourceSKSNodepoolIDString(d))

	return resourceSKSNodepoolRead(ctx, d, meta)
}

func resourceSKSNodepoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning read", resourceSKSNodepoolIDString(d))

	zone := d.Get(resSKSNodepoolAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	sks, err := client.GetSKSCluster(ctx, zone, d.Get(resSKSNodepoolAttrClusterID).(string))
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Parent SKS cluster doesn't exist anymore, so does the SKS Nodepool.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var sksNodepool *exov2.SKSNodepool
	for _, np := range sks.Nodepools {
		if *np.ID == d.Id() {
			sksNodepool = np
			break
		}
	}
	if sksNodepool == nil {
		// Resource doesn't exist anymore, signaling the core to remove it from the state.
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceSKSNodepoolIDString(d))

	return resourceSKSNodepoolApply(ctx, client, d, sksNodepool)
}

func resourceSKSNodepoolUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning update", resourceSKSNodepoolIDString(d))

	zone := d.Get(resSKSNodepoolAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutUpdate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	sksCluster, err := client.GetSKSCluster(ctx, zone, d.Get(resSKSNodepoolAttrClusterID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	var sksNodepool *exov2.SKSNodepool
	for _, np := range sksCluster.Nodepools {
		if *np.ID == d.Id() {
			sksNodepool = np
			break
		}
	}
	if sksNodepool == nil {
		return diag.Errorf("SKS Nodepool %q not found", d.Id())
	}

	var updated bool

	if d.HasChange(resSKSNodepoolAttrAntiAffinityGroupIDs) {
		set := d.Get(resSKSNodepoolAttrAntiAffinityGroupIDs).(*schema.Set)
		sksNodepool.AntiAffinityGroupIDs = func() *[]string {
			list := make([]string, set.Len())
			for i, v := range set.List() {
				list[i] = v.(string)
			}
			return &list
		}()
		updated = true
	}

	if d.HasChange(resSKSNodepoolAttrDeployTargetID) {
		v := d.Get(resSKSNodepoolAttrDeployTargetID).(string)
		sksNodepool.DeployTargetID = &v
		updated = true
	}

	if d.HasChange(resSKSNodepoolAttrDescription) {
		v := d.Get(resSKSNodepoolAttrDescription).(string)
		sksNodepool.Description = &v
		updated = true
	}

	if d.HasChange(resSKSNodepoolAttrDiskSize) {
		v := int64(d.Get(resSKSNodepoolAttrDiskSize).(int))
		sksNodepool.DiskSize = &v
		updated = true
	}

	if d.HasChange(resSKSNodepoolAttrInstancePrefix) {
		v := d.Get(resSKSNodepoolAttrInstancePrefix).(string)
		sksNodepool.InstancePrefix = &v
		updated = true
	}

	if d.HasChange(resSKSNodepoolAttrInstanceType) {
		instanceType, err := client.FindInstanceType(ctx, zone, d.Get(resSKSNodepoolAttrInstanceType).(string))
		if err != nil {
			return diag.Errorf("error retrieving instance type: %s", err)
		}
		sksNodepool.InstanceTypeID = instanceType.ID
		updated = true
	}

	if d.HasChange(resSKSNodepoolAttrName) {
		v := d.Get(resSKSNodepoolAttrName).(string)
		sksNodepool.Name = &v
		updated = true
	}

	if d.HasChange(resSKSNodepoolAttrPrivateNetworkIDs) {
		set := d.Get(resSKSNodepoolAttrPrivateNetworkIDs).(*schema.Set)
		sksNodepool.PrivateNetworkIDs = func() *[]string {
			list := make([]string, set.Len())
			for i, v := range set.List() {
				list[i] = v.(string)
			}
			return &list
		}()
		updated = true
	}

	if d.HasChange(resSKSNodepoolAttrSecurityGroupIDs) {
		set := d.Get(resSKSNodepoolAttrSecurityGroupIDs).(*schema.Set)
		sksNodepool.SecurityGroupIDs = func() *[]string {
			list := make([]string, set.Len())
			for i, v := range set.List() {
				list[i] = v.(string)
			}
			return &list
		}()
		updated = true
	}

	if updated {
		if err = sksCluster.UpdateNodepool(ctx, sksNodepool); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(resSKSNodepoolAttrSize) {
		if err = sksCluster.ScaleNodepool(ctx, sksNodepool, int64(d.Get(resSKSNodepoolAttrSize).(int))); err != nil {
			return diag.FromErr(err)
		}
	}

	log.Printf("[DEBUG] %s: update finished successfully", resourceSKSNodepoolIDString(d))

	return resourceSKSNodepoolRead(ctx, d, meta)
}

func resourceSKSNodepoolDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning delete", resourceSKSNodepoolIDString(d))

	zone := d.Get(resSKSNodepoolAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	cluster, err := client.GetSKSCluster(ctx, zone, d.Get(resSKSNodepoolAttrClusterID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	sksNodepoolID := d.Id()
	if err = cluster.DeleteNodepool(ctx, &exov2.SKSNodepool{ID: &sksNodepoolID}); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceSKSNodepoolIDString(d))

	return nil
}

func resourceSKSNodepoolApply(
	ctx context.Context,
	client *egoscale.Client,
	d *schema.ResourceData,
	sksNodepool *exov2.SKSNodepool,
) diag.Diagnostics {
	if sksNodepool.AntiAffinityGroupIDs != nil {
		antiAffinityGroupIDs := make([]string, len(*sksNodepool.AntiAffinityGroupIDs))
		for i, id := range *sksNodepool.AntiAffinityGroupIDs {
			antiAffinityGroupIDs[i] = id
		}
		if err := d.Set(resSKSNodepoolAttrAntiAffinityGroupIDs, antiAffinityGroupIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(resSKSNodepoolAttrCreatedAt, sksNodepool.CreatedAt.String()); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resSKSNodepoolAttrDeployTargetID, defaultString(sksNodepool.DeployTargetID, "")); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resSKSNodepoolAttrDescription, defaultString(sksNodepool.Description, "")); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resSKSNodepoolAttrDiskSize, *sksNodepool.DiskSize); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resSKSNodepoolAttrInstancePoolID, *sksNodepool.InstancePoolID); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resSKSNodepoolAttrInstancePrefix, defaultString(sksNodepool.InstancePrefix, "")); err != nil {
		return diag.FromErr(err)
	}

	instanceType, err := client.GetInstanceType(
		ctx,
		d.Get(resSKSNodepoolAttrZone).(string),
		*sksNodepool.InstanceTypeID,
	)
	if err != nil {
		return diag.Errorf("error retrieving instance type: %s", err)
	}
	if err := d.Set(resSKSNodepoolAttrInstanceType, *instanceType.Size); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resSKSNodepoolAttrName, *sksNodepool.Name); err != nil {
		return diag.FromErr(err)
	}

	if sksNodepool.PrivateNetworkIDs != nil {
		privateNetworkIDs := make([]string, len(*sksNodepool.PrivateNetworkIDs))
		for i, id := range *sksNodepool.PrivateNetworkIDs {
			privateNetworkIDs[i] = id
		}
		if err := d.Set(resSKSNodepoolAttrPrivateNetworkIDs, privateNetworkIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if sksNodepool.SecurityGroupIDs != nil {
		securityGroupIDs := make([]string, len(*sksNodepool.SecurityGroupIDs))
		for i, id := range *sksNodepool.SecurityGroupIDs {
			securityGroupIDs[i] = id
		}
		if err := d.Set(resSKSNodepoolAttrSecurityGroupIDs, securityGroupIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(resSKSNodepoolAttrSize, *sksNodepool.Size); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resSKSNodepoolAttrState, *sksNodepool.State); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resSKSNodepoolAttrTemplateID, *sksNodepool.TemplateID); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resSKSNodepoolAttrVersion, *sksNodepool.Version); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
