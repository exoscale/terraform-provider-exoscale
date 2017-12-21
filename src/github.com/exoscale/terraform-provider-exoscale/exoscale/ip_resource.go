package exoscale

import (
	"fmt"
	"log"
	"net/url"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
)

func ipAddressResource() *schema.Resource {
	return &schema.Resource{
		Create: ipCreate,
		Exists: ipExists,
		Read:   ipRead,
		Delete: ipDelete,

		Schema: map[string]*schema.Schema{
			"ip": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"zone": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func ipCreate(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	zoneName := d.Get("zone").(string)

	params := url.Values{}
	params.Set("name", zoneName)
	zones, err := client.GetZones(params)

	if len(zones) != 1 {
		return fmt.Errorf("Invalid zone: %s", zoneName)
	}

	profile := egoscale.IpAddressProfile{
		Zone: zones[0].Id,
	}

	ipAddress, err := client.CreateIpAddress(profile, async)
	if err != nil {
		return err
	}

	d.SetId(ipAddress.Id)
	d.Set("ip", ipAddress.IpAddress)

	return nil
}

func ipExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := GetComputeClient(meta)
	params := url.Values{}
	params.Set("id", d.Id())
	ipAddresses, err := client.ListPublicIpAddresses(params)
	return err == nil && len(ipAddresses) == 1, nil
}

func ipRead(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	params := url.Values{}
	params.Set("id", d.Id())
	ipAddress, err := client.ListPublicIpAddresses(params)
	if err != nil {
		return err
	}

	if len(ipAddress) != 1 {
		return fmt.Errorf("IP Address not found: %s (%s)", d.Id(), d.Get("ip"))
	}

	ip := ipAddress[0]

	d.Set("ip", ip.IpAddress)
	d.Set("zone", ip.ZoneName)

	return nil
}

func ipDelete(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	err := client.DestroyIpAddress(d.Id(), async)
	if err != nil {
		return err
	}

	log.Printf("Deleted ip id: %s\n", d.Id())
	return nil
}
