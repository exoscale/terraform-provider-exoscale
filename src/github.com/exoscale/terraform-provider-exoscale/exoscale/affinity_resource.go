package exoscale

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func affinityResource() *schema.Resource {
	return &schema.Resource{
		Create: affinityCreate,
		Read:   affinityRead,
		Update: nil,
		Delete: affinityDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func affinityCreate(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	groupName := d.Get("name").(string)
	affinity, err := client.CreateAffinityGroup(groupName, async)
	if err != nil {
		return err
	}

	d.SetId(affinity.Id)
	d.Set("name", affinity.Name)

	return affinityRead(d, meta)
}

func affinityRead(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	groups, err := client.GetAffinityGroups()
	if err != nil {
		return err
	}

	for k, v := range groups {
		if v == d.Id() {
			d.Set("name", k)
			return nil
		}
	}

	return fmt.Errorf("Affinity Group %s not found", d.Id())
}

func affinityDelete(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	groupName := d.Get("name").(string)
	_, err := client.DeleteAffinityGroup(groupName, async)

	return err
}
