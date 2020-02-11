package exoscale

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/helper/validation"
)

func resourceNetworkIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_network")
}

func resourceNetwork() *schema.Resource {
	s := map[string]*schema.Schema{
		"zone": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"network_offering": {
			Type:       schema.TypeString,
			Optional:   true,
			Deprecated: "This attribute is deprecated, please remove it from your configuration.",
		},
		"name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"display_text": { // TODO: rename to "description"
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
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
			ValidateFunc: validation.SingleIP(),
		},
	}

	addTags(s, "tags")

	return &schema.Resource{
		Schema: s,

		Create: resourceNetworkCreate,
		Read:   resourceNetworkRead,
		Update: resourceNetworkUpdate,
		Delete: resourceNetworkDelete,
		Exists: resourceNetworkExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning create", resourceNetworkIDString(d))

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

	startIP := net.ParseIP(d.Get("start_ip").(string))
	endIP := net.ParseIP(d.Get("end_ip").(string))
	netmask := net.ParseIP(d.Get("netmask").(string))
	if (startIP == nil && endIP != nil) || (startIP != nil && endIP == nil) {
		return errors.New("start_ip and end_ip must be both specified")
	} else if (startIP != nil && endIP != nil) && netmask == nil {
		return errors.New("netmask must be specified with start_ip and end_ip")
	}

	req := &egoscale.CreateNetwork{
		Name:        name,
		DisplayText: displayText,
		ZoneID:      zone.ID,
		StartIP:     startIP,
		EndIP:       endIP,
		Netmask:     netmask,
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

	log.Printf("[DEBUG] %s: create finished successfully", resourceNetworkIDString(d))

	return resourceNetworkRead(d, meta)
}

func resourceNetworkRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning read", resourceNetworkIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	networks, err := resourceNetworkFind(ctx, d, meta)
	if err != nil {
		return err
	}
	if networks.Count == 0 {
		return fmt.Errorf("no network found for ID %s", d.Id())
	}

	network := networks.Network[0]

	log.Printf("[DEBUG] %s: read finished successfully", resourceNetworkIDString(d))

	return resourceNetworkApply(d, &network)
}

func resourceNetworkFind(ctx context.Context, d *schema.ResourceData, meta interface{}) (*egoscale.ListNetworksResponse, error) {
	client := GetComputeClient(meta)
	id := egoscale.MustParseUUID(d.Id())

	r, err := client.RequestWithContext(ctx, &egoscale.ListZones{})
	if err != nil {
		return nil, err
	}
	zones := r.(*egoscale.ListZonesResponse).Zone

	var resp interface{}
	for _, zone := range zones {
		resp, err = client.RequestWithContext(ctx, &egoscale.ListNetworks{
			ID:     id,
			ZoneID: zone.ID,
		})
		if r, ok := err.(*egoscale.ErrorResponse); ok && r.ErrorCode == egoscale.ParamError {
			continue
		} else if ok && r.ErrorCode != egoscale.NotFound {
			return nil, err
		}

		if resp.(*egoscale.ListNetworksResponse).Count > 0 {
			return resp.(*egoscale.ListNetworksResponse), nil
		}
	}

	return nil, fmt.Errorf("no network found for ID %s", id)
}

func resourceNetworkExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	networks, err := resourceNetworkFind(ctx, d, meta)
	if err != nil {
		return false, err
	}

	if networks.Count == 0 {
		d.SetId("")
		return false, nil
	}

	return true, nil
}

func resourceNetworkUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning update", resourceNetworkIDString(d))

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

	log.Printf("[DEBUG] %s: update finished successfully", resourceNetworkIDString(d))

	return resourceNetworkRead(d, meta)
}

func resourceNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning delete", resourceNetworkIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	network := &egoscale.DeleteNetwork{ID: id}

	if err = client.BooleanRequestWithContext(ctx, network); err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceNetworkIDString(d))

	return nil
}

func resourceNetworkApply(d *schema.ResourceData, network *egoscale.Network) error {
	d.SetId(network.ID.String())
	if err := d.Set("name", network.Name); err != nil {
		return err
	}

	if err := d.Set("display_text", network.DisplayText); err != nil {
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
