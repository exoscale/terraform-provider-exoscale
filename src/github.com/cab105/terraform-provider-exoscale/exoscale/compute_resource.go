package exoscale

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/runseb/egoscale/src/egoscale"
)

func computeResource() *schema.Resource {
	return &schema.Resource{
		Create: resourceCreate,
		Read:   resourceRead,
		Delete: resourceDelete,
		Update: resourceUpdate,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"template": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"size": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"userdata": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"keypair": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"state": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"publicIP": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"networks": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type:    schema.TypeMap,
					Default: make(map[string]string),
				},
			},
		},
	}
}

func resourceCreate(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(ComputeEndpoint, meta)
	/* Missing SecurityGroups */
	profile := egoscale.MachineProfile{
		Name:            d.Get("name").(string),
		Keypair:         d.Get("keypair").(string),
		Userdata:        d.Get("userdata").(string),
		ServiceOffering: d.Get("size").(string),
		Template:        d.Get("template").(string),
		Zone:            d.Get("zone").(string),
	}

	id, err := client.CreateVirtualMachine(profile); if err != nil {
		return err
	}

	log.Printf("## job_id: %s\n", id)
	d.SetId(id)

	/* CAB: We're creating the resource only and not starting it */

	return resourceRead(d, meta)
}

func resourceRead(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(ComputeEndpoint, meta)
	machine, err := client.GetVirtualMachine(d.Get("id").(string))

	if err != nil {
		return err
	}

	d.Set("name", machine.Name)
	d.Set("keypain", machine.Keypair)
	d.Set("userdata", "")
	d.Set("size", machine.Serviceofferingname)
	d.Set("template", machine.Templatename)
	d.Set("zone", machine.Zonename)
	d.Set("state", machine.State)
	d.Set("publicIP", machine.Publicip)

	for _, n := range machine.Nic {
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
	}

	return nil
}

func resourceUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceDelete(d *schema.ResourceData, meta interface{}) error {
	client := GetClient(ComputeEndpoint, meta)

	resp, err := client.DestroyVirtualMachine(d.Get("id").(string))

	if err != nil {
		return err
	}

	log.Printf("Deleted vm id: %s\n", resp)
	return nil
}
