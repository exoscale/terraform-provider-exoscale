package exoscale

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func networkResource() *schema.Resource {
	s := map[string]*schema.Schema{
		"name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"display_text": {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		"network_offering": {
			Type:     schema.TypeString,
			Required: true,
		},
		"zone": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"cidr": {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.CIDRNetwork(0, 32),
		},
		"netmask": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"gateway": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"dns1": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"dns2": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"network_domain": {
			Type:     schema.TypeString,
			Computed: true,
		},
	}

	addTags(s, "tags")

	return &schema.Resource{
		Create: createNetwork,
		Exists: existsNetwork,
		Read:   readNetwork,
		Update: updateNetwork,
		Delete: deleteNetwork,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},

		Schema: s,
	}
}

func createNetwork(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	name := d.Get("name").(string)
	displayText := d.Get("display_text").(string)
	if displayText == "" {
		displayText = name
	}

	zoneName := d.Get("zone").(string)
	zone, err := getZoneByName(ctx, client, zoneName)
	if err != nil {
		return err
	}

	networkName := d.Get("network_offering").(string)
	networkOffering, err := getNetworkOfferingByName(ctx, client, networkName)
	if err != nil {
		return err
	}

	if networkOffering.SpecifyIPRanges {
		return fmt.Errorf("SpecifyIPRanges is not yet supported.")
	}

	netmask := net.IPv4zero
	gateway := net.IPv4zero

	if cidr, ok := d.GetOk("cidr"); ok {
		c := cidr.(string)
		ip, ipnet, err := net.ParseCIDR(c)
		if err != nil {
			return err
		}

		if ip.To4() == nil {
			return fmt.Errorf("Provided cidr %s is not an IPv4 address", c)
		}

		// subnet address
		subnetIP := ip.Mask(ipnet.Mask)
		// netmask
		netmask = net.IPv4(
			ipnet.Mask[0],
			ipnet.Mask[1],
			ipnet.Mask[2],
			ipnet.Mask[3])

		// last address
		gateway = net.IPv4(
			subnetIP[0]+^ipnet.Mask[0],
			subnetIP[1]+^ipnet.Mask[1],
			subnetIP[2]+^ipnet.Mask[2],
			subnetIP[3]+^ipnet.Mask[3])
	}

	resp, err := client.RequestWithContext(ctx, &egoscale.CreateNetwork{
		Name:              name,
		DisplayText:       displayText,
		NetworkOfferingID: networkOffering.ID,
		ZoneID:            zone.ID,
		Netmask:           netmask,
		Gateway:           gateway,
	})

	if err != nil {
		return err
	}

	network := resp.(*egoscale.Network)
	d.SetId(network.ID)

	if cmd := createTags(d, "tags", network.ResourceType()); cmd != nil {
		if err := client.BooleanRequestWithContext(ctx, cmd); err != nil {
			// Attempting to destroy the freshly created network
			e := client.BooleanRequestWithContext(ctx, &egoscale.DeleteNetwork{
				ID: network.ID,
			})

			if e != nil {
				log.Printf("[WARNING] Failure to create the tags, but the network was created. %v", e)
			}

			return err
		}
	}

	return readNetwork(d, meta)
}

func readNetwork(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)
	resp, err := client.RequestWithContext(ctx, &egoscale.ListNetworks{
		ID: d.Id(),
	})

	if err != nil {
		return handleNotFound(d, err)
	}

	networks := resp.(*egoscale.ListNetworksResponse)
	if networks.Count == 0 {
		return fmt.Errorf("No network found for ID: %s", d.Id())
	}

	network := networks.Network[0]
	return applyNetwork(d, &network)
}

func existsNetwork(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)
	resp, err := client.RequestWithContext(ctx, &egoscale.ListNetworks{
		ID: d.Id(),
	})

	if err != nil {
		e := handleNotFound(d, err)
		return d.Id() != "", e
	}

	networks := resp.(*egoscale.ListNetworksResponse)
	if networks.Count == 0 {
		d.SetId("")
		return false, nil
	}

	return true, nil
}

func updateNetwork(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	client := GetComputeClient(meta)

	d.Partial(true)

	// Update name and display_text
	resp, err := client.RequestWithContext(ctx, &egoscale.UpdateNetwork{
		ID:          d.Id(),
		Name:        d.Get("name").(string),
		DisplayText: d.Get("display_text").(string),
	})

	if err != nil {
		return err
	}

	network := resp.(*egoscale.Network)

	err = applyNetwork(d, network)
	if err != nil {
		return err
	}

	d.SetPartial("name")
	d.SetPartial("display_text")

	// Update tags
	requests, err := updateTags(d, "tags", network.ResourceType())
	if err != nil {
		return err
	}

	for _, req := range requests {
		_, err := client.RequestWithContext(ctx, req)
		if err != nil {
			return err
		}
	}

	err = readNetwork(d, meta)
	if err != nil {
		return err
	}

	d.SetPartial("tags")

	d.Partial(false)
	return nil
}

func deleteNetwork(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	err := client.BooleanRequestWithContext(ctx, &egoscale.DeleteNetwork{
		ID: d.Id(),
	})

	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func applyNetwork(d *schema.ResourceData, network *egoscale.Network) error {
	d.SetId(network.ID)
	d.Set("name", network.Name)
	d.Set("display_text", network.DisplayText)
	d.Set("network_domain", network.NetworkDomain)
	d.Set("network_offering", network.NetworkOfferingName)
	d.Set("zone", network.ZoneName)
	d.Set("cidr", network.Cidr)

	if network.Gateway != nil {
		d.Set("gateway", network.Gateway.String())
	} else {
		d.Set("gateway", "")
	}

	if network.Netmask != nil {
		d.Set("netmask", network.Netmask.String())
	} else {
		d.Set("netmask", "")
	}

	if network.DNS1 != nil {
		d.Set("dns1", network.DNS1.String())
	} else {
		d.Set("dns1", "")
	}

	if network.DNS2 != nil {
		d.Set("dns2", network.DNS2.String())
	} else {
		d.Set("dns2", "")
	}

	// tags
	tags := make(map[string]interface{})
	for _, tag := range network.Tags {
		tags[tag.Key] = tag.Value
	}
	d.Set("tags", tags)

	return nil
}
