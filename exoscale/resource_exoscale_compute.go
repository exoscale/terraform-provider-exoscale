package exoscale

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

const computeHostnameRegexp = `^[a-zA-Z0-9][a-zA-Z0-9\-]+$`

func resourceComputeIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_compute")
}

func resourceCompute() *schema.Resource {
	s := map[string]*schema.Schema{
		"zone": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"template": {
			Type:          schema.TypeString,
			Optional:      true,
			Computed:      true,
			ForceNew:      true,
			ConflictsWith: []string{"template_id"},
		},
		"template_id": {
			Type:          schema.TypeString,
			Optional:      true,
			Computed:      true,
			ForceNew:      true,
			ConflictsWith: []string{"template"},
		},
		"disk_size": {
			Type:         schema.TypeInt,
			Required:     true,
			ValidateFunc: validation.IntAtLeast(10),
		},
		"key_pair": {
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
		},
		"name": {
			Type:       schema.TypeString,
			Computed:   true,
			Deprecated: "use `hostname` attribute instead",
		},
		"display_name": {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		"hostname": {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
			ValidateFunc: validation.StringMatch(
				regexp.MustCompile(computeHostnameRegexp),
				"alphanumeric and hyphen characters",
			),
		},
		"size": {
			Type:     schema.TypeString,
			Optional: true,
			Default:  "Medium",
		},
		"user_data": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "cloud-init configuration",
		},
		"user_data_base64": {
			Type:        schema.TypeBool,
			Computed:    true,
			Description: "was the cloud-init configuration base64 encoded",
		},
		"keyboard": {
			Type:     schema.TypeString,
			Optional: true,
			ValidateFunc: validation.StringInSlice([]string{
				"de", "de-ch", "es", "fi", "fr", "fr-be", "fr-ch", "is",
				"it", "jp", "nl-be", "no", "pt", "uk", "us",
			}, true),
		},
		"reverse_dns": {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: ValidateRegexp(`^.*\.$`),
		},
		"state": {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
			ValidateFunc: validation.StringInSlice([]string{
				"Running", "Stopped",
			}, true),
		},
		"ip4": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Request an IPv4 address on the default NIC",
		},
		"ip6": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Request an IPv6 address on the default NIC",
		},
		"ip_address": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"gateway": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"ip6_address": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"ip6_cidr": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"affinity_group_ids": {
			Type:          schema.TypeSet,
			Optional:      true,
			ForceNew:      true,
			Computed:      true,
			Set:           schema.HashString,
			ConflictsWith: []string{"affinity_groups"},
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"affinity_groups": {
			Type:          schema.TypeSet,
			Optional:      true,
			ForceNew:      true,
			Computed:      true,
			Set:           schema.HashString,
			ConflictsWith: []string{"affinity_group_ids"},
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"security_group_ids": {
			Type:          schema.TypeSet,
			Optional:      true,
			Computed:      true,
			Set:           schema.HashString,
			ConflictsWith: []string{"security_groups"},
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"security_groups": {
			Type:          schema.TypeSet,
			Optional:      true,
			Computed:      true,
			Set:           schema.HashString,
			ConflictsWith: []string{"security_group_ids"},
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"username": {
			Type:       schema.TypeString,
			Computed:   true,
			Deprecated: "broken, use `compute_template` data source `username` attribute",
		},
		"password": {
			Type:      schema.TypeString,
			Computed:  true,
			Sensitive: true,
		},
	}

	addTags(s, "tags")

	return &schema.Resource{
		Schema: s,

		Create: resourceComputeCreate,
		Read:   resourceComputeRead,
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

func resourceComputeCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning create", resourceComputeIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	displayName := d.Get("display_name").(string)
	instanceName := ""
	if _, ok := d.GetOk("hostname"); ok {
		instanceName = d.Get("hostname").(string)
	} else if displayName != "" {
		instanceName = displayName
		if !regexp.MustCompile(computeHostnameRegexp).MatchString(instanceName) {
			return errors.New("if the `hostname` attribute is not set, the `display_name` attribute is used " +
				"instead and its value must be compatible with an instance hostname (contain only alphanumeric " +
				"and hyphen characters)")
		}
	}

	// ServiceOffering
	size := d.Get("size").(string)
	resp, err := client.RequestWithContext(ctx, &egoscale.ListServiceOfferings{
		Name: size,
	})
	if err != nil {
		return err
	}

	services := resp.(*egoscale.ListServiceOfferingsResponse)
	if len(services.ServiceOffering) != 1 {
		return fmt.Errorf("Unable to find the size: %#v", size)
	}
	service := services.ServiceOffering[0].ID

	// XXX Use Generic Get...
	zoneName := d.Get("zone").(string)
	zone, err := getZoneByName(ctx, client, zoneName)
	if err != nil {
		return err
	}

	username := ""
	templateID := ""
	_, byName := d.GetOk("template")
	_, byID := d.GetOk("template_id")
	if !byName && !byID {
		return errors.New("either template or template_id must be specified")
	}

	if byName {
		resp, err = client.GetWithContext(ctx, &egoscale.ListTemplates{
			ZoneID:         zone.ID,
			Name:           d.Get("template").(string),
			TemplateFilter: "featured",
		})
		if err != nil {
			return err
		}

		template := resp.(*egoscale.Template)
		templateID = template.ID.String()
		if name, ok := template.Details["username"]; username == "" && ok {
			username = name
		}
	} else {
		templateID = d.Get("template_id").(string)
	}

	if username == "" {
		log.Printf("[INFO] %s: username not found in the template details, falling back to `root`",
			resourceComputeIDString(d))
		username = "root"
	}

	diskSize := int64(d.Get("disk_size").(int))

	// Affinity Groups
	var affinityGroups []string
	if affinitySet, ok := d.Get("affinity_groups").(*schema.Set); ok {
		affinityGroups = make([]string, affinitySet.Len())
		for i, group := range affinitySet.List() {
			affinityGroups[i] = group.(string)
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

	// Security Groups
	var securityGroups []string
	if securitySet, ok := d.Get("security_groups").(*schema.Set); ok {
		securityGroups = make([]string, securitySet.Len())
		for i, group := range securitySet.List() {
			securityGroups[i] = group.(string)
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

	userData, base64Encoded, err := prepareUserData(d, meta, "user_data")
	if err != nil {
		return err
	}

	if err := d.Set("user_data_base64", base64Encoded); err != nil {
		return err
	}
	startVM := d.Get("state").(string) != "Stopped"

	details := make(map[string]string)
	details["ip4"] = strconv.FormatBool(d.Get("ip4").(bool))
	details["ip6"] = strconv.FormatBool(d.Get("ip6").(bool))

	req := &egoscale.DeployVirtualMachine{
		Name:               instanceName,
		DisplayName:        displayName,
		RootDiskSize:       int64(diskSize),
		KeyPair:            d.Get("key_pair").(string),
		Keyboard:           d.Get("keyboard").(string),
		UserData:           userData,
		ServiceOfferingID:  service,
		TemplateID:         egoscale.MustParseUUID(templateID),
		ZoneID:             zone.ID,
		AffinityGroupIDs:   affinityGroupIDs,
		AffinityGroupNames: affinityGroups,
		SecurityGroupIDs:   securityGroupIDs,
		SecurityGroupNames: securityGroups,
		Details:            details,
		StartVM:            &startVM,
	}

	resp, err = client.RequestWithContext(ctx, req)
	if err != nil {
		return err
	}

	/* Copy VM to our struct */
	machine := resp.(*egoscale.VirtualMachine)
	d.SetId(machine.ID.String())

	if reverseDNS := d.Get("reverse_dns").(string); reverseDNS != "" {
		_, err := client.RequestWithContext(ctx, &egoscale.UpdateReverseDNSForVirtualMachine{
			DomainName: reverseDNS,
			ID:         machine.ID,
		})
		if err != nil {
			return err
		}
	}

	cmd, err := createTags(d, "tags", machine.ResourceType())
	if err != nil {
		return err
	}

	if cmd != nil {
		if err := client.BooleanRequestWithContext(ctx, cmd); err != nil {
			// Attempting to destroy the freshly created machine
			if e := client.DeleteWithContext(ctx, machine); e != nil {
				log.Printf("[WARNING] Failure to create the tags, but the machine was deployed. %v", e)
			}

			return err
		}
	}

	// Connection info
	password := ""
	if machine.PasswordEnabled {
		password = machine.Password
	}

	if err := d.Set("username", username); err != nil {
		return err
	}
	if err := d.Set("password", password); err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: create finished successfully", resourceComputeIDString(d))

	return resourceComputeRead(d, meta)
}

func resourceComputeExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return false, err
	}

	machine := &egoscale.VirtualMachine{ID: id}

	// The CS API returns an error if it doesn't exist
	_, err = client.GetWithContext(ctx, machine)
	if err != nil {
		e := handleNotFound(d, err)
		return d.Id() != "", e
	}

	return true, nil
}

func resourceComputeRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning read", resourceComputeIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	machine := &egoscale.VirtualMachine{ID: id}
	resp, err := client.GetWithContext(ctx, machine)
	if err != nil {
		return handleNotFound(d, err)
	}

	machine = resp.(*egoscale.VirtualMachine)

	// user_data
	resp, err = client.RequestWithContext(ctx, &egoscale.GetVirtualMachineUserData{
		VirtualMachineID: id,
	})
	if err != nil {
		return err
	}
	vmUserData := resp.(*egoscale.VirtualMachineUserData)
	userData := vmUserData.UserData

	// When the data wasn't already encoded, decode it.
	base64Encoded := d.Get("user_data_base64").(bool)
	if !base64Encoded {
		userData, err = vmUserData.Decode()
		if err != nil {
			return err
		}
	}

	if err := d.Set("user_data", userData); err != nil {
		return err
	}

	// disk_size
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
	volumeGib := volume.Size >> 30 // B to GiB
	if err := d.Set("disk_size", volumeGib); err != nil {
		return err
	}

	// connection info
	username := d.Get("username").(string)
	if username == "" {
		username = getSSHUsername(machine.TemplateName)
		if err := d.Set("username", username); err != nil {
			return err
		}
	}

	if err := reverseDNSApply(ctx, d, client); err != nil {
		return err
	}

	password := d.Get("password").(string)
	if machine.PasswordEnabled && password == "" {
		resp, err := client.RequestWithContext(ctx, &egoscale.GetVMPassword{
			ID: machine.ID,
		})
		if err != nil {
			if r, ok := err.(*egoscale.ErrorResponse); ok {
				if r.ErrorCode != egoscale.ParamError && r.ErrorCode != 4350 {
					return err
				}
			} else {
				return err
			}
		} else {
			pwd := resp.(*egoscale.Password)
			// XXX https://cwiki.apache.org/confluence/pages/viewpage.action?pageId=34014652
			password = fmt.Sprintf("base64:%s", pwd.EncryptedPassword)
			if err := d.Set("password", password); err != nil {
				return err
			}
		}
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceComputeIDString(d))

	return resourceComputeApply(d, machine)
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

	if d.HasChange("hostname") {
		req.Name = d.Get("hostname").(string)
		rebootRequired = true
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

	if d.HasChange("reverse_dns") {
		cmd := partialCommand{
			partial: "reverse_dns",
			request: &egoscale.DeleteReverseDNSFromVirtualMachine{ID: id},
		}

		if reverseDNS := d.Get("reverse_dns").(string); reverseDNS != "" {
			cmd.request = &egoscale.UpdateReverseDNSForVirtualMachine{
				DomainName: reverseDNS,
				ID:         id,
			}
		}

		commands = append(commands, cmd)
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
	d.SetPartial("hostname")
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
	log.Printf("[DEBUG] %s: beginning import", resourceComputeIDString(d))

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

	if err := reverseDNSApply(ctx, d, client); err != nil {
		return nil, err
	}

	resources := make([]*schema.ResourceData, 0, 1+len(nics)+len(secondaryIPs))
	resources = append(resources, d)

	for _, secondaryIP := range secondaryIPs {
		log.Printf("[DEBUG] %s: importing exoscale_secondary_ipaddress resource (ID = %s)",
			resourceComputeIDString(d),
			secondaryIP.ID.String())

		resource := resourceSecondaryIPAddress()
		d := resource.Data(nil)
		d.SetType("exoscale_secondary_ipaddress")
		if err := d.Set("compute_id", id.String()); err != nil {
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
		log.Printf("[DEBUG] %s: importing exoscale_nic resource (ID = %s)",
			resourceComputeIDString(d),
			nic.ID.String())

		resource := resourceNIC()
		d := resource.Data(nil)
		d.SetType("exoscale_nic")
		if err := resourceNICApply(d, nic); err != nil {
			return nil, err
		}

		resources = append(resources, d)
	}

	log.Printf("[DEBUG] %s: import finished successfully", resourceComputeIDString(d))

	return resources, nil
}

func resourceComputeApply(d *schema.ResourceData, machine *egoscale.VirtualMachine) error {
	// This should go away once the attribute has been phased out
	if err := d.Set("name", machine.Name); err != nil {
		return err
	}
	if err := d.Set("hostname", machine.Name); err != nil {
		return err
	}
	if err := d.Set("display_name", machine.DisplayName); err != nil {
		return err
	}
	if err := d.Set("key_pair", machine.KeyPair); err != nil {
		return err
	}
	if err := d.Set("size", machine.ServiceOfferingName); err != nil {
		return err
	}
	if err := d.Set("template", machine.TemplateName); err != nil {
		return err
	}
	if err := d.Set("template_id", machine.TemplateID.String()); err != nil {
		return err
	}
	if err := d.Set("zone", machine.ZoneName); err != nil {
		return err
	}

	// don't converge state for migrating instances
	state := machine.State
	if state == "Migrating" {
		state = "Running"
	}
	if err := d.Set("state", state); err != nil {
		return err
	}

	d.Set("ip4", false)      // nolint: errcheck
	d.Set("ip6", false)      // nolint: errcheck
	d.Set("ip_address", "")  // nolint: errcheck
	d.Set("gateway", "")     // nolint: errcheck
	d.Set("ip6_address", "") // nolint: errcheck
	d.Set("ip6_cidr", "")    // nolint: errcheck
	if nic := machine.DefaultNic(); nic != nil {
		d.Set("ip4", true) // nolint: errcheck
		if nic.IPAddress != nil {
			if err := d.Set("ip_address", nic.IPAddress.String()); err != nil {
				return err
			}
		}
		if nic.Gateway != nil {
			if err := d.Set("gateway", nic.Gateway.String()); err != nil {
				return err
			}
		}
		if nic.IP6Address != nil {
			d.Set("ip6", true) // nolint: errcheck
			if err := d.Set("ip6_address", nic.IP6Address.String()); err != nil {
				return err
			}
		}
		if nic.IP6CIDR != nil {
			if err := d.Set("ip6_cidr", nic.IP6CIDR.String()); err != nil {
				return err
			}
		}
	}

	// affinity groups
	affinityGroups := make([]string, len(machine.AffinityGroup))
	affinityGroupIDs := make([]string, len(machine.AffinityGroup))
	for i, ag := range machine.AffinityGroup {
		affinityGroups[i] = ag.Name
		affinityGroupIDs[i] = ag.ID.String()
	}
	if err := d.Set("affinity_groups", affinityGroups); err != nil {
		return err
	}
	if err := d.Set("affinity_group_ids", affinityGroupIDs); err != nil {
		return err
	}

	// security groups
	securityGroups := make([]string, len(machine.SecurityGroup))
	securityGroupIDs := make([]string, len(machine.SecurityGroup))
	for i, sg := range machine.SecurityGroup {
		securityGroups[i] = sg.Name
		securityGroupIDs[i] = sg.ID.String()
	}
	if err := d.Set("security_groups", securityGroups); err != nil {
		return err
	}
	if err := d.Set("security_group_ids", securityGroupIDs); err != nil {
		return err
	}

	// tags
	tags := make(map[string]interface{})
	for _, tag := range machine.Tags {
		tags[tag.Key] = tag.Value
	}
	if err := d.Set("tags", tags); err != nil {
		return err
	}

	// Connection info for the provisioners
	connInfo := map[string]string{
		"type": "ssh",
		"host": d.Get("ip_address").(string),
	}

	if d.Get("password").(string) != "" {
		connInfo["password"] = d.Get("password").(string)
	}

	d.SetConnInfo(connInfo)

	return nil
}

func getSSHUsername(template string) string {
	name := strings.ToLower(template)

	if strings.Contains(name, "ubuntu") {
		return "ubuntu"
	}

	if strings.Contains(name, "centos") {
		return "centos"
	}

	if strings.Contains(name, "redhat") {
		return "cloud-user"
	}

	if strings.Contains(name, "fedora") {
		return "fedora"
	}

	if strings.Contains(name, "coreos") {
		return "core"
	}

	if strings.Contains(name, "debian") {
		return "debian"
	}

	return "root"
}

func getSecurityGroup(ctx context.Context, client *egoscale.Client, name string) (*egoscale.SecurityGroup, error) {
	sg := &egoscale.SecurityGroup{Name: name}

	resp, err := client.GetWithContext(ctx, sg)
	if err != nil {
		return nil, err
	}

	return resp.(*egoscale.SecurityGroup), nil
}

// prepareUserData base64 encode the user-data and gzip it if supported
func prepareUserData(d *schema.ResourceData, meta interface{}, key string) (string, bool, error) {
	userData := d.Get(key).(string)

	// template_cloudinit_config alows to gzip but not base64, prevent such case
	if len(userData) > 2 && userData[0] == '\x1f' && userData[1] == '\x8b' {
		return "", false, errors.New("user_data appears to be gzipped: it should be left raw, or also be base64 encoded")
	}

	// If the data is already base64 encoded, do nothing.
	_, err := base64.StdEncoding.DecodeString(userData)
	if err == nil {
		return userData, true, nil
	}

	byteUserData := []byte(userData)

	if meta.(BaseConfig).gzipUserData {
		b := new(bytes.Buffer)
		gz := gzip.NewWriter(b)

		if _, err := gz.Write(byteUserData); err != nil {
			return "", false, err
		}
		if err := gz.Flush(); err != nil {
			return "", false, err
		}
		if err := gz.Close(); err != nil {
			return "", false, err
		}

		byteUserData = b.Bytes()
	}
	return base64.StdEncoding.EncodeToString(byteUserData), false, nil
}

func reverseDNSApply(ctx context.Context, d *schema.ResourceData, client *egoscale.Client) error {
	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.RequestWithContext(
		ctx,
		&egoscale.QueryReverseDNSForVirtualMachine{
			ID: id,
		})
	if err != nil {
		return err
	}
	vm := resp.(*egoscale.VirtualMachine)

	if nic := vm.DefaultNic(); nic != nil && len(nic.ReverseDNS) > 0 {
		if err := d.Set("reverse_dns", nic.ReverseDNS[0].DomainName); err != nil {
			return err
		}
	}

	return nil
}
