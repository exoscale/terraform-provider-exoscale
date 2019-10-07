package exoscale

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"strings"

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
		Update: resourceComputeUpdate,
		Delete: resourceComputeDelete,
		Exists: resourceComputeExists,

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
	size := d.Get("size").(string)

	// ServiceOffering
	so := d.Get("serviceoffering").(string)
	resp, err := client.RequestWithContext(ctx, &egoscale.ListServiceOfferings{
		Name: so,
	})
	if err != nil {
		return err
	}

	services := resp.(*egoscale.ListServiceOfferingsResponse)
	if len(services.ServiceOffering) != 1 {
		return fmt.Errorf("Unable to find the serviceoffering: %#v", size)
	}
	serviceoffering := services.ServiceOffering[0]

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

	userData, base64Encoded, err := prepareUserData(d, meta, "user_data")
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

	userData, err := base64.StdEncoding.DecodeString(instancePool.UserData)
	if err != nil {
		return err
	}

	if err := d.Set("user_data", userData); err != nil {
		return err
	}

	if err := d.Set("zone", zone.Name); err != nil {
		return err
	}

	if err := d.Set("user_data", userData); err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceInstancePoolIDString(d))

	return resourceInstancePoolApply(d, instancePool)
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
		e := handleNotFound(d, err)
		return d.Id() != "", e
	}

	return true, nil
}

func resourceComputeUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning update", resourceComputeIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	// Get() gives us the new state
	initialState := d.Get("state").(string)
	if d.HasChange("state") {
		o, _ := d.GetChange("state")
		initialState = o.(string)
	}

	if initialState != "Running" && initialState != "Stopped" {
		return fmt.Errorf("VM %s must be either Running or Stopped. got %s", d.Id(), initialState)
	}

	rebootRequired := false
	startRequired := false
	stopRequired := false

	d.Partial(true)

	commands := make([]partialCommand, 0)

	// Update command is synchronous, hence it won't be put with the others
	req := &egoscale.UpdateVirtualMachine{
		ID: id,
	}

	if d.HasChange("display_name") {
		req.DisplayName = d.Get("display_name").(string)
	}

	if d.HasChange("user_data") {
		userData, base64Encoded, err := prepareUserData(d, meta, "user_data")
		if err != nil {
			return err
		}

		req.UserData = userData
		rebootRequired = true

		if err := d.Set("user_data_base64", base64Encoded); err != nil {
			return err
		}
	}

	if d.HasChange("security_groups") {
		rebootRequired = true

		securityGroupIDs := make([]egoscale.UUID, 0)
		if securitySet, ok := d.Get("security_groups").(*schema.Set); ok {
			for _, group := range securitySet.List() {
				sg, err := getSecurityGroup(ctx, client, group.(string))
				if err != nil {
					return err
				}
				securityGroupIDs = append(securityGroupIDs, *sg.ID)
			}
		}

		if len(securityGroupIDs) == 0 {
			return errors.New("a Compute instance must have at least one Security Group, none found")
		}

		req.SecurityGroupIDs = securityGroupIDs
	} else if d.HasChange("security_group_ids") {
		rebootRequired = true

		securityGroupIDs := make([]egoscale.UUID, 0)
		if securitySet, ok := d.Get("security_group_ids").(*schema.Set); ok {
			for _, group := range securitySet.List() {
				id, err := egoscale.ParseUUID(group.(string))
				if err != nil {
					return err
				}
				securityGroupIDs = append(securityGroupIDs, *id)
			}
		}

		if len(securityGroupIDs) == 0 {
			return errors.New("a Compute instance must have at least one Security Group, none found")
		}

		req.SecurityGroupIDs = securityGroupIDs
	}

	if d.HasChange("disk_size") {
		o, n := d.GetChange("disk_size")
		oldSize := o.(int)
		newSize := n.(int)

		if oldSize > newSize {
			return fmt.Errorf("A volume can only be expanded. From %dG to %dG is not allowed", oldSize, newSize)
		}

		rebootRequired = true

		volumes, err := client.ListWithContext(ctx, &egoscale.Volume{
			VirtualMachineID: id,
			Type:             "ROOT",
		})
		if err != nil {
			return err
		}
		if len(volumes) != 1 {
			return fmt.Errorf("ROOT volume not found for the VM %s", d.Id())
		}
		volume := volumes[0].(*egoscale.Volume)
		commands = append(commands, partialCommand{
			partial: "disk_size",
			request: &egoscale.ResizeVolume{
				ID:   volume.ID,
				Size: int64(d.Get("disk_size").(int)),
			},
		})
	}

	if d.HasChange("size") {
		o, n := d.GetChange("size")
		oldSize := o.(string)
		newSize := n.(string)
		if !strings.EqualFold(oldSize, newSize) {
			rebootRequired = true
			resp, err := client.RequestWithContext(ctx, &egoscale.ListServiceOfferings{
				Name: newSize,
			})
			if err != nil {
				return err
			}

			services := resp.(*egoscale.ListServiceOfferingsResponse)
			if len(services.ServiceOffering) != 1 {
				return fmt.Errorf("size %q not found", newSize)
			}

			commands = append(commands, partialCommand{
				partial: "size",
				request: &egoscale.ScaleVirtualMachine{
					ID:                id,
					ServiceOfferingID: services.ServiceOffering[0].ID,
				},
			})
		}
	}

	updates, err := updateTags(d, "tags", "userVM")
	if err != nil {
		return err
	}
	for _, update := range updates {
		commands = append(commands, partialCommand{
			partial: "tags",
			request: update,
		})
	}

	if d.HasChange("ip4") {
		activateIP4 := d.Get("ip4").(bool)
		if !activateIP4 {
			return errors.New("the IPv4 address cannot be deactivated")
		}
	}

	if d.HasChange("ip6") {
		activateIP6 := d.Get("ip6").(bool)
		if activateIP6 {
			resp, err := client.Request(&egoscale.ListNics{VirtualMachineID: id})
			if err != nil {
				return err
			}

			nics := resp.(*egoscale.ListNicsResponse)
			if len(nics.Nic) == 0 {
				return fmt.Errorf("The VM has no NIC %v", d.Id())
			}

			commands = append(commands, partialCommand{
				partials: []string{"ip6", "ip6_address", "ip6_cidr"},
				request:  &egoscale.ActivateIP6{NicID: nics.Nic[0].ID},
			})
		} else {
			return errors.New("the IPv6 address cannot be deactivated")
		}
	}

	if d.HasChange("state") {
		switch d.Get("state").(string) {
		case "Running":
			startRequired = true

		case "Stopped":
			stopRequired = true
			rebootRequired = false
			startRequired = false

		default:
			return fmt.Errorf("new state %q cannot be applied", d.Get("state").(string))
		}
	}

	// Stop
	if initialState != "Stopped" && (rebootRequired || stopRequired) {
		resp, err := client.RequestWithContext(ctx, &egoscale.StopVirtualMachine{
			ID: id,
		})
		if err != nil {
			return err
		}

		m := resp.(*egoscale.VirtualMachine)
		if err := resourceComputeApply(d, m); err != nil {
			return err
		}
		d.SetPartial("state")
	}

	// Update, we ignore the result as a full read is require for the user-data/volume
	_, err = client.RequestWithContext(ctx, req)
	if err != nil {
		return err
	}

	if err := resourceComputeRead(d, meta); err != nil {
		return err
	}
	d.SetPartial("user_data")
	d.SetPartial("user_data_base64")
	d.SetPartial("display_name")
	d.SetPartial("security_groups")

	if (initialState == "Running" && rebootRequired) || startRequired {
		commands = append(commands, partialCommand{
			partial: "state",
			request: &egoscale.StartVirtualMachine{
				ID: id,
			},
		})
	}

	for _, cmd := range commands {
		_, err := client.RequestWithContext(ctx, cmd.request)
		if err != nil {
			return err
		}

		d.SetPartial(cmd.partial)
		if cmd.partials != nil {
			for _, partial := range cmd.partials {
				d.SetPartial(partial)
			}
		}
	}

	d.Partial(false)

	log.Printf("[DEBUG] %s: update finished successfully", resourceComputeIDString(d))

	return resourceComputeRead(d, meta)
}

func resourceComputeDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning delete", resourceComputeIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	vm := &egoscale.VirtualMachine{ID: id}

	if err := client.DeleteWithContext(ctx, vm); err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceComputeIDString(d))

	return nil
}

func resourceComputeImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	machine := &egoscale.VirtualMachine{}

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		machine.Name = d.Id()
	} else {
		machine.ID = id
	}

	resp, err := client.GetWithContext(ctx, machine)
	if err != nil {
		if e := handleNotFound(d, err); e != nil {
			return nil, e
		}
		if d.Id() == "" {
			return nil, fmt.Errorf("Failure to import the compute resource: %s", id)
		}
	}

	vm := resp.(*egoscale.VirtualMachine)
	defaultNic := vm.DefaultNic()
	if defaultNic == nil {
		return nil, fmt.Errorf("VM %v has no default NIC", d.Id())
	}
	secondaryIPs := defaultNic.SecondaryIP
	nics := vm.NicsByType("Isolated")

	resources := make([]*schema.ResourceData, 0, 1+len(nics)+len(secondaryIPs))
	resources = append(resources, d)

	for _, secondaryIP := range secondaryIPs {
		resource := resourceSecondaryIPAddress()
		d := resource.Data(nil)
		d.SetType("exoscale_secondary_ipaddress")
		if err := d.Set("compute_id", id); err != nil {
			return nil, err
		}
		secondaryIP.NicID = defaultNic.ID
		secondaryIP.NetworkID = defaultNic.NetworkID
		if err := resourceSecondaryIPAddressApply(d, &secondaryIP); err != nil {
			return nil, err
		}

		resources = append(resources, d)
	}

	for _, nic := range nics {
		resource := resourceNIC()
		d := resource.Data(nil)
		d.SetType("exoscale_nic")
		if err := resourceNICApply(d, nic); err != nil {
			return nil, err
		}

		resources = append(resources, d)
	}

	return resources, nil
}

func resourceInstancePoolApply(d *schema.ResourceData, instancePool *egoscale.InstancePool) error {
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

	return nil
}

func getSecurityGroup(ctx context.Context, client *egoscale.Client, name string) (*egoscale.SecurityGroup, error) {
	sg := &egoscale.SecurityGroup{Name: name}

	resp, err := client.GetWithContext(ctx, sg)
	if err != nil {
		return nil, err
	}

	return resp.(*egoscale.SecurityGroup), nil
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
