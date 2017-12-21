package exoscale

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func ipAssociateResource() *schema.Resource {
	return &schema.Resource{
		Create: ipAssociateCreate,
		Read:   ipAssociateRead,
		Delete: ipAssociateDelete,

		Schema: map[string]*schema.Schema{
			"compute_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"elasticip": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func ipAssociateCreate(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	virtualMachineId := d.Get("compute_id").(string)
	ipAddress := d.Get("elasticip").(string)

	nics, err := client.ListNics(virtualMachineId)
	if err != nil {
		return err
	}

	if len(nics) == 0 {
		return fmt.Errorf("The VM has no NIC %v", virtualMachineId)
	}

	secondaryIp, err := client.AddIpToNic(nics[0].Id, ipAddress, async)
	if err != nil {
		return err
	}

	d.SetId(secondaryIp.Id)

	return nil
}

func ipAssociateRead(d *schema.ResourceData, meta interface{}) error {
	// do nothing
	return nil
}

func ipAssociateDelete(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	err := client.RemoveIpFromNic(d.Id(), async)
	if err != nil {
		return err
	}

	return nil
}
