package exoscale

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
)

func elasticIPResource() *schema.Resource {
	s := map[string]*schema.Schema{
		"ip_address": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"zone": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "Name of the Data-Center",
		},
	}

	addTags(s, "tags")

	return &schema.Resource{
		Create: createElasticIP,
		Read:   readElasticIP,
		Update: updateElasticIP,
		Exists: existsElasticIP,
		Delete: deleteElasticIP,

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

func createElasticIP(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	zoneName := d.Get("zone").(string)

	zone, err := getZoneByName(ctx, client, zoneName)
	if err != nil {
		return err
	}

	req := &egoscale.AssociateIPAddress{
		ZoneID: zone.ID,
	}

	resp, err := client.RequestWithContext(ctx, req)
	if err != nil {
		return err
	}

	elasticIP, ok := resp.(*egoscale.IPAddress)
	if !ok {
		return fmt.Errorf("wrong type: an IPAddress was expected, got %T", resp)
	}
	d.SetId(elasticIP.ID.String())
	if err := d.Set("ip_address", elasticIP.IPAddress.String()); err != nil {
		return err
	}

	cmd, err := createTags(d, "tags", elasticIP.ResourceType())
	if err != nil {
		return err
	}

	if cmd != nil {
		if err := client.BooleanRequestWithContext(ctx, cmd); err != nil {
			// Attempting to destroy the freshly created ip address
			e := client.BooleanRequestWithContext(ctx, &egoscale.DisassociateIPAddress{
				ID: elasticIP.ID,
			})

			if e != nil {
				log.Printf("[WARNING] Failure to create the tags, but the ip address was created. %v", e)
			}

			return err
		}
	}

	return readElasticIP(d, meta)
}

func existsElasticIP(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return false, err
	}

	resp, err := client.RequestWithContext(ctx, &egoscale.ListPublicIPAddresses{
		ID: id,
	})

	if err != nil {
		e := handleNotFound(d, err)
		return d.Id() != "", e
	}

	elasticIPes := resp.(*egoscale.ListPublicIPAddressesResponse)
	return elasticIPes.Count == 1, nil
}

func readElasticIP(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	ipAddress := &egoscale.IPAddress{
		IsElastic: true,
	}

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		ip := net.ParseIP(d.Id())
		if ip == nil {
			return fmt.Errorf("%q is neither a valid ID or IP address", d.Id())
		}
		ipAddress.IPAddress = ip
	} else {
		ipAddress.ID = id
	}

	if err := client.GetWithContext(ctx, ipAddress); err != nil {
		return handleNotFound(d, err)
	}

	return applyElasticIP(d, ipAddress)
}

func updateElasticIP(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	client := GetComputeClient(meta)

	d.Partial(true)

	requests, err := updateTags(d, "tags", new(egoscale.IPAddress).ResourceType())
	if err != nil {
		return err
	}

	for _, req := range requests {
		_, err := client.RequestWithContext(ctx, req)
		if err != nil {
			return err
		}
	}

	err = readElasticIP(d, meta)
	if err != nil {
		return err
	}

	d.SetPartial("tags")
	d.Partial(false)

	return err
}

func deleteElasticIP(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	return client.DeleteWithContext(ctx, &egoscale.IPAddress{
		ID: id,
	})
}

func applyElasticIP(d *schema.ResourceData, ip *egoscale.IPAddress) error {
	d.SetId(ip.ID.String())
	if err := d.Set("ip_address", ip.IPAddress.String()); err != nil {
		return err
	}
	if err := d.Set("zone", ip.ZoneName); err != nil {
		return err
	}

	// tags
	tags := make(map[string]interface{})
	for _, tag := range ip.Tags {
		tags[tag.Key] = tag.Value
	}
	if err := d.Set("tags", tags); err != nil {
		return err
	}

	return nil
}
