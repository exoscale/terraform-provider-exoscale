package exoscale

import (
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
)

func computeResource() *schema.Resource {
	return &schema.Resource{
		Create: resourceCreate,
		Exists: resourceExists,
		Read:   resourceRead,
		Delete: resourceDelete,
		Update: resourceUpdate,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"template": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"size": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"disk_size": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"zone": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"user_data": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"keypair": {
				Type:     schema.TypeString,
				Required: true,
			},
			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"networks": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type:    schema.TypeMap,
					Default: make(map[string]string),
				},
			},
			"affinitygroups": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"securitygroups": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Set: schema.HashString,
			},
		},
	}
}

func resourceCreate(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	topo, err := client.GetTopology()
	if err != nil {
		return err
	}

	diskSize := d.Get("disk_size").(int)
	service := topo.Profiles[strings.ToLower(d.Get("size").(string))]

	if service == "" {
		return fmt.Errorf("Invalid service: %s", d.Get("size").(string))
	}

	zoneName := d.Get("zone").(string)
	zone := topo.Zones[strings.ToLower(zoneName)]
	if zone == nil {
		return fmt.Errorf("Invalid zone: %s", zoneName)
	}

	template := topo.Images[convertTemplateName(d.Get("template").(string))]
	if template == nil {
		return fmt.Errorf("Invalid template: %s", d.Get("template").(string))
	}

	// If the exact diskSize doesn't exist pick the smallest one and go for it
	templateId := template[diskSize]
	if templateId == "" {
		smallestDiskSize := diskSize
		for s := range template {
			if s < smallestDiskSize {
				smallestDiskSize = s
			}
		}

		templateId = template[smallestDiskSize]
		if templateId == "" {
			return fmt.Errorf("Invalid disk size: %d", diskSize)
		}
	}

	affinityCount := d.Get("affinitygroups.#").(int)
	var affinityGroups []string
	if affinityCount > 0 {
		affinityGroups = make([]string, affinityCount)
		for i := 0; i < affinityCount; i++ {
			ag := fmt.Sprintf("affinitygroups.%d", i)
			affinityGroups[i] = d.Get(ag).(string)
		}
	}

	var securityGroups []string
	if securitySet, ok := d.Get("securitygroups").(*schema.Set); ok {
		securityGroups = make([]string, securitySet.Len())
		for i, group := range securitySet.List() {
			groupName := group.(string)

			securityGroup, err := getSecurityGroup(client, groupName)

			if err != nil {
				return err
			}
			securityGroups[i] = securityGroup.Id
		}
	}

	profile := egoscale.VirtualMachineProfile{
		Name:            d.Get("name").(string),
		DiskSize:        uint64(diskSize),
		Keypair:         d.Get("keypair").(string),
		UserData:        d.Get("user_data").(string),
		ServiceOffering: service,
		Template:        templateId,
		Zone:            zone.Id,
		AffinityGroups:  affinityGroups,
		SecurityGroups:  securityGroups,
	}

	vm, err := client.CreateVirtualMachine(profile, async)
	if err != nil {
		return err
	}

	/* Copy VM to our struct */
	d.SetId(string(vm.Id))

	// Connection info for the provisioners
	d.SetConnInfo(map[string]string{
		"host": vm.Nic[0].IpAddress,
	})

	return resourceRead(d, meta)
}

func resourceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := GetComputeClient(meta)
	_, err := client.GetVirtualMachine(d.Id())

	return err == nil, nil
}

func resourceRead(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	machine, err := client.GetVirtualMachine(d.Id())
	if err != nil {
		return err
	}

	volume, err := client.GetRootVolumeForVirtualMachine(d.Id())
	if err != nil {
		return err
	}

	d.Set("name", machine.DisplayName)
	d.Set("keypair", machine.KeyPair)
	//d.Set("user_data", "")
	d.Set("size", machine.ServiceOfferingName)
	d.Set("disk_size", volume.Size>>30) // B to GiB
	d.Set("template", machine.TemplateName)
	d.Set("zone", machine.ZoneName)
	d.Set("state", machine.State)

	nicArray := make([]map[string]string, len(machine.Nic))
	for j, n := range machine.Nic {
		i := make(map[string]string)
		i["ip6address"] = n.Ip6Address
		i["ip4address"] = n.IpAddress
		i["type"] = n.Type
		i["networkname"] = n.NetworkName

		if n.IsDefault {
			i["default"] = "true"
		} else {
			i["default"] = "false"
		}
		nicArray[j] = i
	}

	d.Set("networks", nicArray)

	return nil
}

func resourceUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceDelete(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	_, err := client.DestroyVirtualMachine(d.Id(), async)

	if err != nil {
		return err
	}

	log.Printf("Deleted vm id: %s\n", d.Id())
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
	} else {
		return ""
	}
}

// getSecurityGroup finds a SecurityGroup by UUID or name
func getSecurityGroup(client *egoscale.Client, name string) (*egoscale.SecurityGroup, error) {
	params := url.Values{}
	if isUuid(name) {
		log.Printf("[DEBUG] search Security Group by id: %s", name)
		params.Set("id", name)
	} else {
		log.Printf("[DEBUG] search Security Group by name: %s", name)
		params.Set("securitygroupname", name)
	}
	sgs, err := client.GetSecurityGroups(params)
	if err != nil {
		return nil, err
	}

	if len(sgs) == 1 {
		return sgs[0], nil
	}
	return nil, fmt.Errorf("Invalid security group: %s. Found %d.", name, len(sgs))
}

// isUuid matches a UUIDv4
func isUuid(uuid string) bool {
	re := regexp.MustCompile(`(?i)^[a-f0-9]{8}-[a-f0-9]{4}-4[a-f0-9]{3}-[89ab][a-f0-9]{3}-[a-f0-9]{12}$`)
	return re.MatchString(uuid)
}
