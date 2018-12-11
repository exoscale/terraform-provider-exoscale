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

const defaultNetmask = "255.255.255.0"

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
		"start_ip": {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.SingleIP(),
		},
		"end_ip": {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.SingleIP(),
		},
		"netmask": {
			Type:         schema.TypeString,
			Optional:     true,
			Description:  fmt.Sprintf("Network mask (default to %s)", defaultNetmask),
			ValidateFunc: validation.SingleIP(),
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

	startIP := net.ParseIP(d.Get("start_ip").(string))
	endIP := net.ParseIP(d.Get("end_ip").(string))
	netmask := net.ParseIP(d.Get("netmask").(string))
	if startIP == nil && endIP == nil {
		netmask = nil
	} else if netmask == nil {
		netmask = net.ParseIP(defaultNetmask)
	}

	req := &egoscale.CreateNetwork{
		Name:              name,
		DisplayText:       displayText,
		NetworkOfferingID: networkOffering.ID,
		ZoneID:            zone.ID,
		StartIP:           startIP,
		EndIP:             endIP,
		Netmask:           netmask,
	}

	resp, err := client.RequestWithContext(ctx, req)

	if err != nil {
		return err
	}

	network := resp.(*egoscale.Network)
	d.SetId(network.ID.String())

	cmd, err := createTags(d, "tags", network.ResourceType())
	if err != nil {
		return err
	}
	if cmd != nil {
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

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.RequestWithContext(ctx, &egoscale.ListNetworks{
		ID: id,
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

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return false, err
	}

	resp, err := client.RequestWithContext(ctx, &egoscale.ListNetworks{
		ID: id,
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

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	if d.HasChange("start_ip") || d.HasChange("end_ip") {
		for _, key := range []string{"start_ip", "end_ip"} {
			o, n := d.GetChange(key)
			if o.(string) != "" && n.(string) == "" {
				return fmt.Errorf("[ERROR] new value of %q cannot be empty. old value was %s. The resource must be recreated instead", key, o.(string))
			}
		}
	}

	// Update name and display_text
	updateNetwork := &egoscale.UpdateNetwork{
		ID:          id,
		Name:        d.Get("name").(string),
		DisplayText: d.Get("display_text").(string),
		StartIP:     net.ParseIP(d.Get("start_ip").(string)),
		EndIP:       net.ParseIP(d.Get("end_ip").(string)),
		Netmask:     net.ParseIP(d.Get("netmask").(string)),
	}

	// Update tags
	requests, err := updateTags(d, "tags", egoscale.Network{}.ResourceType())
	if err != nil {
		return err
	}

	requests = append(requests, updateNetwork)

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

	return nil
}

func deleteNetwork(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	if err = client.BooleanRequestWithContext(ctx, &egoscale.DeleteNetwork{ID: id}); err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func applyNetwork(d *schema.ResourceData, network *egoscale.Network) error {
	d.SetId(network.ID.String())
	if err := d.Set("name", network.Name); err != nil {
		return err
	}

	if err := d.Set("display_text", network.DisplayText); err != nil {
		return err
	}

	if err := d.Set("network_offering", network.NetworkOfferingName); err != nil {
		return err
	}

	if err := d.Set("zone", network.ZoneName); err != nil {
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

	// tags
	tags := make(map[string]interface{})
	for _, tag := range network.Tags {
		tags[tag.Key] = tag.Value
	}
	if err := d.Set("tags", tags); err != nil {
		return err
	}

	return nil
}
