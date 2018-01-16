package exoscale

import (
	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
)

func affinityGroupResource() *schema.Resource {
	return &schema.Resource{
		Create: createAffinityGroup,
		Exists: existsAffinityGroup,
		Read:   readAffinityGroup,
		Delete: deleteAffinityGroup,

		Importer: &schema.ResourceImporter{
			State: importAffinityGroup,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
			"type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "host anti-affinity",
			},
			"virtual_machine_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Set:      schema.HashString,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

func createAffinityGroup(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	req := &egoscale.CreateAffinityGroup{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Type:        d.Get("type").(string),
	}
	r, err := client.AsyncRequest(req, async)
	if err != nil {
		return err
	}

	ag := r.(egoscale.CreateAffinityGroupResponse).AffinityGroup
	return applyAffinityGroup(d, ag)
}

func existsAffinityGroup(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := GetComputeClient(meta)

	r, err := client.Request(&egoscale.ListAffinityGroups{
		ID: d.Id(),
	})

	if err != nil {
		e := handleNotFound(d, err)
		return d.Id() != "", e
	}

	return r.(*egoscale.ListAffinityGroupsResponse).Count > 0, nil
}

func readAffinityGroup(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)

	r, err := client.Request(&egoscale.ListAffinityGroups{
		ID: d.Id(),
	})
	if err != nil {
		return handleNotFound(d, err)
	}

	return applyAffinityGroup(d, r.(egoscale.ListAffinityGroupsResponse).AffinityGroup[0])
}

func applyAffinityGroup(d *schema.ResourceData, affinity egoscale.AffinityGroup) error {
	d.SetId(affinity.ID)
	d.Set("name", affinity.Name)
	d.Set("description", affinity.Description)
	d.Set("type", affinity.Type)
	d.Set("virtual_machine_ids", affinity.VirtualMachineIDs)

	return nil
}

func deleteAffinityGroup(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	req := &egoscale.DeleteAffinityGroup{
		ID: d.Id(),
	}
	return client.BooleanAsyncRequest(req, async)
}

func importAffinityGroup(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := readAffinityGroup(d, meta); err != nil {
		return nil, err
	}

	resources := make([]*schema.ResourceData, 1)
	resources[0] = d
	return resources, nil
}
