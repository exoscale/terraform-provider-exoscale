package exoscale

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceInstancePoolIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_instance_pool")
}

func resourceInstancePool() *schema.Resource {
	s := map[string]*schema.Schema{
		"name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"template_id": {
			Type:     schema.TypeString,
			Required: true,
		},
		"size": {
			Type:     schema.TypeInt,
			Required: true,
		},
		"zone": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"service_offering": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
			ValidateFunc: func(val interface{}, key string) (warns []string, errs []error) {
				v := val.(string)
				if strings.ContainsAny(v, "ABCDEFGHIJKLMNOPQRSTUVWXYZ") {
					errs = append(errs, fmt.Errorf("%q must be lowercase, got: %q", key, v))
				}

				return
			},
		},
		"key_pair": {
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
		},
		"disk_size": {
			Type:     schema.TypeInt,
			Computed: true,
			Optional: true,
			ForceNew: true,
		},
		"description": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"user_data": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"security_group_ids": {
			Type:     schema.TypeSet,
			Optional: true,
			Set:      schema.HashString,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"network_ids": {
			Type:     schema.TypeSet,
			Optional: true,
			Set:      schema.HashString,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"state": {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		"virtual_machines": {
			Type:     schema.TypeSet,
			Optional: true,
			Computed: true,
			Set:      schema.HashString,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
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

	name := d.Get("name").(string)
	description := d.Get("description").(string)

	size := d.Get("size").(int)

	diskSize := d.Get("disk_size").(int)

	resp, err := client.GetWithContext(ctx, &egoscale.ServiceOffering{
		Name: d.Get("service_offering").(string),
	})
	if err != nil {
		return err
	}
	serviceOffering := resp.(*egoscale.ServiceOffering)

	zoneName := d.Get("zone").(string)
	zone, err := getZoneByName(ctx, client, zoneName)
	if err != nil {
		return err
	}

	var securityGroupIDs []egoscale.UUID
	if securityIDSet, ok := d.Get("security_group_ids").(*schema.Set); ok {
		securityGroupIDs = make([]egoscale.UUID, securityIDSet.Len())
		for i, group := range securityIDSet.List() {
			id, err := egoscale.ParseUUID(group.(string))
			if err != nil {
				return err
			}
			securityGroupIDs[i] = *id
		}
	}

	var networkIDs []egoscale.UUID
	if networkIDSet, ok := d.Get("network_ids").(*schema.Set); ok {
		networkIDs = make([]egoscale.UUID, networkIDSet.Len())
		for i, group := range networkIDSet.List() {
			id, err := egoscale.ParseUUID(group.(string))
			if err != nil {
				return err
			}
			networkIDs[i] = *id
		}
	}

	userData := base64.StdEncoding.EncodeToString([]byte(d.Get("user_data").(string)))

	req := &egoscale.CreateInstancePool{
		Name:              name,
		Description:       description,
		KeyPair:           d.Get("key_pair").(string),
		UserData:          userData,
		ServiceOfferingID: serviceOffering.ID,
		TemplateID:        egoscale.MustParseUUID(d.Get("template_id").(string)),
		ZoneID:            zone.ID,
		SecurityGroupIDs:  securityGroupIDs,
		NetworkIDs:        networkIDs,
		Size:              size,
		RootDiskSize:      diskSize,
	}

	resp, err = client.RequestWithContext(ctx, req)
	if err != nil {
		return err
	}

	instancePool := resp.(*egoscale.CreateInstancePoolResponse)
	d.SetId(instancePool.ID.String())

	log.Printf("[DEBUG] %s: create finished successfully", resourceInstancePoolIDString(d))

	return resourceInstancePoolRead(d, meta)
}

func resourceInstancePoolRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning read", resourceInstancePoolIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	resp, err := client.RequestWithContext(ctx, egoscale.ListZones{})
	if err != nil {
		return err
	}
	zones := resp.(*egoscale.ListZonesResponse).Zone

	for _, zone := range zones {
		instancePool, err := getInstancePoolByID(ctx, client, egoscale.MustParseUUID(d.Id()), zone.ID)
		if err != nil {
			continue
		}

		log.Printf("[DEBUG] %s: read finished successfully", resourceInstancePoolIDString(d))

		return resourceInstancePoolApply(ctx, client, d, instancePool)
	}

	return fmt.Errorf("Instance pool %q not found", d.Id())
}

func findInstancePool(ctx context.Context, d *schema.ResourceData, meta interface{}) (*egoscale.InstancePool, error) {
	client := GetComputeClient(meta)

	resp, err := client.RequestWithContext(ctx, egoscale.ListZones{})
	if err != nil {
		return nil, err
	}
	zones := resp.(*egoscale.ListZonesResponse).Zone

	var instancePool *egoscale.InstancePool
	for _, zone := range zones {
		get := egoscale.GetInstancePool{
			ID:     egoscale.MustParseUUID(d.Id()),
			ZoneID: zone.ID,
		}
		resp, err := client.RequestWithContext(ctx, get)
		if csError, ok := err.(*egoscale.ErrorResponse); ok && csError.ErrorCode == egoscale.NotFound {
			continue
		} else if ok && csError.ErrorCode != egoscale.NotFound {
			return nil, err
		}

		instancePool = &resp.(*egoscale.GetInstancePoolResponse).InstancePools[0]
	}
	if instancePool == nil {
		return nil, fmt.Errorf("Instance pool %q not found", d.Id())
	}

	return instancePool, nil
}

func resourceInstancePoolExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	if _, err := findInstancePool(ctx, d, meta); err != nil {
		return false, err
	}

	return true, nil
}

func resourceInstancePoolUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning update", resourceInstancePoolIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	zone, err := getZoneByName(ctx, client, d.Get("zone").(string))
	if err != nil {
		return err
	}

	req := &egoscale.UpdateInstancePool{
		ID:     id,
		ZoneID: zone.ID,
	}

	d.Partial(true)

	if d.HasChange("name") {
		req.Name = d.Get("name").(string)
		d.SetPartial("name")
	}

	if d.HasChange("description") {
		req.Description = d.Get("description").(string)
		d.SetPartial("description")
	}

	if d.HasChange("template_id") {
		req.TemplateID = egoscale.MustParseUUID(d.Get("template_id").(string))
		d.SetPartial("template_id")
	}

	var userData string
	if d.HasChange("user_data") {
		userData = base64.StdEncoding.EncodeToString([]byte(d.Get("user_data").(string)))
		req.UserData = userData
		d.SetPartial("user_data")
	}

	_, err = client.RequestWithContext(ctx, req)
	if err != nil {
		return err
	}

	if d.HasChange("size") {
		scale := &egoscale.ScaleInstancePool{
			ID:     id,
			ZoneID: zone.ID,
			Size:   d.Get("size").(int),
		}

		d.SetPartial("size")

		_, err = client.RequestWithContext(ctx, scale)
		if err != nil {
			return err
		}
	}

	d.Partial(false)

	log.Printf("[DEBUG] %s: update finished successfully", resourceInstancePoolIDString(d))

	return resourceInstancePoolRead(d, meta)
}

func resourceInstancePoolDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning delete", resourceInstancePoolIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	zone, err := getZoneByName(ctx, client, d.Get("zone").(string))
	if err != nil {
		return err
	}

	req := &egoscale.DestroyInstancePool{
		ID:     id,
		ZoneID: zone.ID,
	}

	if _, err := client.RequestWithContext(ctx, req); err != nil {
		return err
	}

	get := &egoscale.GetInstancePool{
		ID:     id,
		ZoneID: zone.ID,
	}
	for c := time.Tick(time.Second * 10); ; { // nolint: staticcheck
		_, err := client.RequestWithContext(ctx, get)
		if csError, ok := err.(*egoscale.ErrorResponse); ok && csError.ErrorCode == egoscale.NotFound {
			break
		} else if ok && csError.ErrorCode != egoscale.NotFound {
			return err
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("Context timeout after: %v", d.Timeout(schema.TimeoutDelete))
		case <-c:
			continue
		}
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceInstancePoolIDString(d))

	return nil
}

func resourceInstancePoolApply(ctx context.Context, client *egoscale.Client, d *schema.ResourceData, instancePool *egoscale.InstancePool) error {
	if err := d.Set("name", instancePool.Name); err != nil {
		return err
	}
	if err := d.Set("description", instancePool.Description); err != nil {
		return err
	}
	if err := d.Set("key_pair", instancePool.KeyPair); err != nil {
		return err
	}
	if err := d.Set("size", instancePool.Size); err != nil {
		return err
	}

	if err := d.Set("disk_size", instancePool.RootDiskSize); err != nil {
		return err
	}

	if err := d.Set("state", instancePool.State); err != nil {
		return err
	}
	if err := d.Set("template_id", instancePool.TemplateID.String()); err != nil {
		return err
	}

	resp, err := client.GetWithContext(ctx, &egoscale.ServiceOffering{
		ID: instancePool.ServiceOfferingID,
	})
	if err != nil {
		return err
	}
	service := resp.(*egoscale.ServiceOffering)
	if err := d.Set("service_offering", strings.ToLower(service.Name)); err != nil {
		return err
	}

	resp, err = client.GetWithContext(ctx, egoscale.Zone{
		ID: instancePool.ZoneID,
	})
	if err != nil {
		return err
	}

	zone := resp.(*egoscale.Zone)

	if err := d.Set("zone", zone.Name); err != nil {
		return err
	}

	userData, err := base64.StdEncoding.DecodeString(instancePool.UserData)
	if err != nil {
		return err
	}

	if err := d.Set("user_data", string(userData)); err != nil {
		return err
	}

	securityGroupIDs := make([]string, len(instancePool.SecurityGroupIDs))
	for i, sg := range instancePool.SecurityGroupIDs {
		securityGroupIDs[i] = sg.String()
	}
	if err := d.Set("security_group_ids", securityGroupIDs); err != nil {
		return err
	}

	networkIDs := make([]string, len(instancePool.NetworkIDs))
	for i, n := range instancePool.NetworkIDs {
		networkIDs[i] = n.String()
	}
	if err := d.Set("network_ids", networkIDs); err != nil {
		return err
	}

	virtualMachines := make([]string, len(instancePool.VirtualMachines))
	for i, vm := range instancePool.VirtualMachines {
		resp, err := client.GetWithContext(ctx, &egoscale.VirtualMachine{ID: vm.ID})
		if err != nil {
			return err
		}
		v := resp.(*egoscale.VirtualMachine)
		virtualMachines[i] = v.Name
	}
	if err := d.Set("virtual_machines", virtualMachines); err != nil {
		return err
	}

	return nil
}

func getInstancePoolByID(ctx context.Context, client *egoscale.Client, id, zone *egoscale.UUID) (*egoscale.InstancePool, error) {
	resp, err := client.RequestWithContext(ctx, egoscale.GetInstancePool{
		ID:     id,
		ZoneID: zone,
	})
	if err != nil {
		return nil, err
	}
	r := resp.(*egoscale.GetInstancePoolResponse)

	return &r.InstancePools[0], nil
}
