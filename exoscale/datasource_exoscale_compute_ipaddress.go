package exoscale

import (
	"context"
	"fmt"
	"net"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func datasourceComputeIPAddress() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"zone": {
				Type:        schema.TypeString,
				Description: "Name of the zone",
				Required:    true,
			},
			"description": {
				Type:          schema.TypeString,
				Description:   "Description of the IP",
				Optional:      true,
				ConflictsWith: []string{"ip_address", "id", "tags"},
			},
			"ip_address": {
				Type:          schema.TypeString,
				Description:   "IP Address",
				Optional:      true,
				ConflictsWith: []string{"description", "id", "tags"},
			},
			"id": {
				Type:          schema.TypeString,
				Description:   "ID of the ip",
				Optional:      true,
				ConflictsWith: []string{"description", "ip_address", "tags"},
			},
			"tags": {
				Type:          schema.TypeMap,
				Description:   "Map of tags (key: value)",
				Optional:      true,
				ConflictsWith: []string{"description", "ip_address", "id"},
				Computed:      true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},

		Read: datasourceComputeIPAddressRead,
	}
}

func datasourceComputeIPAddressRead(d *schema.ResourceData, meta interface{}) error {
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

	t, ok := d.GetOk("tags")
	switch {
	case d.Get("id").(string) != "":
		return datasourceComputeIPAddressApply(d, ips)
	case d.Get("ip_address").(string) != "":
		return datasourceComputeIPAddressApply(d, ips)
	case ok:
		ipAddrs := make([]egoscale.IPAddress, 0)
		for _, ip := range ips {
			if compareTags(ip, t.(map[string]interface{})) {
				ipAddrs = append(ipAddrs, ip)
			}
		}
		return datasourceComputeIPAddressApply(d, ipAddrs)
	case d.Get("description").(string) != "":
		ipAddrs := make([]egoscale.IPAddress, 0)
		for _, ip := range ips {
			if ip.Description == d.Get("description").(string) {
				ipAddrs = append(ipAddrs, ip)
			}
		}
		return datasourceComputeIPAddressApply(d, ipAddrs)
	}

	return fmt.Errorf(`You must set at least one attribute "id", "ip_address", "tags" or "description"`)
}

func datasourceComputeIPAddressApply(d *schema.ResourceData, ipAddresses []egoscale.IPAddress) error {
	len := len(ipAddresses)
	switch {
	case len == 0:
		return fmt.Errorf("No IP Address found")
	case len > 1:
		return fmt.Errorf("More than one IP Address found")
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

	return i > 0
}
