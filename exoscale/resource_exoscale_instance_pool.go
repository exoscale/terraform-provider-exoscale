package exoscale

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/exoscale/egoscale"
	exov2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

const (
	defaultInstancePoolInstancePrefix = "pool"

	resInstancePoolAttrAffinityGroupIDs = "affinity_group_ids"
	resInstancePoolAttrDescription      = "description"
	resInstancePoolAttrDiskSize         = "disk_size"
	resInstancePoolAttrElasticIPIDs     = "elastic_ip_ids"
	resInstancePoolAttrID               = "id"
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

		Create: resourceInstancePoolCreate,
		Read:   resourceInstancePoolRead,
		Update: resourceInstancePoolUpdate,
		Delete: resourceInstancePoolDelete,
		Exists: resourceInstancePoolExists,

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

func resourceInstancePoolCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning create", resourceInstancePoolIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	instancePool := &exov2.InstancePool{
		Description:    d.Get(resInstancePoolAttrDescription).(string),
		DiskSize:       int64(d.Get(resInstancePoolAttrDiskSize).(int)),
		Name:           d.Get(resInstancePoolAttrName).(string),
		InstancePrefix: d.Get(resInstancePoolAttrInstancePrefix).(string),
		SSHKey:         d.Get(resInstancePoolAttrKeyPair).(string),
		Size:           int64(d.Get(resInstancePoolAttrSize).(int)),
		TemplateID:     d.Get(resInstancePoolAttrTemplateID).(string),
		UserData:       base64.StdEncoding.EncodeToString([]byte(d.Get(resInstancePoolAttrUserData).(string))),
	}

	resp, err := client.GetWithContext(ctx, &egoscale.ServiceOffering{
		Name: d.Get(resInstancePoolAttrServiceOffering).(string),
	})
	if err != nil {
		return err
	}
	instancePool.InstanceTypeID = resp.(*egoscale.ServiceOffering).ID.String()

	if set, ok := d.Get(resInstancePoolAttrAffinityGroupIDs).(*schema.Set); ok {
		instancePool.AntiAffinityGroupIDs = func() []string {
			list := make([]string, set.Len())
			for i, v := range set.List() {
				list[i] = v.(string)
			}
			return list
		}()
	}

	if set, ok := d.Get(resInstancePoolAttrSecurityGroupIDs).(*schema.Set); ok {
		instancePool.SecurityGroupIDs = func() []string {
			list := make([]string, set.Len())
			for i, v := range set.List() {
				list[i] = v.(string)
			}
			return list
		}()
	}

	if set, ok := d.Get(resInstancePoolAttrNetworkIDs).(*schema.Set); ok {
		instancePool.PrivateNetworkIDs = func() []string {
			list := make([]string, set.Len())
			for i, v := range set.List() {
				list[i] = v.(string)
			}
			return list
		}()
	}

	if set, ok := d.Get(resInstancePoolAttrElasticIPIDs).(*schema.Set); ok {
		instancePool.ElasticIPIDs = func() []string {
			list := make([]string, set.Len())
			for i, v := range set.List() {
				list[i] = v.(string)
			}
			return list
		}()
	}

	if enableIPv6 := d.Get(resInstancePoolAttrIPv6).(bool); enableIPv6 {
		instancePool.IPv6Enabled = true
	}

	zone := d.Get(resInstancePoolAttrZone).(string)

	// FIXME: we have to reference the embedded egoscale/v2.Client explicitly
	//  here because there is already a CreateInstancePool() method on the root
	//  egoscale client clashing with the v2 one. This can be changed once we
	//  use API V2-only calls.
	instancePool, err = client.Client.CreateInstancePool(ctx, zone, instancePool)
	if err != nil {
		return err
	}
	d.SetId(instancePool.ID)

	log.Printf("[DEBUG] %s: create finished successfully", resourceInstancePoolIDString(d))

	return resourceInstancePoolRead(d, meta)
}

func resourceInstancePoolRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning read", resourceInstancePoolIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	instancePool, err := findInstancePool(ctx, d, meta)
	if err != nil {
		return err
	}

	if instancePool == nil {
		return fmt.Errorf("Instance Pool %q not found", d.Id())
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceInstancePoolIDString(d))

	return resourceInstancePoolApply(ctx, GetComputeClient(meta), d, instancePool)
}

func resourceInstancePoolExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	instancePool, err := findInstancePool(ctx, d, meta)
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			return false, nil
		}
		return false, err
	}

	if instancePool == nil {
		return false, nil
	}

	return true, nil
}

func resourceInstancePoolUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning update", resourceInstancePoolIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()
	client := GetComputeClient(meta)

	instancePool, err := findInstancePool(ctx, d, meta)
	if err != nil {
		return err
	}

	if instancePool == nil {
		return fmt.Errorf("Instance Pool %q not found", d.Id())
	}

	var (
		updated     bool
		resetFields = make([]interface{}, 0)
	)

	if d.HasChange(resInstancePoolAttrAffinityGroupIDs) {
		set := d.Get(resInstancePoolAttrAffinityGroupIDs).(*schema.Set)
		if set.Len() == 0 {
			instancePool.AntiAffinityGroupIDs = nil
			resetFields = append(resetFields, &instancePool.AntiAffinityGroupIDs)
		} else {
			instancePool.AntiAffinityGroupIDs = func() []string {
				list := make([]string, set.Len())
				for i, v := range set.List() {
					list[i] = v.(string)
				}
				return list
			}()
			updated = true
		}
	}

	if d.HasChange(resInstancePoolAttrDescription) {
		if v := d.Get(resInstancePoolAttrDescription).(string); v == "" {
			instancePool.Description = ""
			resetFields = append(resetFields, &instancePool.Description)
		} else {
			instancePool.Description = d.Get(resInstancePoolAttrDescription).(string)
			updated = true
		}
	}

	if d.HasChange(resInstancePoolAttrDiskSize) {
		instancePool.DiskSize = int64(d.Get(resInstancePoolAttrDiskSize).(int))
		updated = true
	}

	if d.HasChange(resInstancePoolAttrElasticIPIDs) {
		set := d.Get(resInstancePoolAttrElasticIPIDs).(*schema.Set)
		if set.Len() == 0 {
			resetFields = append(resetFields, &instancePool.ElasticIPIDs)
		} else {
			instancePool.ElasticIPIDs = func() []string {
				list := make([]string, set.Len())
				for i, v := range set.List() {
					list[i] = v.(string)
				}
				return list
			}()
			updated = true
		}
	}

	if d.HasChange(resInstancePoolAttrInstancePrefix) {
		instancePool.InstancePrefix = d.Get(resInstancePoolAttrInstancePrefix).(string)
		updated = true
	}

	if d.HasChange(resInstancePoolAttrIPv6) {
		// IPv6 can only be enabled, not disabled.
		if enableIPv6 := d.Get("ipv6").(bool); enableIPv6 {
			instancePool.IPv6Enabled = true
			updated = true
		}
	}

	if d.HasChange(resInstancePoolAttrKeyPair) {
		if v := d.Get(resInstancePoolAttrKeyPair).(string); v == "" {
			instancePool.SSHKey = ""
			resetFields = append(resetFields, &instancePool.SSHKey)
		} else {
			instancePool.SSHKey = d.Get(resInstancePoolAttrKeyPair).(string)
			updated = true
		}
	}

	if d.HasChange(resInstancePoolAttrName) {
		instancePool.Name = d.Get(resInstancePoolAttrName).(string)
		updated = true
	}

	if d.HasChange(resInstancePoolAttrNetworkIDs) {
		set := d.Get(resInstancePoolAttrNetworkIDs).(*schema.Set)
		if set.Len() == 0 {
			instancePool.PrivateNetworkIDs = nil
			resetFields = append(resetFields, &instancePool.PrivateNetworkIDs)
		} else {
			instancePool.PrivateNetworkIDs = func() []string {
				list := make([]string, set.Len())
				for i, v := range set.List() {
					list[i] = v.(string)
				}
				return list
			}()
			updated = true
		}
	}

	if d.HasChange(resInstancePoolAttrSecurityGroupIDs) {
		set := d.Get(resInstancePoolAttrSecurityGroupIDs).(*schema.Set)
		if set.Len() == 0 {
			instancePool.SecurityGroupIDs = nil
			resetFields = append(resetFields, &instancePool.SecurityGroupIDs)
		} else {
			instancePool.SecurityGroupIDs = func() []string {
				list := make([]string, set.Len())
				for i, v := range set.List() {
					list[i] = v.(string)
				}
				return list
			}()
			updated = true
		}
	}

	if d.HasChange(resInstancePoolAttrServiceOffering) {
		resp, err := client.GetWithContext(ctx, &egoscale.ServiceOffering{
			Name: d.Get(resInstancePoolAttrServiceOffering).(string),
		})
		if err != nil {
			return err
		}
		instancePool.InstanceTypeID = resp.(*egoscale.ServiceOffering).ID.String()
		updated = true
	}

	if d.HasChange(resInstancePoolAttrTemplateID) {
		instancePool.TemplateID = d.Get(resInstancePoolAttrTemplateID).(string)
		updated = true
	}

	if d.HasChange(resInstancePoolAttrUserData) {
		instancePool.UserData = base64.StdEncoding.EncodeToString([]byte(d.Get(resInstancePoolAttrUserData).(string)))
		updated = true
	}

	zone := d.Get(resInstancePoolAttrZone).(string)

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	if err = client.UpdateInstancePool(ctx, zone, instancePool); err != nil {
		return err
	}

	if updated {
		if err = client.UpdateInstancePool(ctx, zone, instancePool); err != nil {
			return err
		}
	}

	for _, f := range resetFields {
		if err = instancePool.ResetField(ctx, f); err != nil {
			return err
		}
	}

	if d.HasChange(resInstancePoolAttrSize) {
		if err = instancePool.Scale(ctx, int64(d.Get(resInstancePoolAttrSize).(int))); err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] %s: update finished successfully", resourceInstancePoolIDString(d))

	return resourceInstancePoolRead(d, meta)
}

func resourceInstancePoolDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning delete", resourceInstancePoolIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	zone := d.Get(resInstancePoolAttrZone).(string)

	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	err := client.DeleteInstancePool(ctx, zone, d.Id())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceInstancePoolIDString(d))

	return nil
}

func resourceInstancePoolApply(ctx context.Context, client *egoscale.Client, d *schema.ResourceData, instancePool *exov2.InstancePool) error {
	antiAffinityGroupIDs := make([]string, len(instancePool.AntiAffinityGroupIDs))
	for i, id := range instancePool.AntiAffinityGroupIDs {
		antiAffinityGroupIDs[i] = id
	}
	if err := d.Set(resInstancePoolAttrAffinityGroupIDs, antiAffinityGroupIDs); err != nil {
		return err
	}

	if err := d.Set(resInstancePoolAttrDescription, instancePool.Description); err != nil {
		return err
	}

	if err := d.Set(resInstancePoolAttrDiskSize, instancePool.DiskSize); err != nil {
		return err
	}

	elasticIPIDs := make([]string, len(instancePool.ElasticIPIDs))
	for i, id := range instancePool.ElasticIPIDs {
		elasticIPIDs[i] = id
	}
	if err := d.Set(resInstancePoolAttrElasticIPIDs, elasticIPIDs); err != nil {
		return err
	}

	if err := d.Set(resInstancePoolAttrInstancePrefix, instancePool.InstancePrefix); err != nil {
		return err
	}

	if err := d.Set(resInstancePoolAttrIPv6, instancePool.IPv6Enabled); err != nil {
		return err
	}

	if err := d.Set(resInstancePoolAttrKeyPair, instancePool.SSHKey); err != nil {
		return err
	}

	if err := d.Set(resInstancePoolAttrName, instancePool.Name); err != nil {
		return err
	}

	privateNetworkIDs := make([]string, len(instancePool.PrivateNetworkIDs))
	for i, id := range instancePool.PrivateNetworkIDs {
		privateNetworkIDs[i] = id
	}
	if err := d.Set(resInstancePoolAttrNetworkIDs, privateNetworkIDs); err != nil {
		return err
	}

	securityGroupIDs := make([]string, len(instancePool.SecurityGroupIDs))
	for i, id := range instancePool.SecurityGroupIDs {
		securityGroupIDs[i] = id
	}
	if err := d.Set(resInstancePoolAttrSecurityGroupIDs, securityGroupIDs); err != nil {
		return err
	}

	resp, err := client.GetWithContext(ctx, &egoscale.ServiceOffering{
		ID: egoscale.MustParseUUID(instancePool.InstanceTypeID),
	})
	if err != nil {
		return err
	}
	if err := d.Set(resInstancePoolAttrServiceOffering,
		strings.ToLower(resp.(*egoscale.ServiceOffering).Name)); err != nil {
		return err
	}

	if err := d.Set(resInstancePoolAttrSize, instancePool.Size); err != nil {
		return err
	}

	if err := d.Set(resInstancePoolAttrState, instancePool.State); err != nil {
		return err
	}

	if err := d.Set(resInstancePoolAttrTemplateID, instancePool.TemplateID); err != nil {
		return err
	}

	userData, err := base64.StdEncoding.DecodeString(instancePool.UserData)
	if err != nil {
		return err
	}
	if err := d.Set(resInstancePoolAttrUserData, string(userData)); err != nil {
		return err
	}

	instanceIDs := make([]string, len(instancePool.InstanceIDs))
	for i, id := range instancePool.InstanceIDs {
		instanceIDs[i] = id
	}
	if err := d.Set(resInstancePoolAttrVirtualMachines, instanceIDs); err != nil {
		return err
	}

	return nil
}

func findInstancePool(ctx context.Context, d *schema.ResourceData, meta interface{}) (*exov2.InstancePool, error) {
	client := GetComputeClient(meta)

	if zone, ok := d.GetOk(resInstancePoolAttrZone); ok {
		ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone.(string)))
		instancePool, err := client.GetInstancePool(ctx, zone.(string), d.Id())
		if err != nil {
			return nil, err
		}

		return instancePool, nil
	}

	zones, err := client.ListZones(exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), defaultZone)))
	if err != nil {
		return nil, err
	}

	var instancePool *exov2.InstancePool
	for _, zone := range zones {
		i, err := client.GetInstancePool(
			exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone)),
			zone,
			d.Id())
		if err != nil {
			if errors.Is(err, exoapi.ErrNotFound) {
				continue
			}

			return nil, err
		}

		if i != nil {
			instancePool = i
			if err := d.Set(resInstancePoolAttrZone, zone); err != nil {
				return nil, err
			}
			break
		}
	}

	return instancePool, nil
}
