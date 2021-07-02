package exoscale

import (
	"context"
	"errors"
	"fmt"

	exov2 "github.com/exoscale/egoscale/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceNLB() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"zone": {
				Type:        schema.TypeString,
				Description: "Zone of the Network Load Balancer",
				Required:    true,
			},
			"id": {
				Type:          schema.TypeString,
				Description:   "ID of the Network Load Balancer",
				Optional:      true,
				ConflictsWith: []string{"name"},
			},
			"name": {
				Type:          schema.TypeString,
				Description:   "Name of the Network Load Balancer",
				Optional:      true,
				ConflictsWith: []string{"id"},
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},

		Read: dataSourceNLBRead,
	}
}

func dataSourceNLBRead(d *schema.ResourceData, meta interface{}) error {
	var (
		nlbID   string
		nlbName string
	)

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	zoneName := d.Get("zone").(string)
	zone, err := getZoneByName(ctx, client, zoneName)
	if err != nil {
		return err
	}

	_, byID := d.GetOk("id")
	_, byName := d.GetOk("name")
	switch {
	case byID:
		nlbID = d.Get("id").(string)

	case byName:
		nlbName = d.Get("name").(string)

	default:
		return errors.New("either name or id must be specified")
	}

	nlbs, err := client.ListNetworkLoadBalancers(ctx, zone.Name)
	if err != nil {
		return fmt.Errorf("Network Load Balancers listing failed: %s", err)
	}

	var nlb *exov2.NetworkLoadBalancer
	for _, n := range nlbs {
		if byID && n.ID == nlbID {
			nlb = n
			break
		}

		if byName && n.Name == nlbName {
			nlb = n
			break
		}
	}
	if nlb == nil {
		return errors.New("Network Load Balancer not found")
	}

	d.SetId(nlb.ID)

	if err := d.Set("id", d.Id()); err != nil {
		return err
	}
	if err := d.Set("name", nlb.Name); err != nil {
		return err
	}
	if err := d.Set("description", nlb.Description); err != nil {
		return err
	}
	if err := d.Set("created_at", nlb.CreatedAt.String()); err != nil {
		return err
	}
	if err := d.Set("state", nlb.State); err != nil {
		return err
	}
	if err := d.Set("ip_address", nlb.IPAddress.String()); err != nil {
		return err
	}

	return nil
}
