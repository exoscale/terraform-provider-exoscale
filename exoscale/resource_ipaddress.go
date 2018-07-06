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
	d.SetId(elasticIP.ID)
	d.Set("ip_address", elasticIP.IPAddress.String())

	if cmd := createTags(d, "tags", elasticIP.ResourceType()); cmd != nil {
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

	resp, err := client.RequestWithContext(ctx, &egoscale.ListPublicIPAddresses{
		ID: d.Id(),
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

	// This permits to import a resource using the IP Address rather than using the ID.
	id := d.Id()
	ip := net.ParseIP(id)
	if ip != nil {
		id = ""
	}

	ipAddress := &egoscale.IPAddress{
		ID:        id,
		IPAddress: ip,
		IsElastic: true,
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

	return client.DeleteWithContext(ctx, &egoscale.IPAddress{
		ID: d.Id(),
	})
}

func applyElasticIP(d *schema.ResourceData, ip *egoscale.IPAddress) error {
	d.SetId(ip.ID)
	d.Set("ip_address", ip.IPAddress.String())
	d.Set("zone", ip.ZoneName)

	// tags
	tags := make(map[string]interface{})
	for _, tag := range ip.Tags {
		tags[tag.Key] = tag.Value
	}
	d.Set("tags", tags)

	return nil
}
