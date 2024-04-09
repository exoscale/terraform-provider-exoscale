package exoscale

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
)

const (
	defaultSKSNodepoolDiskSize       int64 = 50
	defaultSKSNodepoolInstancePrefix       = "pool"

	sksNodepoolAddonStorageLVM = "storage-lvm"

	resSKSNodepoolAttrAntiAffinityGroupIDs   = "anti_affinity_group_ids"
	resSKSNodepoolAttrClusterID              = "cluster_id"
	resSKSNodepoolAttrCreatedAt              = "created_at"
	resSKSNodepoolAttrDeployTargetID         = "deploy_target_id"
	resSKSNodepoolAttrDescription            = "description"
	resSKSNodepoolAttrDiskSize               = "disk_size"
	resSKSNodepoolAttrInstancePoolID         = "instance_pool_id"
	resSKSNodepoolAttrInstancePrefix         = "instance_prefix"
	resSKSNodepoolAttrInstanceType           = "instance_type"
	resSKSNodepoolAttrKubeletGC              = "kubelet_image_gc"
	resSKSNodepoolAttrKubeletGCMinAge        = "min_age"
	resSKSNodepoolAttrKubeletGCHighThreshold = "high_threshold"
	resSKSNodepoolAttrKubeletGCLowThreshold  = "low_threshold"
	resSKSNodepoolAttrLabels                 = "labels"
	resSKSNodepoolAttrID                     = "id"
	resSKSNodepoolAttrName                   = "name"
	resSKSNodepoolAttrPrivateNetworkIDs      = "private_network_ids"
	resSKSNodepoolAttrSecurityGroupIDs       = "security_group_ids"
	resSKSNodepoolAttrSize                   = "size"
	resSKSNodepoolAttrState                  = "state"
	resSKSNodepoolAttrStorageLVM             = "storage_lvm"
	resSKSNodepoolAttrTaints                 = "taints"
	resSKSNodepoolAttrTemplateID             = "template_id"
	resSKSNodepoolAttrVersion                = "version"
	resSKSNodepoolAttrZone                   = "zone"
)

func resourceSKSNodepoolIDString(d general.ResourceIDStringer) string {
	return general.ResourceIDString(d, "exoscale_sks_nodepool")
}

func resourceSKSNodepool() *schema.Resource {
	s := map[string]*schema.Schema{
		resSKSNodepoolAttrAntiAffinityGroupIDs: {
			Type:        schema.TypeSet,
			Optional:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "A list of [exoscale_anti_affinity_group](./anti_affinity_group.md) (IDs) to be attached to the managed instances.",
		},
		resSKSNodepoolAttrClusterID: {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "The parent [exoscale_sks_cluster](./sks_cluster.md) ID.",
		},
		resSKSNodepoolAttrCreatedAt: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The pool creation date.",
		},
		resSKSNodepoolAttrDeployTargetID: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "A deploy target ID.",
		},
		resSKSNodepoolAttrDescription: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "A free-form text describing the pool.",
		},
		resSKSNodepoolAttrDiskSize: {
			Type:        schema.TypeInt,
			Optional:    true,
			Default:     defaultSKSNodepoolDiskSize,
			Description: "The managed instances disk size (GiB; default: `50`).",
		},
		resSKSNodepoolAttrInstancePoolID: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The underlying [exoscale_instance_pool](./instance_pool.md) ID.",
		},
		resSKSNodepoolAttrInstancePrefix: {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     defaultSKSNodepoolInstancePrefix,
			Description: "The string used to prefix the managed instances name (default `pool`).",
		},
		resSKSNodepoolAttrInstanceType: {
			Type:             schema.TypeString,
			Required:         true,
			ValidateDiagFunc: validateComputeInstanceType,
			// Ignore case differences
			DiffSuppressFunc: suppressCaseDiff,
			Description:      "The managed compute instances type (`<family>.<size>`, e.g. `standard.medium`; use the [Exoscale CLI](https://github.com/exoscale/cli/) - `exo compute instance-type list` - for the list of available types).",
		},
		resSKSNodepoolAttrKubeletGC: {
			Type:        schema.TypeSet,
			Optional:    true,
			Description: "Configuration for this nodepool's kubelet image garbage collector",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					resSKSNodepoolAttrKubeletGCMinAge: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "The minimum age for an unused image before it is garbage collected",
					},
					resSKSNodepoolAttrKubeletGCHighThreshold: {
						Type:        schema.TypeInt,
						Optional:    true,
						Description: "The percent of disk usage after which image garbage collection is always run",
					},
					resSKSNodepoolAttrKubeletGCLowThreshold: {
						Type:        schema.TypeInt,
						Optional:    true,
						Description: "The percent of disk usage before which image garbage collection is never run",
					},
				},
			},
		},
		resSKSNodepoolAttrLabels: {
			Type:        schema.TypeMap,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Description: "A map of key/value labels.",
		},
		resSKSNodepoolAttrID: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The SKS node pool ID.",
		},
		resSKSNodepoolAttrName: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The SKS node pool name.",
		},
		resSKSNodepoolAttrPrivateNetworkIDs: {
			Type:        schema.TypeSet,
			Optional:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "A list of [exoscale_private_network](./private_network.md) (IDs) to be attached to the managed instances.",
		},
		resSKSNodepoolAttrSecurityGroupIDs: {
			Type:        schema.TypeSet,
			Optional:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "A list of [exoscale_security_group](./security_group.md) (IDs) to be attached to the managed instances.",
		},
		resSKSNodepoolAttrSize: {
			Type:     schema.TypeInt,
			Required: true,
		},
		resSKSNodepoolAttrState: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The current pool state.",
		},
		resSKSNodepoolAttrStorageLVM: {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Create nodes with non-standard partitioning for persistent storage (requires min 100G of disk space) (may only be set at creation time).",
		},
		resSKSNodepoolAttrTaints: {
			Type:        schema.TypeMap,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Description: `A map of key/value Kubernetes [taints](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) ('taints = { <key> = "<value>:<effect>" }').`,
		},
		resSKSNodepoolAttrTemplateID: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The managed instances template ID.",
		},
		resSKSNodepoolAttrVersion: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The managed instances version.",
		},
		resSKSNodepoolAttrZone: {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
		},
	}

	return &schema.Resource{
		Schema: s,

		Description: "Manage Exoscale Scalable Kubernetes Service (SKS) Node Pools.",

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
			Create: schema.DefaultTimeout(config.DefaultTimeout),
			Read:   schema.DefaultTimeout(config.DefaultTimeout),
			Update: schema.DefaultTimeout(config.DefaultTimeout),
			Delete: schema.DefaultTimeout(config.DefaultTimeout),
		},
	}
}

func resourceSKSNodepoolCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning create", map[string]interface{}{
		"id": resourceSKSNodepoolIDString(d),
	})

	zone := d.Get(resSKSNodepoolAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := getClient(meta)

	sksCluster, err := client.GetSKSCluster(ctx, zone, d.Get(resSKSNodepoolAttrClusterID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	sksNodepool := new(egoscale.SKSNodepool)

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

	if k, ok := d.GetOk(resSKSNodepoolAttrKubeletGC); ok {
		kubeletGc := k.(*schema.Set).List()[0].(map[string]interface{})
		sksNodepoolKubeletGc := new(egoscale.SKSNodepoolKubeletImageGc)

		if val, ok := kubeletGc[resSKSNodepoolAttrKubeletGCMinAge]; ok {
			sksNodepoolKubeletGcMinAge := val.(string)
			sksNodepoolKubeletGc.MinAge = &sksNodepoolKubeletGcMinAge
		}

		if val, ok := kubeletGc[resSKSNodepoolAttrKubeletGCLowThreshold]; ok {
			sksNodepoolKubeletGcLowThreshold := val.(int)
			sksNodepoolKubeletGcLowThresholdInt64 := int64(sksNodepoolKubeletGcLowThreshold)
			sksNodepoolKubeletGc.LowThreshold = &sksNodepoolKubeletGcLowThresholdInt64

		}

		if val, ok := kubeletGc[resSKSNodepoolAttrKubeletGCHighThreshold]; ok {
			sksNodepoolKubeletGcHighThreshold := val.(int)
			sksNodepoolKubeletGcHighThresholdInt64 := int64(sksNodepoolKubeletGcHighThreshold)
			sksNodepoolKubeletGc.HighThreshold = &sksNodepoolKubeletGcHighThresholdInt64

		}

		sksNodepool.KubeletImageGc = sksNodepoolKubeletGc
	}

	if l, ok := d.GetOk(resSKSNodepoolAttrLabels); ok {
		labels := make(map[string]string)
		for k, v := range l.(map[string]interface{}) {
			labels[k] = v.(string)
		}
		sksNodepool.Labels = &labels
	}

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

	var addOns []string
	if enableStorageLVM := d.Get(resSKSNodepoolAttrStorageLVM).(bool); enableStorageLVM {
		addOns = append(addOns, sksNodepoolAddonStorageLVM)
	}
	if len(addOns) > 0 {
		sksNodepool.AddOns = &addOns
	}

	if t, ok := d.GetOk(resSKSNodepoolAttrTaints); ok {
		taints := make(map[string]*egoscale.SKSNodepoolTaint)
		for k, v := range t.(map[string]interface{}) {
			taint, err := parseSKSNodepoolTaint(v.(string))
			if err != nil {
				return diag.Errorf("invalid taint %q: %s", v.(string), err)
			}
			taints[k] = taint
		}
		sksNodepool.Taints = &taints
	}

	sksNodepool, err = client.CreateSKSNodepool(ctx, zone, sksCluster, sksNodepool)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*sksNodepool.ID)

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": resourceSKSNodepoolIDString(d),
	})

	return resourceSKSNodepoolRead(ctx, d, meta)
}

func resourceSKSNodepoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceSKSNodepoolIDString(d),
	})

	zone := d.Get(resSKSNodepoolAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := getClient(meta)

	sks, err := client.GetSKSCluster(ctx, zone, d.Get(resSKSNodepoolAttrClusterID).(string))
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Parent SKS cluster doesn't exist anymore, so does the SKS Nodepool.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var sksNodepool *egoscale.SKSNodepool
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

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceSKSNodepoolIDString(d),
	})

	return diag.FromErr(resourceSKSNodepoolApply(ctx, client, d, sksNodepool))
}

func resourceSKSNodepoolUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning update", map[string]interface{}{
		"id": resourceSKSNodepoolIDString(d),
	})

	zone := d.Get(resSKSNodepoolAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutUpdate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := getClient(meta)

	sksCluster, err := client.GetSKSCluster(ctx, zone, d.Get(resSKSNodepoolAttrClusterID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	var sksNodepool *egoscale.SKSNodepool
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

	if d.HasChange(resSKSNodepoolAttrLabels) {
		labels := make(map[string]string)
		for k, v := range d.Get(resSKSNodepoolAttrLabels).(map[string]interface{}) {
			labels[k] = v.(string)
		}
		sksNodepool.Labels = &labels
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

	if d.HasChange(resSKSNodepoolAttrTaints) {
		taints := make(map[string]*egoscale.SKSNodepoolTaint)
		for k, v := range d.Get(resSKSNodepoolAttrTaints).(map[string]interface{}) {
			taint, err := parseSKSNodepoolTaint(v.(string))
			if err != nil {
				return diag.Errorf("invalid taint %q: %s", v.(string), err)
			}
			taints[k] = taint
		}
		sksNodepool.Taints = &taints
		updated = true
	}

	if updated {
		if err = client.UpdateSKSNodepool(ctx, zone, sksCluster, sksNodepool); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(resSKSNodepoolAttrSize) {
		if err = client.ScaleSKSNodepool(
			ctx,
			zone,
			sksCluster,
			sksNodepool,
			int64(d.Get(resSKSNodepoolAttrSize).(int)),
		); err != nil {
			return diag.FromErr(err)
		}
	}

	tflog.Debug(ctx, "update finished successfully", map[string]interface{}{
		"id": resourceSKSNodepoolIDString(d),
	})

	return resourceSKSNodepoolRead(ctx, d, meta)
}

func resourceSKSNodepoolDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning delete", map[string]interface{}{
		"id": resourceSKSNodepoolIDString(d),
	})

	zone := d.Get(resSKSNodepoolAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := getClient(meta)

	sksCluster, err := client.GetSKSCluster(ctx, zone, d.Get(resSKSNodepoolAttrClusterID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	sksNodepoolID := d.Id()
	if err = client.DeleteSKSNodepool(ctx, zone, sksCluster, &egoscale.SKSNodepool{ID: &sksNodepoolID}); err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": resourceSKSNodepoolIDString(d),
	})

	return nil
}

func resourceSKSNodepoolApply(
	ctx context.Context,
	client *egoscale.Client,
	d *schema.ResourceData,
	sksNodepool *egoscale.SKSNodepool,
) error {
	if sksNodepool.AntiAffinityGroupIDs != nil {
		antiAffinityGroupIDs := make([]string, len(*sksNodepool.AntiAffinityGroupIDs))
		copy(antiAffinityGroupIDs, *sksNodepool.AntiAffinityGroupIDs)
		if err := d.Set(resSKSNodepoolAttrAntiAffinityGroupIDs, antiAffinityGroupIDs); err != nil {
			return err
		}
	}

	if sksNodepool.AddOns != nil {
		if err := d.Set(resSKSNodepoolAttrStorageLVM, in(*sksNodepool.AddOns, sksNodepoolAddonStorageLVM)); err != nil {
			return err
		}
	}

	if err := d.Set(resSKSNodepoolAttrCreatedAt, sksNodepool.CreatedAt.String()); err != nil {
		return err
	}

	if err := d.Set(resSKSNodepoolAttrDeployTargetID, defaultString(sksNodepool.DeployTargetID, "")); err != nil {
		return err
	}

	if err := d.Set(resSKSNodepoolAttrDescription, defaultString(sksNodepool.Description, "")); err != nil {
		return err
	}

	if err := d.Set(resSKSNodepoolAttrDiskSize, *sksNodepool.DiskSize); err != nil {
		return err
	}

	if err := d.Set(resSKSNodepoolAttrInstancePoolID, *sksNodepool.InstancePoolID); err != nil {
		return err
	}

	if err := d.Set(resSKSNodepoolAttrInstancePrefix, defaultString(sksNodepool.InstancePrefix, "")); err != nil {
		return err
	}

	instanceType, err := client.GetInstanceType(
		ctx,
		d.Get(resSKSNodepoolAttrZone).(string),
		*sksNodepool.InstanceTypeID,
	)
	if err != nil {
		return fmt.Errorf("error retrieving instance type: %w", err)
	}
	if err := d.Set(resSKSNodepoolAttrInstanceType, fmt.Sprintf(
		"%s.%s",
		strings.ToLower(*instanceType.Family),
		strings.ToLower(*instanceType.Size),
	)); err != nil {
		return err
	}

	kubeletGc := d.Get(resSKSNodepoolAttrKubeletGC).(*schema.Set)
	if err := d.Set(resSKSNodepoolAttrKubeletGC, schema.NewSet(kubeletGc.F, []interface{}{
		func() map[string]interface{} {
			i := map[string]interface{}{}
			if sksNodepool.KubeletImageGc.MinAge != nil {
				i[resSKSNodepoolAttrKubeletGCMinAge] = *sksNodepool.KubeletImageGc.MinAge
			}
			if sksNodepool.KubeletImageGc.HighThreshold != nil {
				i[resSKSNodepoolAttrKubeletGCHighThreshold] = int(*sksNodepool.KubeletImageGc.HighThreshold)
			}
			if sksNodepool.KubeletImageGc.LowThreshold != nil {
				i[resSKSNodepoolAttrKubeletGCLowThreshold] = int(*sksNodepool.KubeletImageGc.LowThreshold)
			}
			return i
		}(),
	})); err != nil {
		return err
	}

	if err := d.Set(resSKSNodepoolAttrLabels, sksNodepool.Labels); err != nil {
		return err
	}

	if err := d.Set(resSKSNodepoolAttrName, *sksNodepool.Name); err != nil {
		return err
	}

	if sksNodepool.PrivateNetworkIDs != nil {
		privateNetworkIDs := make([]string, len(*sksNodepool.PrivateNetworkIDs))
		copy(privateNetworkIDs, *sksNodepool.PrivateNetworkIDs)
		if err := d.Set(resSKSNodepoolAttrPrivateNetworkIDs, privateNetworkIDs); err != nil {
			return err
		}
	}

	if sksNodepool.SecurityGroupIDs != nil {
		securityGroupIDs := make([]string, len(*sksNodepool.SecurityGroupIDs))
		copy(securityGroupIDs, *sksNodepool.SecurityGroupIDs)
		if err := d.Set(resSKSNodepoolAttrSecurityGroupIDs, securityGroupIDs); err != nil {
			return err
		}
	}

	if err := d.Set(resSKSNodepoolAttrSize, *sksNodepool.Size); err != nil {
		return err
	}

	if err := d.Set(resSKSNodepoolAttrState, *sksNodepool.State); err != nil {
		return err
	}

	if sksNodepool.Taints != nil {
		taints := make(map[string]string)
		for k, v := range *sksNodepool.Taints {
			taints[k] = fmt.Sprintf("%s:%s", v.Value, v.Effect)
		}
		if err := d.Set(resSKSNodepoolAttrTaints, taints); err != nil {
			return err
		}
	}

	if err := d.Set(resSKSNodepoolAttrTemplateID, *sksNodepool.TemplateID); err != nil {
		return err
	}

	if err := d.Set(resSKSNodepoolAttrVersion, *sksNodepool.Version); err != nil {
		return err
	}

	return nil
}

// parseSKSNodepoolTaint parses a CLI-formatted Kubernetes Node taint
// description formatted as VALUE:EFFECT, and returns discrete values
// for the value/effect as egoscale.SKSNodepoolTaint, or an error if
// the input value parsing failed.
func parseSKSNodepoolTaint(v string) (*egoscale.SKSNodepoolTaint, error) {
	parts := strings.SplitN(v, ":", 2)
	if len(parts) != 2 {
		return nil, errors.New("expected format VALUE:EFFECT")
	}
	taintValue, taintEffect := parts[0], parts[1]

	if taintValue == "" || taintEffect == "" {
		return nil, errors.New("expected format VALUE:EFFECT")
	}

	return &egoscale.SKSNodepoolTaint{
		Effect: taintEffect,
		Value:  taintValue,
	}, nil
}
