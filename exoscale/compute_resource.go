package exoscale

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func computeResource() *schema.Resource {
	s := map[string]*schema.Schema{
		"name": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"display_name": {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		"template": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"size": {
			Type:     schema.TypeString,
			Optional: true,
			Default:  "Medium",
			ValidateFunc: validation.StringInSlice([]string{
				"Micro", "Tiny", "Small", "Medium", "Large", "Extra-Large", "Huge",
				"Mega", "Titan",
			}, true),
		},
		"disk_size": {
			Type:         schema.TypeInt,
			Required:     true,
			ValidateFunc: validation.IntAtLeast(10),
		},
		"zone": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"user_data": {
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
			StateFunc: func(v interface{}) string {
				switch v.(type) {
				case string:
					return strconv.FormatInt(int64(hashcode.String(v.(string))), 10)
				default:
					return ""
				}
			},
		},
		"key_pair": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"keyboard": {
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
			ValidateFunc: validation.StringInSlice([]string{
				"de", "de-ch", "es", "fi", "fr", "fr-be", "fr-ch", "is",
				"it", "jp", "nl-be", "no", "pt", "uk", "us",
			}, true),
		},
		"state": {
			Type:     schema.TypeString,
			Optional: true,
			Default:  "Running",
			ValidateFunc: validation.StringInSlice([]string{
				"Starting", "Running", "Stopped", "Destroyed",
				"Expunging", "Migrating", "Error", "Unknown",
				"Shutdowned",
			}, true),
		},
		"ip_address": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"affinity_groups": {
			Type:     schema.TypeSet,
			Optional: true,
			Computed: true,
			Set:      schema.HashString,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"security_groups": {
			Type:     schema.TypeSet,
			Optional: true,
			Computed: true,
			Set:      schema.HashString,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
	}

	addTags(s, "tags")

	return &schema.Resource{
		Create: createCompute,
		Exists: existsCompute,
		Read:   readCompute,
		Update: updateCompute,
		Delete: deleteCompute,

		Importer: &schema.ResourceImporter{
			State: importCompute,
		},

		Schema: s,
	}
}

func createCompute(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)

	displayName := d.Get("display_name").(string)
	hostName := regexp.MustCompile("^[a-zA-Z0-9][a-zA-Z0-9\\-]+$")
	if !hostName.MatchString(displayName) {
		return fmt.Errorf("At creation time, the `display_name` must match a value compatible with the `hostname` (alpha-numeric and hyphens")
	}

	// check that the name doesn't already exists!!

	topo, err := client.GetTopology()
	if err != nil {
		return err
	}

	diskSize := int64(d.Get("disk_size").(int))
	service := topo.Profiles[strings.ToLower(d.Get("size").(string))]

	if service == "" {
		return fmt.Errorf("Invalid service: %s", d.Get("size").(string))
	}

	zoneName := d.Get("zone").(string)
	zone, err := getZoneByName(client, zoneName)
	if err != nil {
		return err
	}

	template := topo.Images[convertTemplateName(d.Get("template").(string))]
	if template == nil {
		return fmt.Errorf("Invalid template: %s", d.Get("template").(string))
	}

	// If the exact diskSize doesn't exist pick the smallest one and go for it
	templateID := template[diskSize]
	if templateID == "" {
		smallestDiskSize := diskSize
		for s := range template {
			if s < smallestDiskSize {
				smallestDiskSize = s
			}
		}

		templateID = template[smallestDiskSize]
		if templateID == "" {
			return fmt.Errorf("Invalid disk size: %d", diskSize)
		}
	}

	var affinityGroups []string
	if affinitySet, ok := d.Get("affinity_groups").(*schema.Set); ok {
		affinityGroups = make([]string, affinitySet.Len())
		for i, group := range affinitySet.List() {
			ag, err := getAffinityGroupID(client, group.(string))
			if err != nil {
				return err
			}
			affinityGroups[i] = ag
		}
	}

	var securityGroups []string
	if securitySet, ok := d.Get("security_groups").(*schema.Set); ok {
		securityGroups = make([]string, securitySet.Len())
		for i, group := range securitySet.List() {
			sg, err := getSecurityGroupID(client, group.(string))
			if err != nil {
				return err
			}
			securityGroups[i] = sg
		}
	}

	userData, err := prepareUserData(d, "user_data")
	if err != nil {
		return err
	}

	startVM := true
	if d.Get("state").(string) != "Running" {
		startVM = false
	}

	req := &egoscale.DeployVirtualMachine{
		Name:              displayName,
		DisplayName:       displayName,
		RootDiskSize:      int64(diskSize),
		KeyPair:           d.Get("key_pair").(string),
		Keyboard:          d.Get("keyboard").(string),
		UserData:          userData,
		ServiceOfferingID: service,
		TemplateID:        templateID,
		ZoneID:            zone.ID,
		AffinityGroupIDs:  affinityGroups,
		SecurityGroupIDs:  securityGroups,
		StartVM:           &startVM,
	}

	resp, err := client.Request(req)
	if err != nil {
		return err
	}

	/* Copy VM to our struct */
	machine := resp.(*egoscale.DeployVirtualMachineResponse).VirtualMachine
	d.SetId(machine.ID)

	if cmd := createTags(d, "tags", machine.ResourceType()); cmd != nil {
		if err := client.BooleanRequest(cmd); err != nil {
			// Attempting to destroy the freshly created machine
			_, e := client.Request(&egoscale.DestroyVirtualMachine{
				ID: machine.ID,
			})

			if e != nil {
				log.Printf("[WARNING] Failure to create the tags, but the machine was deployed. %v", e)
			}

			return err
		}
	}

	return readCompute(d, meta)
}

func existsCompute(d *schema.ResourceData, meta interface{}) (bool, error) {
	_, err := getVirtualMachine(d, meta)

	// The CS API returns an error if it doesn't exist
	if err != nil {
		e := handleNotFound(d, err)
		return d.Id() != "", e
	}

	return true, nil
}

func readCompute(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)

	machine, err := getVirtualMachine(d, meta)
	if err != nil {
		return handleNotFound(d, err)
	}

	// affinity_groups
	affinitySet := schema.NewSet(schema.HashString, nil)
	for _, affinityGroup := range machine.AffinityGroup {
		affinitySet.Add(affinityGroup.Name)
	}
	d.Set("affinity_groups", affinitySet)

	// security_group
	securitySet := schema.NewSet(schema.HashString, nil)
	for _, securityGroup := range machine.SecurityGroup {
		securitySet.Add(securityGroup.Name)
	}
	d.Set("security_groups", securitySet)

	// disk_size
	volume, err := client.GetRootVolumeForVirtualMachine(d.Id())
	if err != nil {
		return err
	}
	d.Set("disk_size", volume.Size>>30) // B to GiB

	return applyCompute(d, *machine)
}

func updateCompute(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)

	initialState := d.Get("state").(string)
	if initialState != "Running" && initialState != "Stopped" {
		return fmt.Errorf("VM %s must be either Running or Stopped. got %s", d.Id(), initialState)
	}

	rebootRequired := false
	startRequired := false
	stopRequired := false

	d.Partial(true)

	// partialCommand represents an update command, it's made of
	// the partial key which is expected to change and the
	// request that has to be run.
	type partialCommand struct {
		partial string
		request egoscale.Command
	}

	commands := make([]partialCommand, 0)

	// Update command is synchronous, hence it won't be put with the others
	req := &egoscale.UpdateVirtualMachine{
		ID: d.Id(),
	}

	if d.HasChange("display_name") {
		req.DisplayName = d.Get("display_name").(string)
	}

	if d.HasChange("security_groups") {
		rebootRequired = true

		securityGroups := make([]string, 0)
		if securitySet, ok := d.Get("security_groups").(*schema.Set); ok {
			for _, group := range securitySet.List() {
				id, err := getSecurityGroupID(client, group.(string))
				if err != nil {
					return err
				}
				securityGroups = append(securityGroups, id)
			}
		}

		if len(securityGroups) == 0 {
			return fmt.Errorf("A VM must have at least one Security Group, none found")
		}

		req.SecurityGroupIDs = securityGroups
	}

	if d.HasChange("disk_size") {
		o, n := d.GetChange("disk_size")
		oldSize := o.(int)
		newSize := n.(int)

		if oldSize > newSize {
			return fmt.Errorf("A volume can only be expanded. From %dG to %dG is not allowed", oldSize, newSize)
		}

		rebootRequired = true
		volume, err := client.GetRootVolumeForVirtualMachine(d.Id())
		if err != nil {
			return err
		}
		commands = append(commands, partialCommand{
			partial: "disk_size",
			request: &egoscale.ResizeVolume{
				ID:   volume.ID,
				Size: int64(d.Get("disk_size").(int)),
			},
		})
	}

	if d.HasChange("size") {
		rebootRequired = true
		services, err := client.Request(&egoscale.ListServiceOfferings{
			Name: d.Get("size").(string),
		})
		if err != nil {
			return err
		}
		commands = append(commands, partialCommand{
			partial: "size",
			request: &egoscale.ScaleVirtualMachine{
				ID:                d.Id(),
				ServiceOfferingID: services.(*egoscale.ListServiceOfferingsResponse).ServiceOffering[0].ID,
			},
		})
	}

	if d.HasChange("affinity_groups") {
		rebootRequired = true
		if affinitySet, ok := d.Get("affinity_groups").(*schema.Set); ok {
			affinityGroups := make([]string, affinitySet.Len())
			for i, group := range affinitySet.List() {
				id, err := getAffinityGroupID(client, group.(string))
				if err != nil {
					return err
				}
				affinityGroups[i] = id
			}
			commands = append(commands, partialCommand{
				partial: "affinity_groups",
				request: &egoscale.UpdateVMAffinityGroup{
					ID:               d.Id(),
					AffinityGroupIDs: affinityGroups,
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

	if d.HasChange("state") {
		switch d.Get("state").(string) {
		case "Running":
			startRequired = true
		case "Stopped":
			stopRequired = true
		default:
			return fmt.Errorf("The new state cannot applied, %s. Do it manually", d.Get("state").(string))
		}
	}

	// Stop
	if initialState != "Stopped" && (rebootRequired || stopRequired) {
		resp, err := client.Request(&egoscale.StopVirtualMachine{
			ID: d.Id(),
		})
		if err != nil {
			return err
		}

		m := resp.(*egoscale.StopVirtualMachineResponse).VirtualMachine
		applyCompute(d, m)
		d.SetPartial("state")
	}

	// Update
	resp, err := client.Request(req)
	if err != nil {
		return err
	}

	m := resp.(*egoscale.UpdateVirtualMachineResponse).VirtualMachine
	applyCompute(d, m)
	d.SetPartial("display_name")
	d.SetPartial("security_groups")

	if initialState == "Running" && (rebootRequired || startRequired) {
		commands = append(commands, partialCommand{
			partial: "state",
			request: &egoscale.StartVirtualMachine{
				ID: d.Id(),
			},
		})
	}

	for _, cmd := range commands {
		_, err := client.Request(cmd.request)
		if err != nil {
			return err
		}

		d.SetPartial(cmd.partial)
	}

	// Update oneself
	err = readCompute(d, meta)

	d.Partial(false)

	return err
}

func deleteCompute(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)

	req := &egoscale.DestroyVirtualMachine{
		ID: d.Id(),
	}
	_, err := client.Request(req)

	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func importCompute(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	id := d.Id()
	machine, err := getVirtualMachine(d, meta)
	if err != nil {
		if e := handleNotFound(d, err); e != nil {
			return nil, e
		}
		if d.Id() == "" {
			return nil, fmt.Errorf("Failure to import the compute resource: %s", id)
		}
	}

	// XXX simple, yet not complete
	secondaryIPs := machine.Nic[0].SecondaryIP
	nics := machine.NicsByType("Isolated")

	resources := make([]*schema.ResourceData, 0, 1+len(nics)+len(secondaryIPs))
	resources = append(resources, d)

	if secondaryIPs != nil {
		for _, secondaryIP := range secondaryIPs {
			resource := secondaryIPResource()
			d := resource.Data(nil)
			d.SetType("exoscale_secondary_ipaddress")
			d.Set("compute_id", id)
			if err := applySecondaryIP(d, secondaryIP); err != nil {
				return nil, err
			}
			d.Set("nic_id", machine.Nic[0].ID)
			d.Set("network_id", machine.Nic[0].NetworkID)

			resources = append(resources, d)
		}
	}

	for _, nic := range nics {
		resource := nicResource()
		d := resource.Data(nil)
		d.SetType("exoscale_nic")
		if err := applyNic(d, nic); err != nil {
			return nil, err
		}

		resources = append(resources, d)
	}

	return resources, nil
}

func applyCompute(d *schema.ResourceData, machine egoscale.VirtualMachine) error {
	d.Set("name", machine.Name)
	d.Set("display_name", machine.DisplayName)
	d.Set("key_pair", machine.KeyPair)
	d.Set("size", machine.ServiceOfferingName)
	d.Set("template", machine.TemplateName)
	d.Set("zone", machine.ZoneName)
	d.Set("state", machine.State)

	if len(machine.Nic) > 0 && machine.Nic[0].IPAddress != nil {
		d.Set("ip_address", machine.Nic[0].IPAddress.String())
	} else {
		d.Set("ip_address", "")
	}

	// affinity groups
	affinityGroups := make([]string, len(machine.AffinityGroup))
	for i, ag := range machine.AffinityGroup {
		affinityGroups[i] = ag.Name
	}
	d.Set("affinity_groups", affinityGroups)

	// security groups
	securityGroups := make([]string, len(machine.SecurityGroup))
	for i, sg := range machine.SecurityGroup {
		securityGroups[i] = sg.Name
	}
	d.Set("security_groups", securityGroups)

	// tags
	tags := make(map[string]interface{})
	for _, tag := range machine.Tags {
		tags[tag.Key] = tag.Value
	}
	d.Set("tags", tags)

	// Connection info for the provisioners
	d.SetConnInfo(map[string]string{
		"host": d.Get("ip_address").(string),
	})

	return nil
}

func getVirtualMachine(d *schema.ResourceData, meta interface{}) (*egoscale.VirtualMachine, error) {
	client := GetComputeClient(meta)

	// Permit to search for a VM by its name (useful when doing imports
	id := d.Id()
	name := ""
	if !isUUID(id) {
		name = id
		id = ""
	}

	resp, err := client.Request(&egoscale.ListVirtualMachines{
		ID:   id,
		Name: name,
	})

	if err != nil {
		return nil, err
	}

	vms := resp.(*egoscale.ListVirtualMachinesResponse)
	if len(vms.VirtualMachine) == 0 {
		// Ugly... this reproduces the CS behavior
		err := &egoscale.ErrorResponse{
			ErrorCode: egoscale.ParamError,
			ErrorText: fmt.Sprintf("VirtualMachine not found %s", d.Id()),
		}
		return nil, err
	} else if len(vms.VirtualMachine) > 1 {
		return nil, fmt.Errorf("More than one VM found. Query: %s", d.Id())
	}

	machine := vms.VirtualMachine[0]
	d.SetId(machine.ID)
	return &machine, nil
}

/*
 * An auxiliary function to ensure that the template string passed in maps to
 * the string provided by the egoscale driver.
 */
func convertTemplateName(t string) string {
	re := regexp.MustCompile(`^Linux (?P<name>.+?) (?P<version>[0-9.]+).*$`)
	submatch := re.FindStringSubmatch(t)
	if len(submatch) > 0 {
		name := strings.Replace(strings.ToLower(submatch[1]), " ", "-", -1)
		version := submatch[2]
		image := fmt.Sprintf("%s-%s", name, version)

		return image
	}
	return ""
}

func getSecurityGroupID(client *egoscale.Client, name string) (string, error) {
	if isUUID(name) {
		return name, nil
	}
	req := &egoscale.ListSecurityGroups{
		SecurityGroupName: name,
	}
	resp, err := client.Request(req)
	if err != nil {
		return "", err
	}

	securityGroups := resp.(*egoscale.ListSecurityGroupsResponse)
	if securityGroups.Count == 0 {
		return "", fmt.Errorf("SecurityGroup not found %s", name)
	}

	return securityGroups.SecurityGroup[0].ID, nil
}

func getAffinityGroupID(client *egoscale.Client, name string) (string, error) {
	if isUUID(name) {
		return name, nil
	}
	req := &egoscale.ListAffinityGroups{
		Name: name,
	}
	resp, err := client.Request(req)
	if err != nil {
		return "", err
	}

	affinityGroups := resp.(*egoscale.ListAffinityGroupsResponse)
	if affinityGroups.Count == 0 {
		return "", fmt.Errorf("AffinityGroup not found %s", name)
	}

	return affinityGroups.AffinityGroup[0].ID, nil
}

// isUuid matches a UUIDv4
func isUUID(uuid string) bool {
	re := regexp.MustCompile(`(?i)^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`)
	return re.MatchString(uuid)
}

func prepareUserData(d *schema.ResourceData, key string) (string, error) {
	userData := d.Get(key).(string)
	if strings.HasPrefix(userData, "#cloud-config") || strings.HasPrefix(userData, "Content-Type: multipart/mixed;") {
		log.Printf("[DEBUG] cloud-config detected, gzipping")

		b := new(bytes.Buffer)
		gz := gzip.NewWriter(b)
		if _, err := gz.Write([]byte(userData)); err != nil {
			return "", err
		}
		if err := gz.Flush(); err != nil {
			return "", err
		}
		if err := gz.Close(); err != nil {
			return "", err
		}

		return base64.StdEncoding.EncodeToString(b.Bytes()), nil
	}

	return userData, nil
}
