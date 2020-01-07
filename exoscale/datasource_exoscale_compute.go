package exoscale

import (
	"context"
	"errors"
	"fmt"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceCompute() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": {
				Type:          schema.TypeString,
				Description:   "ID of the Compute",
				Optional:      true,
				ConflictsWith: []string{"name", "tag"},
			},
			"name": {
				Type:          schema.TypeString,
				Description:   "Name of the Compute",
				Optional:      true,
				ConflictsWith: []string{"id", "tag"},
			},
			"tags": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description:   "Map of tags (key: value)",
				Optional:      true,
				ConflictsWith: []string{"id", "name"},
			},
		},

		Read: dataSourceComputeRead,
	}
}

func dataSourceComputeRead(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	req := egoscale.ListVirtualMachines{}

	computeName, byName := d.GetOk("name")
	computeID, byID := d.GetOk("id")
	computeTag, byTag := d.GetOk("tag")
	switch {
	case !byName && !byID && !byTag:
		return errors.New("either name, id or tag must be specified")
	case computeID != "":
		var err error
		if req.ID, err = egoscale.ParseUUID(computeID.(string)); err != nil {
			return fmt.Errorf("invalid value for id: %s", err)
		}
	case byTag:
		for key, value := range computeTag.(map[string]string) {
			req.Tags = append(req.Tags, egoscale.ResourceTag{
				Key:   key,
				Value: value,
			})
		}
	default:
		req.Name = computeName.(string)
	}

	resp, err := client.RequestWithContext(ctx, &req)
	if err != nil {
		return fmt.Errorf("compute list query failed: %s", err)
	}

	var compute egoscale.VirtualMachine
	nt := resp.(*egoscale.ListVirtualMachinesResponse).Count
	switch {
	case nt == 0:
		return errors.New("compute not found")

	case nt > 1:
		return errors.New("multiple results returned, expected only one")

	default:
		compute = resp.(*egoscale.ListVirtualMachinesResponse).VirtualMachine[0]
	}

	d.SetId(compute.ID.String())

	return nil
}
