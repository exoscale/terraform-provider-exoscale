package exoscale

import (
	"context"
	"errors"
	"fmt"

	"github.com/exoscale/egoscale"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceAffinity() *schema.Resource {
	return &schema.Resource{
		Description: "Fetch Exoscale Anti-Affinity Groups data.",
		Schema: map[string]*schema.Schema{
			"id": {
				Description:   "The anti-affinity group ID to match (conflicts with `name`)",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
			},
			"name": {
				Description:   "The group name to match (conflicts with `id`)",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
			},
		},

		Read: dataSourceAffinityRead,
	}
}

func dataSourceAffinityRead(d *schema.ResourceData, meta interface{}) error {
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	req := egoscale.ListAffinityGroups{Type: "host anti-affinity"}

	agName, byName := d.GetOk("name")
	agID, byID := d.GetOk("id")
	if !byName && !byID {
		return errors.New("either name or id must be specified")
	}

	req.Name = agName.(string)

	if agID != "" {
		if req.ID, err = egoscale.ParseUUID(agID.(string)); err != nil {
			return fmt.Errorf("invalid value for id: %s", err)
		}
	}

	resp, err := client.GetWithContext(ctx, &req)
	if err != nil {
		return err
	}
	ag := resp.(*egoscale.AffinityGroup)

	d.SetId(ag.ID.String())

	if err := d.Set("id", d.Id()); err != nil {
		return err
	}
	if err := d.Set("name", ag.Name); err != nil {
		return err
	}

	return nil
}
