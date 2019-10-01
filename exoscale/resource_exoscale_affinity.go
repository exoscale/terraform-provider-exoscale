package exoscale

import (
	"context"
	"log"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceAffinityIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_affinity")
}

func resourceAffinity() *schema.Resource {
	return &schema.Resource{
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

		Create: resourceAffinityCreate,
		Read:   resourceAffinityRead,
		Delete: resourceAffinityDelete,
		Exists: resourceAffinityExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceAffinityCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning create", resourceAffinityIDString(d))

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

	ag := resp.(*egoscale.AffinityGroup)
	d.SetId(ag.ID.String())

	log.Printf("[DEBUG] %s: create finished successfully", resourceAffinityIDString(d))

	return resourceAffinityRead(d, meta)
}

func resourceAffinityExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return false, err
	}

	ag := &egoscale.AffinityGroup{ID: id}
	_, err = client.GetWithContext(ctx, ag)
	if err != nil {
		e := handleNotFound(d, err)
		return d.Id() != "", e
	}

	return true, nil
}

func resourceAffinityRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning read", resourceAffinityIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	ag := &egoscale.AffinityGroup{ID: id}

	resp, err := client.GetWithContext(ctx, ag)
	if err != nil {
		return handleNotFound(d, err)
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceAffinityIDString(d))

	return resourceAffinityApply(d, resp.(*egoscale.AffinityGroup))
}

func resourceAffinityDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning delete", resourceAffinityIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	ag := &egoscale.AffinityGroup{ID: id}

	if err := client.DeleteWithContext(ctx, ag); err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceAffinityIDString(d))

	return nil
}

func resourceAffinityApply(d *schema.ResourceData, affinity *egoscale.AffinityGroup) error {
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
