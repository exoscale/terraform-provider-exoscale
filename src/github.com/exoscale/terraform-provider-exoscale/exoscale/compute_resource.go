package exoscale

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
)

const DelayBeforeRetry = 5 // seconds

func computeResource() *schema.Resource {
	return &schema.Resource{
		Create: resourceCreate,
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
			"userdata": {
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
	topo, err := client.GetTopology()
	if err != nil {
		return err
	}

	diskSize := d.Get("disk_size").(int)
	service := topo.Profiles[strings.ToLower(d.Get("size").(string))]

	if service == "" {
		return fmt.Errorf("Invalid service: %s", d.Get("size").(string))
	}

	zone := topo.Zones[strings.ToLower(d.Get("zone").(string))]
	if zone == "" {
		return fmt.Errorf("Invalid zone: %s", d.Get("zone").(string))
	}

	template := topo.Images[convertTemplateName(d.Get("template").(string))]
	if template == nil {
		return fmt.Errorf("Invalid template: %s", d.Get("template").(string))
	}

	templateId := template[diskSize]
	if templateId == "" {
		return fmt.Errorf("Invalid disk size: %d", diskSize)
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
				securityGroups[i] = securityGroup.Id
			} else {
				return err
			}
		}
	}

	profile := egoscale.MachineProfile{
		Name:            d.Get("name").(string),
		Keypair:         d.Get("keypair").(string),
		Userdata:        d.Get("userdata").(string),
		ServiceOffering: service,
		Template:        templateId,
		Zone:            zone,
		AffinityGroups:  affinityGroups,
		SecurityGroups:  securityGroups,
	}

	jobId, err := client.CreateVirtualMachine(profile)
	if err != nil {
		return err
	}

	var timeoutSeconds = meta.(BaseConfig).timeout
	var retries = timeoutSeconds / DelayBeforeRetry

	var resp *egoscale.QueryAsyncJobResultResponse
	var succeeded = false
	for i := 0; i < retries; i++ {
		resp, err = client.PollAsyncJob(jobId)
		if err != nil {
			return err
		}

		if resp.Jobstatus == 1 {
			succeeded = true
			break
		}

		time.Sleep(DelayBeforeRetry * time.Second)
	}

	if !succeeded {
		return errors.New(fmt.Sprintf("Virtual machine creation did not succeed within %d seconds. You may increase "+
			"the timeout in the provider configuration.", timeoutSeconds))
	}

	vm, err := client.AsyncToVirtualMachine(*resp)
	if err != nil {
		return err
	}

	/* Copy VM to our struct */
	d.SetId(vm.Id)

	return resourceRead(d, meta)
}

func resourceRead(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	machine, err := client.GetVirtualMachine(d.Id())

	if err != nil {
		return err
	}

	d.Set("name", machine.Displayname)
	d.Set("keypair", machine.Keypair)
	d.Set("userdata", "")
	d.Set("size", machine.Serviceofferingname)
	d.Set("template", machine.Templatename)
	d.Set("zone", machine.Zonename)
	d.Set("state", machine.State)

	nicArray := make([]map[string]string, len(machine.Nic))
	for j, n := range machine.Nic {
		i := make(map[string]string)
		i["ip6address"] = n.Ip6address
		i["ip4address"] = n.Ipaddress
		i["type"] = n.Type
		i["networkname"] = n.Networkname

		if n.Isdefault {
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

	resp, err := client.DestroyVirtualMachine(d.Id())

	if err != nil {
		return err
	}

	log.Printf("Deleted vm id: %s\n", resp)
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
		params.Set("name", name)
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
