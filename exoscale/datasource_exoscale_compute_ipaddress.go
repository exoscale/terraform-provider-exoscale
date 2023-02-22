package exoscale

import (
	"context"
	"fmt"
	"net"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceComputeIPAddress() *schema.Resource {
	return &schema.Resource{
		Description: "Fetch Exoscale Elastic IPs (EIP) data.",
		Schema: map[string]*schema.Schema{
			"zone": {
				Description: "The Exoscale Zone name.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"description": {
				Description:   "The EIP description to match.",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"ip_address", "id", "tags"},
			},
			"ip_address": {
				Description:   "The EIP IPv4 address to match.",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"description", "id", "tags"},
			},
			"id": {
				Description:   "The Elastic IP (EIP) ID to match.",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"description", "ip_address", "tags"},
			},
			"tags": {
				Description:   "The EIP tags to match.",
				Type:          schema.TypeMap,
				Optional:      true,
				ConflictsWith: []string{"description", "ip_address", "id"},
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},

		Read: dataSourceComputeIPAddressRead,
	}
}

func dataSourceComputeIPAddressRead(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	zoneName := d.Get("zone").(string)
	zone, err := getZoneByName(ctx, client, zoneName)
	if err != nil {
		return err
	}

	id := d.Get("id").(string)

	var uuid *egoscale.UUID
	if id != "" {
		uuid = egoscale.MustParseUUID(id)
	}

	resp, err := client.RequestWithContext(ctx,
		egoscale.ListPublicIPAddresses{
			ZoneID:    zone.ID,
			ID:        uuid,
			IPAddress: net.ParseIP(d.Get("ip_address").(string)),
		},
	)
	if err != nil {
		return err
	}

	ips := resp.(*egoscale.ListPublicIPAddressesResponse).PublicIPAddress

	t, byTags := d.GetOk("tags")
	switch {
	case d.Get("id").(string) != "":
		return dataSourceComputeIPAddressApply(d, ips)
	case d.Get("ip_address").(string) != "":
		return dataSourceComputeIPAddressApply(d, ips)
	case byTags:
		ipAddrs := make([]egoscale.IPAddress, 0)
		for _, ip := range ips {
			if compareTags(ip, t.(map[string]interface{})) {
				ipAddrs = append(ipAddrs, ip)
			}
		}
		return dataSourceComputeIPAddressApply(d, ipAddrs)
	case d.Get("description").(string) != "":
		ipAddrs := make([]egoscale.IPAddress, 0)
		for _, ip := range ips {
			if ip.Description == d.Get("description").(string) {
				ipAddrs = append(ipAddrs, ip)
			}
		}
		return dataSourceComputeIPAddressApply(d, ipAddrs)
	}

	return fmt.Errorf(`You must set at least one attribute "id", "ip_address", "tags" or "description"`)
}

func dataSourceComputeIPAddressApply(d *schema.ResourceData, ipAddresses []egoscale.IPAddress) error {
	len := len(ipAddresses)
	switch {
	case len == 0:
		return fmt.Errorf("No matching Elastic IP found")
	case len > 1:
		return fmt.Errorf("More than one Elastic IPs found")
	}
	ip := ipAddresses[0]
	d.SetId(ip.ID.String())

	if err := d.Set("id", d.Id()); err != nil {
		return err
	}
	if err := d.Set("description", ip.Description); err != nil {
		return err
	}
	if err := d.Set("ip_address", ip.IPAddress.String()); err != nil {
		return err
	}
	tags := make(map[string]interface{})
	for _, tag := range ip.Tags {
		tags[tag.Key] = tag.Value
	}
	if err := d.Set("tags", tags); err != nil {
		return err
	}

	return nil
}

func compareTags(ip egoscale.IPAddress, t map[string]interface{}) bool {
	i := 0
	for _, tag := range ip.Tags {
		for k, v := range t {
			if tag.Key == k && tag.Value == v.(string) {
				i++
			}
		}
	}

	return i > 0 && i >= len(ip.Tags)
}
