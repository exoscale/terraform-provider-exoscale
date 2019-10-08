package exoscale

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceInstancePoolIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_compute_instance_pool")
}

func resourceInstancePool() *schema.Resource {
	s := map[string]*schema.Schema{
		"zone": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"template": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"size": {
			Type:         schema.TypeInt,
			Required:     true,
			ValidateFunc: validation.IntAtLeast(1),
		},
		"key_pair": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"name": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"description": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"serviceoffering": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"user_data": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "cloud-init configuration",
		},
		"state": {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
			// ValidateFunc: validation.StringInSlice([]string{
			// 	"Running", "Stopped",
			// }, true),
		},
		"affinity_group_ids": {
			Type:     schema.TypeSet,
			Optional: true,
			ForceNew: true,
			Computed: true,
			Set:      schema.HashString,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"security_group_ids": {
			Type:     schema.TypeSet,
			Optional: true,
			Computed: true,
			Set:      schema.HashString,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"networks_ids": {
			Type:     schema.TypeSet,
			Optional: true,
			Computed: true,
			Set:      schema.HashString,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
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
			State: resourceComputeImport,
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

	// Instance pool size
	size := d.Get("size").(int)

	// ServiceOffering
	resp, err := client.GetWithContext(ctx, &egoscale.ServiceOffering{
		Name: d.Get("serviceoffering").(string),
	})
	if err != nil {
		return err
	}
	serviceoffering := resp.(*egoscale.ServiceOffering)

	// XXX Use Generic Get...
	zoneName := d.Get("zone").(string)
	zone, err := getZoneByName(ctx, client, zoneName)
	if err != nil {
		return err
	}

	template, err := getTemplateByName(ctx, client, zone.ID, d.Get("template").(string), "featured")
	if err != nil {
		template, err = getTemplateByName(ctx, client, zone.ID, d.Get("template").(string), "self")
		if err != nil {
			return err
		}
	}

	var affinityGroupIDs []egoscale.UUID
	if affinityIDSet, ok := d.Get("affinity_group_ids").(*schema.Set); ok {
		affinityGroupIDs = make([]egoscale.UUID, affinityIDSet.Len())
		for i, group := range affinityIDSet.List() {
			id, err := egoscale.ParseUUID(group.(string))
			if err != nil {
				return err
			}
			affinityGroupIDs[i] = *id
		}
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
	if networkIDSet, ok := d.Get("networks_ids").(*schema.Set); ok {
		networkIDs = make([]egoscale.UUID, networkIDSet.Len())
		for i, group := range networkIDSet.List() {
			id, err := egoscale.ParseUUID(group.(string))
			if err != nil {
				return err
			}
			networkIDs[i] = *id
		}
	}

	userData, _, err := prepareUserData(d, meta, "user_data")
	if err != nil {
		return err
	}

	req := &egoscale.CreateInstancePool{
		Name:              name,
		Description:       description,
		KeyPair:           d.Get("key_pair").(string),
		UserData:          userData,
		ServiceOfferingID: serviceoffering.ID,
		TemplateID:        template.ID,
		ZoneID:            zone.ID,
		AffinityGroupIDs:  affinityGroupIDs,
		SecurityGroupIDs:  securityGroupIDs,
		NetworkIDs:        networkIDs,
		Size:              size,
	}

	resp, err = client.RequestWithContext(ctx, req)
	if err != nil {
		return err
	}

	instancePool := resp.(*egoscale.CreateInstancePoolResponse)
	d.SetId(instancePool.ID.String())

	if err := d.Set("state", instancePool.State); err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: create finished successfully", resourceInstancePoolIDString(d))

	return resourceInstancePoolRead(d, meta)
}

func resourceInstancePoolRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning read", resourceInstancePoolIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
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

	pool := &egoscale.GetInstancePool{ID: id, ZoneID: zone.ID}
	resp, err := client.RequestWithContext(ctx, pool)
	if err != nil {
		return handleNotFound(d, err)
	}

	instancePool := &resp.(*egoscale.GetInstancePoolsResponse).ListInstancePoolsResponse[0]

	log.Printf("[DEBUG] %s: read finished successfully", resourceInstancePoolIDString(d))

	return resourceInstancePoolApply(ctx, client, d, instancePool)
}

func resourceInstancePoolExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return false, err
	}

	zone, err := getZoneByName(ctx, client, d.Get("zone").(string))
	if err != nil {
		return false, err
	}

	_, err = client.RequestWithContext(ctx, &egoscale.GetInstancePool{
		ID:     id,
		ZoneID: zone.ID,
	})
	if err != nil {
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

	if d.HasChange("description") {
		req.Description = d.Get("description").(string)
	}

	var userData string
	if d.HasChange("user_data") {
		userData, _, err = prepareUserData(d, meta, "user_data")
		if err != nil {
			return err
		}

		req.UserData = userData
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

		_, err = client.RequestWithContext(ctx, scale)
		if err != nil {
			return err
		}
	}

	log.Printf("[DEBUG] %s: update finished successfully", resourceInstancePoolIDString(d))

	return resourceComputeRead(d, meta)
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

	log.Printf("[DEBUG] %s: delete finished successfully", resourceInstancePoolIDString(d))

	return nil
}

func resourceInstancePoolImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	zone, err := getZoneByName(ctx, client, d.Get("zone").(string))
	if err != nil {
		return nil, err
	}

	instancePool, err := getInstancePoolByName(ctx, client, d.Id(), zone.ID)
	if err != nil {
		return nil, err
	}

	resources := make([]*schema.ResourceData, 0, 1)
	resources = append(resources, d)

	resource := new(schema.ResourceData)
	if err := resourceInstancePoolApply(ctx, client, resource, instancePool); err != nil {
		return nil, err
	}
	resources = append(resources, resource)

	return resources, nil
}

func resourceInstancePoolApply(ctx context.Context, client *egoscale.Client, d *schema.ResourceData, instancePool *egoscale.InstancePool) error {
	if err := d.Set("name", instancePool.Name); err != nil {
		return err
	}
	if err := d.Set("description", instancePool.Name); err != nil {
		return err
	}
	if err := d.Set("key_pair", instancePool.KeyPair); err != nil {
		return err
	}
	if err := d.Set("size", instancePool.Size); err != nil {
		return err
	}
	if err := d.Set("state", instancePool.State); err != nil {
		return err
	}

	// service offering
	resp, err := client.GetWithContext(ctx, &egoscale.ServiceOffering{
		ID: instancePool.ServiceOfferingID,
	})
	if err != nil {
		return err
	}
	service := resp.(*egoscale.ServiceOffering)
	if err := d.Set("serviceoffering", service.Name); err != nil {
		return err
	}

	// template
	template, err := getTemplateByName(ctx, client, instancePool.ZoneID, instancePool.TemplateID.String(), "featured")
	if err != nil {
		template, err = getTemplateByName(ctx, client, instancePool.ZoneID, instancePool.TemplateID.String(), "self")
		if err != nil {
			return err
		}
	}

	if err := d.Set("template", template.Name); err != nil {
		return err
	}

	// zone
	zone, err := getZoneByName(ctx, client, instancePool.ZoneID.String())
	if err != nil {
		return err
	}

	if err := d.Set("zone", zone.Name); err != nil {
		return err
	}

	// user data
	userData, err := base64.StdEncoding.DecodeString(instancePool.UserData)
	if err != nil {
		return err
	}

	if err := d.Set("user_data", userData); err != nil {
		return err
	}

	// affinity groups
	affinityGroupIDs := make([]string, len(instancePool.AffinityGroupIDs))
	for i, ag := range instancePool.AffinityGroupIDs {
		affinityGroupIDs[i] = ag.String()
	}
	if err := d.Set("affinity_group_ids", affinityGroupIDs); err != nil {
		return err
	}

	// security groups
	securityGroupIDs := make([]string, len(instancePool.SecurityGroupIDs))
	for i, sg := range instancePool.SecurityGroupIDs {
		securityGroupIDs[i] = sg.String()
	}
	if err := d.Set("security_group_ids", securityGroupIDs); err != nil {
		return err
	}

	// networks
	networksIDs := make([]string, len(instancePool.NetworkIDs))
	for i, n := range instancePool.NetworkIDs {
		networksIDs[i] = n.String()
	}
	if err := d.Set("networks_ids", networksIDs); err != nil {
		return err
	}

	// virtual Machines
	virtualMachines := make([]string, len(instancePool.VirtualMachines))
	for i, vm := range instancePool.VirtualMachines {
		resp, err := client.GetWithContext(ctx, &egoscale.VirtualMachine{ID: vm.ID})
		if err != nil {
			return err
		}
		v := resp.(*egoscale.VirtualMachine)
		virtualMachines[i] = v.Name
	}
	if err := d.Set("networks_ids", networksIDs); err != nil {
		return err
	}

	return nil
}

func getTemplateByName(ctx context.Context, client *egoscale.Client, zoneID *egoscale.UUID, name string, templateFilter string) (*egoscale.Template, error) {
	req := &egoscale.ListTemplates{
		TemplateFilter: templateFilter,
		ZoneID:         zoneID,
	}

	id, errUUID := egoscale.ParseUUID(name)
	if errUUID != nil {
		req.Name = name
	} else {
		req.ID = id
	}

	resp, err := client.ListWithContext(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(resp) == 0 {
		return nil, fmt.Errorf("template %q not found", name)
	}
	if len(resp) == 1 {
		return resp[0].(*egoscale.Template), nil
	}
	return nil, fmt.Errorf("multiple templates found for %q", name)
}

func getInstancePoolByID(ctx context.Context, client *egoscale.Client, id, zone *egoscale.UUID) (*egoscale.InstancePool, error) {
	resp, err := client.RequestWithContext(ctx, egoscale.GetInstancePool{
		ID:     id,
		ZoneID: zone,
	})
	if err != nil {
		return nil, err
	}
	r := resp.(*egoscale.GetInstancePoolsResponse)

	return &r.ListInstancePoolsResponse[0], nil
}

func getInstancePoolByName(ctx context.Context, client *egoscale.Client, name string, zone *egoscale.UUID) (*egoscale.InstancePool, error) {
	instancePools := []egoscale.InstancePool{}

	id, err := egoscale.ParseUUID(name)
	if err == nil {
		return getInstancePoolByID(ctx, client, id, zone)
	}

	resp, err := client.RequestWithContext(ctx, egoscale.ListInstancePool{
		ZoneID: zone,
	})
	if err != nil {
		return nil, err
	}
	r := resp.(*egoscale.ListInstancePoolsResponse)

	for _, i := range r.ListInstancePoolsResponse {
		if i.Name == name {
			instancePools = append(instancePools, i)
		}
	}

	switch count := len(instancePools); {
	case count == 0:
		return nil, fmt.Errorf("not found: %q", name)
	case count > 1:
		return nil, fmt.Errorf("more than one element found: %q", count)
	}

	return &instancePools[0], nil
}
