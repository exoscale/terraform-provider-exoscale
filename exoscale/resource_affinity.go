package exoscale

import (
	"context"

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
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
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
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	req := &egoscale.CreateAffinityGroup{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		Type:        d.Get("type").(string),
	}
	resp, err := client.RequestWithContext(ctx, req)
	if err != nil {
		return err
	}

	return applyAffinityGroup(d, resp.(*egoscale.AffinityGroup))
}

func existsAffinityGroup(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return false, err
	}

	ag := &egoscale.AffinityGroup{
		ID: id,
	}
	_, err = client.GetWithContext(ctx, ag)
	if err != nil {
		e := handleNotFound(d, err)
		return d.Id() != "", e
	}

	return true, nil
}

func readAffinityGroup(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	ag := &egoscale.AffinityGroup{
		ID: id,
	}
	resp, err := client.GetWithContext(ctx, ag)
	if err != nil {
		return handleNotFound(d, err)
	}

	return applyAffinityGroup(d, resp.(*egoscale.AffinityGroup))
}

func applyAffinityGroup(d *schema.ResourceData, affinity *egoscale.AffinityGroup) error {
	d.SetId(affinity.ID.String())
	if err := d.Set("name", affinity.Name); err != nil {
		return err
	}
	if err := d.Set("description", affinity.Description); err != nil {
		return err
	}
	if err := d.Set("type", affinity.Type); err != nil {
		return err
	}
	ids := make([]string, len(affinity.VirtualMachineIDs))
	for i, id := range affinity.VirtualMachineIDs {
		ids[i] = id.String()
	}
	if err := d.Set("virtual_machine_ids", ids); err != nil {
		return err
	}

	return nil
}

func deleteAffinityGroup(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	return client.DeleteWithContext(ctx, &egoscale.AffinityGroup{
		ID: id,
	})
}
