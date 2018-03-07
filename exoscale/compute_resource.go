package exoscale

import (
	"bytes"
	"compress/gzip"
	"context"
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
		"affinity_group_ids": {
			Type:          schema.TypeSet,
			Optional:      true,
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

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},

		Schema: s,
	}
}

func createCompute(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

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
	zone, err := getZoneByName(ctx, client, zoneName)
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

	// Affinity Groups
	var affinityGroups []string
	if affinitySet, ok := d.Get("affinity_groups").(*schema.Set); ok {
		affinityGroups = make([]string, 0, affinitySet.Len())
		for _, group := range affinitySet.List() {
			affinityGroups = append(affinityGroups, group.(string))
		}

	}

	var affinityGroupIDs []string
	if affinityIDSet, ok := d.Get("affinity_group_ids").(*schema.Set); ok {
		affinityGroupIDs = make([]string, 0, affinityIDSet.Len())
		for _, group := range affinityIDSet.List() {
			affinityGroupIDs = append(affinityGroupIDs, group.(string))
		}
	}

	// Security Groups
	var securityGroups []string
	if securitySet, ok := d.Get("security_groups").(*schema.Set); ok {
		securityGroups = make([]string, 0, securitySet.Len())
		for _, group := range securitySet.List() {
			securityGroups = append(securityGroups, group.(string))
		}
	}

	var securityGroupIDs []string
	if securityIDSet, ok := d.Get("security_group_ids").(*schema.Set); ok {
		securityGroupIDs = make([]string, 0, securityIDSet.Len())
		for _, group := range securityIDSet.List() {
			securityGroupIDs = append(securityGroupIDs, group.(string))
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
		Name:               displayName,
		DisplayName:        displayName,
		RootDiskSize:       int64(diskSize),
		KeyPair:            d.Get("key_pair").(string),
		Keyboard:           d.Get("keyboard").(string),
		UserData:           userData,
		ServiceOfferingID:  service,
		TemplateID:         templateID,
		ZoneID:             zone.ID,
		AffinityGroupIDs:   affinityGroupIDs,
		AffinityGroupNames: affinityGroups,
		SecurityGroupIDs:   securityGroupIDs,
		SecurityGroupNames: securityGroups,
		StartVM:            &startVM,
	}

	resp, err := client.RequestWithContext(ctx, req)
	if err != nil {
		return err
	}

	/* Copy VM to our struct */
	machine := resp.(*egoscale.DeployVirtualMachineResponse).VirtualMachine
	d.SetId(machine.ID)

	if cmd := createTags(d, "tags", machine.ResourceType()); cmd != nil {
		if err := client.BooleanRequestWithContext(ctx, cmd); err != nil {
			// Attempting to destroy the freshly created machine
			_, e := client.RequestWithContext(ctx, &egoscale.DestroyVirtualMachine{
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
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	_, err := getVirtualMachine(ctx, d, meta)

	// The CS API returns an error if it doesn't exist
	if err != nil {
		e := handleNotFound(d, err)
		return d.Id() != "", e
	}

	return true, nil
}

func readCompute(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	// XXX apply context
	machine, err := getVirtualMachine(ctx, d, meta)
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
	// XXX apply context
	volume, err := client.GetRootVolumeForVirtualMachine(d.Id())
	if err != nil {
		return err
	}
	d.Set("disk_size", volume.Size>>30) // B to GiB

	return applyCompute(d, *machine)
}

func updateCompute(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

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
		partial  string
		partials []string
		request  egoscale.Command
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

		securityGroupIDs := make([]string, 0)
		if securitySet, ok := d.Get("security_groups").(*schema.Set); ok {
			for _, group := range securitySet.List() {
				sg, err := getSecurityGroup(ctx, client, group.(string))
				if err != nil {
					return err
				}
				securityGroupIDs = append(securityGroupIDs, sg.ID)
			}
		}

		if len(securityGroupIDs) == 0 {
			return fmt.Errorf("A VM must have at least one Security Group, none found")
		}

		req.SecurityGroupIDs = securityGroupIDs
	} else if d.HasChange("security_group_ids") {
		rebootRequired = true

		securityGroupIDs := make([]string, 0)
		if securitySet, ok := d.Get("security_group_ids").(*schema.Set); ok {
			for _, group := range securitySet.List() {
				securityGroupIDs = append(securityGroupIDs, group.(string))
			}
		}

		if len(securityGroupIDs) == 0 {
			return fmt.Errorf("A VM must have at least one Security Group, none found")
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
		services, err := client.RequestWithContext(ctx, &egoscale.ListServiceOfferings{
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
		o, n := d.GetChange("affinity_groups")
		if o.(*schema.Set).Len() >= n.(*schema.Set).Len() {
			return fmt.Errorf("Affinity Groups cannot be added.")
		}
		if n.(*schema.Set).Difference(o.(*schema.Set)).Len() > 0 {
			return fmt.Errorf("No new Affinity Groups can be added.")
		}

		if affinitySet, ok := d.Get("affinity_groups").(*schema.Set); ok {
			affinityGroups := make([]string, affinitySet.Len())
			for i, group := range affinitySet.List() {
				affinityGroups[i] = group.(string)
			}
			commands = append(commands, partialCommand{
				partials: []string{"affinity_groups", "affinity_group_ids"},
				request: &egoscale.UpdateVMAffinityGroup{
					ID:                 d.Id(),
					AffinityGroupNames: affinityGroups,
				},
			})
		}
	} else if d.HasChange("affinity_group_ids") {
		rebootRequired = true
		o, n := d.GetChange("affinity_group_ids")
		if o.(*schema.Set).Len() >= n.(*schema.Set).Len() {
			return fmt.Errorf("Affinity Groups cannot be added.")
		}
		if n.(*schema.Set).Difference(o.(*schema.Set)).Len() > 0 {
			return fmt.Errorf("No new Affinity Groups can be added.")
		}

		if affinitySet, ok := d.Get("affinity_group_ids").(*schema.Set); ok {
			affinityGroups := make([]string, affinitySet.Len())
			for i, group := range affinitySet.List() {
				affinityGroups[i] = group.(string)
			}
			commands = append(commands, partialCommand{
				partials: []string{"affinity_groups", "affinity_group_ids"},
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
		resp, err := client.RequestWithContext(ctx, &egoscale.StopVirtualMachine{
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
	resp, err := client.RequestWithContext(ctx, req)
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

	// Update oneself
	err = readCompute(d, meta)

	d.Partial(false)

	return err
}

func deleteCompute(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	req := &egoscale.DestroyVirtualMachine{
		ID: d.Id(),
	}
	_, err := client.RequestWithContext(ctx, req)

	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func importCompute(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	id := d.Id()
	machine, err := getVirtualMachine(ctx, d, meta)
	if err != nil {
		if e := handleNotFound(d, err); e != nil {
			return nil, e
		}
		if d.Id() == "" {
			return nil, fmt.Errorf("Failure to import the compute resource: %s", id)
		}
	}

	defaultNic := machine.DefaultNic()
	if defaultNic == nil {
		return nil, fmt.Errorf("VM %v has no default NIC", d.Id())
	}
	secondaryIPs := defaultNic.SecondaryIP
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
	affinityGroupIDs := make([]string, len(machine.AffinityGroup))
	for i, ag := range machine.AffinityGroup {
		affinityGroups[i] = ag.Name
		affinityGroupIDs[i] = ag.ID
	}
	d.Set("affinity_groups", affinityGroups)
	d.Set("affinity_group_ids", affinityGroupIDs)

	// security groups
	securityGroups := make([]string, len(machine.SecurityGroup))
	securityGroupIDs := make([]string, len(machine.SecurityGroup))
	for i, sg := range machine.SecurityGroup {
		securityGroups[i] = sg.Name
		securityGroupIDs[i] = sg.ID
	}
	d.Set("security_groups", securityGroups)
	d.Set("security_group_ids", securityGroupIDs)

	// tags
	tags := make(map[string]interface{})
	for _, tag := range machine.Tags {
		tags[tag.Key] = tag.Value
	}
	d.Set("tags", tags)

	// Connection info for the provisioners
	d.SetConnInfo(map[string]string{
		"type": "ssh",
		"user": getSSHUsername(machine.TemplateName),
		"host": d.Get("ip_address").(string),
	})

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

func getVirtualMachine(ctx context.Context, d *schema.ResourceData, meta interface{}) (*egoscale.VirtualMachine, error) {
	client := GetComputeClient(meta)

	// Permit to search for a VM by its name (useful when doing imports
	id := d.Id()
	name := ""
	if !isUUID(id) {
		name = id
		id = ""
	}

	machine := &egoscale.VirtualMachine{
		ID:   id,
		Name: name,
	}

	if err := client.GetWithContext(ctx, machine); err != nil {
		return nil, err
	}

	d.SetId(machine.ID)
	return machine, nil
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

func getSecurityGroup(ctx context.Context, client *egoscale.Client, name string) (*egoscale.SecurityGroup, error) {
	req := &egoscale.ListSecurityGroups{
		SecurityGroupName: name,
	}
	resp, err := client.RequestWithContext(ctx, req)
	if err != nil {
		return nil, err
	}

	sgs := resp.(*egoscale.ListSecurityGroupsResponse)
	if len(sgs.SecurityGroup) == 0 {
		return nil, fmt.Errorf("SecurityGroup not found %s", name)
	}

	sg := sgs.SecurityGroup[0]
	return &sg, nil
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
