package exoscale

import (
	"fmt"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
)

func networkResource() *schema.Resource {
	return &schema.Resource{
		Create: createNetwork,
		Exists: existsNetwork,
		Read:   readNetwork,
		Update: updateNetwork,
		Delete: deleteNetwork,

		Importer: &schema.ResourceImporter{
			State: importNetwork,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"display_text": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"network_offering": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"zone": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"cidr": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func createNetwork(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)

	name := d.Get("name").(string)
	displayText := d.Get("display_text").(string)
	if displayText == "" {
		displayText = name
	}

	zoneName := d.Get("zone").(string)
	zone, err := getZoneByName(client, zoneName)
	if err != nil {
		return err
	}

	networkName := d.Get("network_offering").(string)
	networkOffering, err := getNetworkOfferingByName(client, networkName)
	if err != nil {
		return err
	}

	resp, err := client.Request(&egoscale.CreateNetwork{
		Name:              name,
		DisplayText:       displayText,
		NetworkOfferingID: networkOffering.ID,
		ZoneID:            zone.ID,
	})

	if err != nil {
		return err
	}

	network := resp.(*egoscale.CreateNetworkResponse).Network

	d.SetId(network.ID)

	return readNetwork(d, meta)
}

func readNetwork(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	resp, err := client.Request(&egoscale.ListNetworks{
		ID: d.Id(),
	})

	if err != nil {
		return err
	}

	networks := resp.(*egoscale.ListNetworksResponse)
	if networks.Count == 0 {
		return fmt.Errorf("No network found for ID: %s", d.Id())
	}

	return applyNetwork(networks.Network[0], d)
}

func existsNetwork(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := GetComputeClient(meta)
	resp, err := client.Request(&egoscale.ListNetworks{
		ID: d.Id(),
	})

	if err != nil {
		// XXX handle 431
		return false, err
	}

	networks := resp.(*egoscale.ListNetworksResponse)
	if networks.Count == 0 {
		d.SetId("")
		return true, nil
	}

	return false, nil
}

func updateNetwork(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func deleteNetwork(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	err := client.BooleanAsyncRequest(&egoscale.DeleteNetwork{
		ID: d.Id(),
	}, async)

	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func importNetwork(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := readNetwork(d, meta); err != nil {
		return nil, err
	}

	resources := make([]*schema.ResourceData, 1)
	resources[0] = d
	return resources, nil
}

func applyNetwork(network *egoscale.Network, d *schema.ResourceData) error {
	d.SetId(network.ID)
	d.Set("name", network.Name)
	d.Set("display_text", network.DisplayText)
	d.Set("network_offering", network.NetworkOfferingName)
	d.Set("zone", network.ZoneName)
	d.Set("cidr", network.Cidr)

	return nil
}
