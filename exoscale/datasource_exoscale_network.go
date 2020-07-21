package exoscale

import (
	"context"
	"errors"
	"fmt"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceNetwork() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"zone": {
				Type:        schema.TypeString,
				Description: "Name of the zone",
				Required:    true,
			},
			"id": {
				Type:          schema.TypeString,
				Description:   "ID of the network",
				Optional:      true,
				ConflictsWith: []string{"name"},
			},
			"name": {
				Type:          schema.TypeString,
				Description:   "Name of the network",
				Optional:      true,
				ConflictsWith: []string{"id"},
			},

			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"start_ip": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"end_ip": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"netmask": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},

		Read: dataSourceNetworkRead,
	}
}

func dataSourceNetworkRead(d *schema.ResourceData, meta interface{}) error {
	var (
		networkID   *egoscale.UUID
		networkName string
	)

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	zoneName := d.Get("zone").(string)
	zone, err := getZoneByName(ctx, client, zoneName)
	if err != nil {
		return err
	}

	_, byName := d.GetOk("name")
	_, byID := d.GetOk("id")
	if !byName && !byID {
		return errors.New("either name or id must be specified")
	}

	if byName {
		networkName = d.Get("name").(string)
	}
	if byID {
		if networkID, err = egoscale.ParseUUID(d.Get("id").(string)); err != nil {
			return fmt.Errorf("invalid value for id: %s", err)
		}
	}

	resp, err := client.ListWithContext(ctx, &egoscale.ListNetworks{ZoneID: zone.ID})
	if err != nil {
		return fmt.Errorf("networks listing failed: %s", err)
	}

	var network *egoscale.Network
	for _, v := range resp {
		net := v.(*egoscale.Network)

		// If search criterion is an unique ID, return the first (i.e. only) match
		if byID && net.ID.Equal(*networkID) {
			network = net
			break
		}

		// If search criterion is a name, check that there isn't multiple networks named
		// identically before returning a match
		if net.Name == networkName {
			// We already found a match before -> multiple results
			if network != nil {
				return fmt.Errorf("found multiple networks named %q, please specify a unique ID instead", net.Name)
			}
			network = net
		}
	}
	if network == nil {
		return errors.New("network not found")
	}

	d.SetId(network.ID.String())

	if err := d.Set("id", d.Id()); err != nil {
		return err
	}
	if err := d.Set("name", network.Name); err != nil {
		return err
	}
	if err := d.Set("description", network.DisplayText); err != nil {
		return err
	}

	if network.StartIP != nil && network.EndIP != nil && network.Netmask != nil {
		if err := d.Set("start_ip", network.StartIP.String()); err != nil {
			return err
		}
		if err := d.Set("end_ip", network.EndIP.String()); err != nil {
			return err
		}
		if err := d.Set("netmask", network.Netmask.String()); err != nil {
			return err
		}
	} else {
		d.Set("start_ip", "") // nolint: errcheck
		d.Set("end_ip", "")   // nolint: errcheck
		d.Set("netmask", "")  // nolint: errcheck
	}

	return nil
}
