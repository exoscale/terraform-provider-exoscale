package exoscale

import (
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
	return &schema.Resource{
		Create: createCompute,
		Exists: existsCompute,
		Read:   readCompute,
		Update: updateCompute,
		Delete: deleteCompute,

		Importer: &schema.ResourceImporter{
			State: importCompute,
		},

		Schema: map[string]*schema.Schema{
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
				Computed: true,
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
				Set:      schema.HashString,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"security_groups": {
				Type:     schema.TypeSet,
				Optional: true,
				Set:      schema.HashString,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func createCompute(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

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

	req := &egoscale.DeployVirtualMachine{
		Name:              displayName,
		DisplayName:       displayName,
		RootDiskSize:      int64(diskSize),
		KeyPair:           d.Get("key_pair").(string),
		Keyboard:          d.Get("keyboard").(string),
		UserData:          []byte(d.Get("user_data").(string)),
		ServiceOfferingID: service,
		TemplateID:        templateID,
		ZoneID:            zone.ID,
		AffinityGroupIDs:  affinityGroups,
		SecurityGroupIDs:  securityGroups,
	}

	r, err := client.AsyncRequest(req, async)
	if err != nil {
		return err
	}

	/* Copy VM to our struct */
	machine := r.(*egoscale.DeployVirtualMachineResponse).VirtualMachine
	d.SetId(machine.ID)

	if t, ok := d.GetOk("tags"); ok {
		m := t.(map[string]interface{})
		tags := make([]*egoscale.ResourceTag, 0, len(m))
		for k, v := range m {
			tags = append(tags, &egoscale.ResourceTag{
				Key:   k,
				Value: v.(string),
			})
		}

		err := client.BooleanAsyncRequest(&egoscale.CreateTags{
			ResourceIDs:  []string{machine.ID},
			ResourceType: "userVM",
			Tags:         tags,
		}, async)

		if err != nil {
			// Attempting to destroy the freshly created machine
			_, e := client.AsyncRequest(&egoscale.DestroyVirtualMachine{
				ID: machine.ID,
			}, async)
			if e != nil {
				log.Printf("[WARNING] Failure to create the tags, but the machine was deployed. %v", e)
			}
			return err
		}
	}

	return readCompute(d, meta)
}

func existsCompute(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := GetComputeClient(meta)
	_, err := client.Request(&egoscale.ListVirtualMachines{
		ID: d.Id(),
	})

	// The CS API returns an error if it doesn't exist
	if err != nil {
		if r, ok := err.(*egoscale.ErrorResponse); ok {
			if r.ErrorCode == 431 {
				return false, nil
			}
		}
		return false, err
	}

	return true, nil
}

func readCompute(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	resp, err := client.Request(&egoscale.ListVirtualMachines{
		ID: d.Id(),
	})
	if err != nil {
		if r, ok := err.(*egoscale.ErrorResponse); ok {
			if r.ErrorCode == 431 {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	vms := resp.(*egoscale.ListVirtualMachinesResponse)
	if vms.Count == 0 {
		d.SetId("")
		return nil
	}

	machine := vms.VirtualMachine[0]

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

	// tags
	tags := make(map[string]interface{})
	for _, tag := range machine.Tags {
		tags[tag.Key] = tag.Value
	}
	d.Set("tags", tags)

	return applyCompute(machine, d)
}

func updateCompute(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	requests := make([]egoscale.AsyncCommand, 0)

	initialState := d.Get("state").(string)
	if initialState != "Running" && initialState != "Stopped" {
		return fmt.Errorf("VM %s must be either Running or Stopped. got %s", d.Id(), initialState)
	}

	rebootRequired := false
	startRequired := false
	stopRequired := false

	d.Partial(true)

	req := &egoscale.UpdateVirtualMachine{
		ID: d.Id(),
	}

	if d.HasChange("display_name") {
		req.DisplayName = d.Get("display_name").(string)
		d.SetPartial("display_name")
	}

	if d.HasChange("user_data") {
		req.UserData = []byte(d.Get("user_data").(string))
		d.SetPartial("user_data")
		rebootRequired = true
	}

	if d.HasChange("key_pair") {
		d.SetPartial("key_pair")
		keyPair := d.Get("key_pair").(string)
		resp, err := client.Request(&egoscale.ListSSHKeyPairs{
			Name: keyPair,
		})
		if err != nil {
			return err
		}

		if resp.(*egoscale.ListSSHKeyPairsResponse).Count == 0 {
			return fmt.Errorf("New SSH KeyPair doesn't exist, aborting. Got %s", keyPair)
		}

		rebootRequired = true
		requests = append(requests, &egoscale.ResetSSHKeyForVirtualMachine{
			ID:      d.Id(),
			KeyPair: keyPair,
		})
	}

	if d.HasChange("disk_size") {
		o, n := d.GetChange("disk_size")
		oldSize := o.(int)
		newSize := n.(int)

		if oldSize > newSize {
			return fmt.Errorf("A volume can only be expanded. From %dG to %dG is not allowed", oldSize, newSize)
		}

		d.SetPartial("disk_size")
		rebootRequired = true
		volume, err := client.GetRootVolumeForVirtualMachine(d.Id())
		if err != nil {
			return err
		}
		requests = append(requests, &egoscale.ResizeVolume{
			ID:   volume.ID,
			Size: int64(d.Get("disk_size").(int)),
		})
	}

	if d.HasChange("size") {
		d.SetPartial("size")
		rebootRequired = true
		services, err := client.Request(&egoscale.ListServiceOfferings{
			Name: d.Get("size").(string),
		})
		if err != nil {
			return err
		}
		requests = append(requests, &egoscale.ScaleVirtualMachine{
			ID:                d.Id(),
			ServiceOfferingID: services.(*egoscale.ListServiceOfferingsResponse).ServiceOffering[0].ID,
		})
	}

	if d.HasChange("security_groups") {
		d.SetPartial("security_groups")
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

	if d.HasChange("affinity_groups") {
		d.SetPartial("affinity_groups")

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
			requests = append(requests, &egoscale.UpdateVMAffinityGroup{
				ID:               d.Id(),
				AffinityGroupIDs: affinityGroups,
			})
		}
	}

	if d.HasChange("tags") {
		d.SetPartial("tags")
		o, n := d.GetChange("tags")

		oldTags := o.(map[string]interface{})
		newTags := n.(map[string]interface{})
		// Remove the intersection between the two sets of tag
		for k, v := range oldTags {
			if value, ok := newTags[k]; ok && v == value {
				delete(oldTags, k)
				delete(newTags, k)
			}
		}

		if len(oldTags) > 0 {
			deleteTags := &egoscale.DeleteTags{
				ResourceIDs:  []string{d.Id()},
				ResourceType: "userVM",
				Tags:         make([]*egoscale.ResourceTag, len(oldTags)),
			}
			i := 0
			for k, v := range oldTags {
				deleteTags.Tags[i] = &egoscale.ResourceTag{
					Key:   k,
					Value: v.(string),
				}
				i++
			}
			requests = append(requests, deleteTags)
		}
		if len(newTags) > 0 {
			createTags := &egoscale.CreateTags{
				ResourceIDs:  []string{d.Id()},
				ResourceType: "userVM",
				Tags:         make([]*egoscale.ResourceTag, len(newTags)),
			}
			i := 0
			for k, v := range newTags {
				createTags.Tags[i] = &egoscale.ResourceTag{
					Key:   k,
					Value: v.(string),
				}
				i++
			}
			requests = append(requests, createTags)
		}
	}

	if d.HasChange("state") {
		d.SetPartial("state")
		switch d.Get("state").(string) {
		case "Running":
			startRequired = true
		case "Stopped":
			stopRequired = true
		default:
			return fmt.Errorf("The new state cannot applied, %s. Do it manually", d.Get("state").(string))
		}
	}

	if initialState != "Stopped" && (rebootRequired || stopRequired) {
		_, err := client.AsyncRequest(&egoscale.StopVirtualMachine{
			ID: d.Id(),
		}, async)
		if err != nil {
			return err
		}
	}

	resp, err := client.Request(req)
	if err != nil {
		return err
	}

	if initialState == "Running" && (rebootRequired || startRequired) {
		requests = append(requests, &egoscale.StartVirtualMachine{
			ID: d.Id(),
		})
	}

	for _, req := range requests {
		_, err := client.AsyncRequest(req, async)
		if err != nil {
			return err
		}
	}

	d.Partial(false)

	return applyCompute(resp.(*egoscale.UpdateVirtualMachineResponse).VirtualMachine, d)
}

func deleteCompute(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	req := &egoscale.DestroyVirtualMachine{
		ID: d.Id(),
	}
	_, err := client.AsyncRequest(req, async)

	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func importCompute(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := readCompute(d, meta); err != nil {
		return nil, err
	}

	resources := make([]*schema.ResourceData, 1)
	resources[0] = d
	return resources, nil
}

func applyCompute(machine *egoscale.VirtualMachine, d *schema.ResourceData) error {
	d.Set("name", machine.Name)
	d.Set("display_name", machine.DisplayName)
	d.Set("key_pair", machine.KeyPair)
	d.Set("size", machine.ServiceOfferingName)
	d.Set("template", machine.TemplateName)
	d.Set("zone", machine.ZoneName)
	d.Set("state", machine.State)

	if len(machine.Nic) > 0 {
		d.Set("ip_address", machine.Nic[0].IPAddress)
	} else {
		d.Set("ip_address", "")
	}

	// Connection info for the provisioners
	d.SetConnInfo(map[string]string{
		"host": d.Get("ip_address").(string),
	})

	return nil
}

/*
 * An ancilliary function to ensure that the template string passed in maps to
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
	return resp.(*egoscale.ListSecurityGroupsResponse).SecurityGroup[0].ID, nil
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
	return resp.(*egoscale.ListAffinityGroupsResponse).AffinityGroup[0].ID, nil
}

// isUuid matches a UUIDv4
func isUUID(uuid string) bool {
	re := regexp.MustCompile(`(?i)^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`)
	return re.MatchString(uuid)
}
