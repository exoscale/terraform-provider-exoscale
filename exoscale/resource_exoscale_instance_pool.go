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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	defaultInstancePoolInstancePrefix = "pool"

	resInstancePoolAttrAffinityGroupIDs = "affinity_group_ids"
	resInstancePoolAttrDeployTargetID   = "deploy_target_id"
	resInstancePoolAttrDescription      = "description"
	resInstancePoolAttrDiskSize         = "disk_size"
	resInstancePoolAttrElasticIPIDs     = "elastic_ip_ids"
	resInstancePoolAttrInstancePrefix   = "instance_prefix"
	resInstancePoolAttrIPv6             = "ipv6"
	resInstancePoolAttrKeyPair          = "key_pair"
	resInstancePoolAttrName             = "name"
	resInstancePoolAttrNetworkIDs       = "network_ids"
	resInstancePoolAttrSecurityGroupIDs = "security_group_ids"
	resInstancePoolAttrServiceOffering  = "service_offering"
	resInstancePoolAttrSize             = "size"
	resInstancePoolAttrState            = "state"
	resInstancePoolAttrTemplateID       = "template_id"
	resInstancePoolAttrUserData         = "user_data"
	resInstancePoolAttrVirtualMachines  = "virtual_machines"
	resInstancePoolAttrZone             = "zone"
)

func resourceInstancePoolIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_instance_pool")
}

func resourceInstancePool() *schema.Resource {
	s := map[string]*schema.Schema{
		resInstancePoolAttrAffinityGroupIDs: {
			Type:     schema.TypeSet,
			Optional: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		resInstancePoolAttrDeployTargetID: {
			Type:     schema.TypeString,
			Optional: true,
		},
		resInstancePoolAttrDescription: {
			Type:     schema.TypeString,
			Optional: true,
		},
		resInstancePoolAttrDiskSize: {
			Type:     schema.TypeInt,
			Computed: true,
			Optional: true,
		},
		resInstancePoolAttrElasticIPIDs: {
			Type:     schema.TypeSet,
			Optional: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		resInstancePoolAttrInstancePrefix: {
			Type:     schema.TypeString,
			Optional: true,
			Default:  defaultInstancePoolInstancePrefix,
		},
		resInstancePoolAttrIPv6: {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
		},
		resInstancePoolAttrKeyPair: {
			Type:     schema.TypeString,
			Optional: true,
		},
		resInstancePoolAttrName: {
			Type:     schema.TypeString,
			Required: true,
		},
		resInstancePoolAttrNetworkIDs: {
			Type:     schema.TypeSet,
			Optional: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		resInstancePoolAttrSecurityGroupIDs: {
			Type:     schema.TypeSet,
			Optional: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		resInstancePoolAttrServiceOffering: {
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
		resInstancePoolAttrSize: {
			Type:         schema.TypeInt,
			Required:     true,
			ValidateFunc: validation.IntAtLeast(1),
		},
		resInstancePoolAttrState: {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		resInstancePoolAttrTemplateID: {
			Type:     schema.TypeString,
			Required: true,
		},
		resInstancePoolAttrUserData: {
			Type:     schema.TypeString,
			Optional: true,
		},
		resInstancePoolAttrVirtualMachines: {
			Type:     schema.TypeSet,
			Optional: true,
			Computed: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		resInstancePoolAttrZone: {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
	}

	return &schema.Resource{
		Schema: s,

		CreateContext: resourceInstancePoolCreate,
		ReadContext:   resourceInstancePoolRead,
		UpdateContext: resourceInstancePoolUpdate,
		DeleteContext: resourceInstancePoolDelete,

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

func resourceInstancePoolCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning create", resourceInstancePoolIDString(d))

	zone := d.Get(resInstancePoolAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	instancePool := new(exov2.InstancePool)

	if v, ok := d.GetOk(resInstancePoolAttrDeployTargetID); ok {
		s := v.(string)
		instancePool.DeployTargetID = &s
	}

	if v, ok := d.GetOk(resInstancePoolAttrDescription); ok {
		s := v.(string)
		instancePool.Description = &s
	}

	if v, ok := d.GetOk(resInstancePoolAttrDiskSize); ok {
		i := int64(v.(int))
		instancePool.DiskSize = &i
	}

	if v, ok := d.GetOk(resInstancePoolAttrName); ok {
		s := v.(string)
		instancePool.Name = &s
	}

	if v, ok := d.GetOk(resInstancePoolAttrInstancePrefix); ok {
		s := v.(string)
		instancePool.InstancePrefix = &s
	}

	if v, ok := d.GetOk(resInstancePoolAttrKeyPair); ok {
		s := v.(string)
		instancePool.SSHKey = &s
	}

	if v, ok := d.GetOk(resInstancePoolAttrSize); ok {
		i := int64(v.(int))
		instancePool.Size = &i
	}

	if v, ok := d.GetOk(resInstancePoolAttrTemplateID); ok {
		s := v.(string)
		instancePool.TemplateID = &s
	}

	instanceType, err := client.FindInstanceType(ctx, zone, d.Get(resInstancePoolAttrServiceOffering).(string))
	if err != nil {
		return diag.Errorf("error retrieving instance type: %s", err)
	}
	instancePool.InstanceTypeID = instanceType.ID

	if set, ok := d.Get(resInstancePoolAttrAffinityGroupIDs).(*schema.Set); ok {
		instancePool.AntiAffinityGroupIDs = func() (v *[]string) {
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

	if set, ok := d.Get(resInstancePoolAttrSecurityGroupIDs).(*schema.Set); ok {
		instancePool.SecurityGroupIDs = func() (v *[]string) {
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

	if set, ok := d.Get(resInstancePoolAttrNetworkIDs).(*schema.Set); ok {
		instancePool.PrivateNetworkIDs = func() (v *[]string) {
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

	if set, ok := d.Get(resInstancePoolAttrElasticIPIDs).(*schema.Set); ok {
		instancePool.ElasticIPIDs = func() (v *[]string) {
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

	enableIPv6 := d.Get(resInstancePoolAttrIPv6).(bool)
	instancePool.IPv6Enabled = &enableIPv6

	if v := d.Get(resInstancePoolAttrUserData).(string); v != "" {
		userData, err := encodeUserData(v)
		if err != nil {
			return diag.FromErr(err)
		}
		instancePool.UserData = &userData
	}

	// FIXME: we have to reference the embedded egoscale/v2.Client explicitly
	//  here because there is already a CreateInstancePool() method on the root
	//  egoscale client clashing with the v2 one. This can be changed once we
	//  use API V2-only calls.
	instancePool, err = client.Client.CreateInstancePool(ctx, zone, instancePool)
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(*instancePool.ID)

	log.Printf("[DEBUG] %s: create finished successfully", resourceInstancePoolIDString(d))

	return resourceInstancePoolRead(ctx, d, meta)
}

func resourceInstancePoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning read", resourceInstancePoolIDString(d))

	zone := d.Get(resInstancePoolAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	instancePool, err := client.GetInstancePool(ctx, zone, d.Id())
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceInstancePoolIDString(d))

	return resourceInstancePoolApply(ctx, GetComputeClient(meta), d, instancePool)
}

func resourceInstancePoolUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning update", resourceInstancePoolIDString(d))

	zone := d.Get(resInstancePoolAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutUpdate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	instancePool, err := client.GetInstancePool(ctx, zone, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var updated bool

	if d.HasChange(resInstancePoolAttrAffinityGroupIDs) {
		set := d.Get(resInstancePoolAttrAffinityGroupIDs).(*schema.Set)
		instancePool.AntiAffinityGroupIDs = func() *[]string {
			list := make([]string, set.Len())
			for i, v := range set.List() {
				list[i] = v.(string)
			}
			return &list
		}()
		updated = true
	}

	if d.HasChange(resInstancePoolAttrDeployTargetID) {
		v := d.Get(resInstancePoolAttrDeployTargetID).(string)
		instancePool.DeployTargetID = &v
		updated = true
	}

	if d.HasChange(resInstancePoolAttrDescription) {
		v := d.Get(resInstancePoolAttrDescription).(string)
		instancePool.Description = &v
		updated = true
	}

	if d.HasChange(resInstancePoolAttrDiskSize) {
		v := int64(d.Get(resInstancePoolAttrDiskSize).(int))
		instancePool.DiskSize = &v
		updated = true
	}

	if d.HasChange(resInstancePoolAttrElasticIPIDs) {
		set := d.Get(resInstancePoolAttrElasticIPIDs).(*schema.Set)
		instancePool.ElasticIPIDs = func() *[]string {
			list := make([]string, set.Len())
			for i, v := range set.List() {
				list[i] = v.(string)
			}
			return &list
		}()
		updated = true
	}

	if d.HasChange(resInstancePoolAttrInstancePrefix) {
		v := d.Get(resInstancePoolAttrInstancePrefix).(string)
		instancePool.InstancePrefix = &v
		updated = true
	}

	if d.HasChange(resInstancePoolAttrIPv6) {
		v := d.Get(resInstancePoolAttrIPv6).(bool)
		instancePool.IPv6Enabled = &v
		updated = true
	}

	if d.HasChange(resInstancePoolAttrKeyPair) {
		v := d.Get(resInstancePoolAttrKeyPair).(string)
		instancePool.SSHKey = &v
		updated = true
	}

	if d.HasChange(resInstancePoolAttrName) {
		v := d.Get(resInstancePoolAttrName).(string)
		instancePool.Name = &v
		updated = true
	}

	if d.HasChange(resInstancePoolAttrNetworkIDs) {
		set := d.Get(resInstancePoolAttrNetworkIDs).(*schema.Set)
		instancePool.PrivateNetworkIDs = func() *[]string {
			list := make([]string, set.Len())
			for i, v := range set.List() {
				list[i] = v.(string)
			}
			return &list
		}()
		updated = true
	}

	if d.HasChange(resInstancePoolAttrSecurityGroupIDs) {
		set := d.Get(resInstancePoolAttrSecurityGroupIDs).(*schema.Set)
		instancePool.SecurityGroupIDs = func() *[]string {
			list := make([]string, set.Len())
			for i, v := range set.List() {
				list[i] = v.(string)
			}
			return &list
		}()
		updated = true
	}

	if d.HasChange(resInstancePoolAttrServiceOffering) {
		instanceType, err := client.FindInstanceType(ctx, zone, d.Get(resInstancePoolAttrServiceOffering).(string))
		if err != nil {
			return diag.Errorf("error retrieving instance type: %s", err)
		}
		instancePool.InstanceTypeID = instanceType.ID
		updated = true
	}

	if d.HasChange(resInstancePoolAttrTemplateID) {
		v := d.Get(resInstancePoolAttrTemplateID).(string)
		instancePool.TemplateID = &v
		updated = true
	}

	if d.HasChange(resInstancePoolAttrUserData) {
		v, err := encodeUserData(d.Get(resInstancePoolAttrUserData).(string))
		if err != nil {
			return diag.FromErr(err)
		}
		instancePool.UserData = &v
		updated = true
	}

	if updated {
		if err = client.UpdateInstancePool(ctx, zone, instancePool); err != nil {
			return diag.FromErr(err)
		}
	}

	if d.HasChange(resInstancePoolAttrSize) {
		if err = instancePool.Scale(ctx, int64(d.Get(resInstancePoolAttrSize).(int))); err != nil {
			return diag.FromErr(err)
		}
	}

	log.Printf("[DEBUG] %s: update finished successfully", resourceInstancePoolIDString(d))

	return resourceInstancePoolRead(ctx, d, meta)
}

func resourceInstancePoolDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning delete", resourceInstancePoolIDString(d))

	zone := d.Get(resInstancePoolAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	err := client.DeleteInstancePool(ctx, zone, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceInstancePoolIDString(d))

	return nil
}

func resourceInstancePoolApply(ctx context.Context, client *egoscale.Client, d *schema.ResourceData, instancePool *exov2.InstancePool) diag.Diagnostics {
	if instancePool.AntiAffinityGroupIDs != nil {
		antiAffinityGroupIDs := make([]string, len(*instancePool.AntiAffinityGroupIDs))
		for i, id := range *instancePool.AntiAffinityGroupIDs {
			antiAffinityGroupIDs[i] = id
		}
		if err := d.Set(resInstancePoolAttrAffinityGroupIDs, antiAffinityGroupIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(resInstancePoolAttrDeployTargetID, defaultString(instancePool.DeployTargetID, "")); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resInstancePoolAttrDescription, defaultString(instancePool.Description, "")); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resInstancePoolAttrDiskSize, *instancePool.DiskSize); err != nil {
		return diag.FromErr(err)
	}

	if instancePool.ElasticIPIDs != nil {
		elasticIPIDs := make([]string, len(*instancePool.ElasticIPIDs))
		for i, id := range *instancePool.ElasticIPIDs {
			elasticIPIDs[i] = id
		}
		if err := d.Set(resInstancePoolAttrElasticIPIDs, elasticIPIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(resInstancePoolAttrInstancePrefix, defaultString(instancePool.InstancePrefix, "")); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resInstancePoolAttrIPv6, defaultBool(instancePool.IPv6Enabled, false)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resInstancePoolAttrKeyPair, instancePool.SSHKey); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resInstancePoolAttrName, instancePool.Name); err != nil {
		return diag.FromErr(err)
	}

	if instancePool.PrivateNetworkIDs != nil {
		privateNetworkIDs := make([]string, len(*instancePool.PrivateNetworkIDs))
		for i, id := range *instancePool.PrivateNetworkIDs {
			privateNetworkIDs[i] = id
		}
		if err := d.Set(resInstancePoolAttrNetworkIDs, privateNetworkIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if instancePool.SecurityGroupIDs != nil {
		securityGroupIDs := make([]string, len(*instancePool.SecurityGroupIDs))
		for i, id := range *instancePool.SecurityGroupIDs {
			securityGroupIDs[i] = id
		}
		if err := d.Set(resInstancePoolAttrSecurityGroupIDs, securityGroupIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	instanceType, err := client.GetInstanceType(
		ctx,
		d.Get(resInstancePoolAttrZone).(string),
		*instancePool.InstanceTypeID,
	)
	if err != nil {
		return diag.Errorf("error retrieving instance type: %s", err)
	}
	instancePool.InstanceTypeID = instanceType.ID
	if err := d.Set(resInstancePoolAttrServiceOffering, strings.ToLower(*instanceType.Size)); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resInstancePoolAttrSize, instancePool.Size); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resInstancePoolAttrState, instancePool.State); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resInstancePoolAttrTemplateID, instancePool.TemplateID); err != nil {
		return diag.FromErr(err)
	}

	if instancePool.UserData != nil {
		userData, err := decodeUserData(*instancePool.UserData)
		if err != nil {
			return diag.Errorf("error decoding user data: %s", err)
		}
		if err := d.Set(resInstancePoolAttrUserData, userData); err != nil {
			return diag.FromErr(err)
		}
	}

	if instancePool.InstanceIDs != nil {
		instanceIDs := make([]string, len(*instancePool.InstanceIDs))
		for i, id := range *instancePool.InstanceIDs {
			instanceIDs[i] = id
		}
		if err := d.Set(resInstancePoolAttrVirtualMachines, instanceIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
