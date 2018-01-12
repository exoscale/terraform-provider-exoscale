package exoscale

import (
	"fmt"
	"log"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
)

func elasticIPResource() *schema.Resource {
	return &schema.Resource{
		Create: createElasticIP,
		Read:   readElasticIP,
		Exists: existsElasticIP,
		Delete: deleteElasticIP,

		Importer: &schema.ResourceImporter{
			State: importElasticIP,
		},

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

func createElasticIP(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	zoneName := d.Get("zone").(string)

	resp, err := client.Request(&egoscale.ListZones{
		Name: zoneName,
	})
	if err != nil {
		return err
	}

	zones := resp.(*egoscale.ListZonesResponse)
	if zones.Count != 1 {
		return fmt.Errorf("Invalid zone: %s", zoneName)
	}

	req := &egoscale.AssociateIPAddress{
		ZoneID: zones.Zone[0].ID,
	}
	resp, err = client.AsyncRequest(req, async)
	if err != nil {
		return err
	}

	elasticIP := resp.(*egoscale.AssociateIPAddressResponse).IPAddress
	d.SetId(elasticIP.ID)
	d.Set("ip", elasticIP.IPAddress)

	return nil
}

func existsElasticIP(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := GetComputeClient(meta)

	resp, err := client.Request(&egoscale.ListPublicIPAddresses{
		ID: d.Id(),
	})
	if err != nil {
		return false, err
	}

	elasticIPes := resp.(*egoscale.ListPublicIPAddressesResponse)
	return elasticIPes.Count == 1, nil
}

func readElasticIP(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)

	resp, err := client.Request(&egoscale.ListPublicIPAddresses{
		ID: d.Id(),
	})
	if err != nil {
		return err
	}

	ips := resp.(*egoscale.ListPublicIPAddressesResponse)
	if ips.Count != 1 {
		return fmt.Errorf("IP Address not found: %s (%s)", d.Id(), d.Get("ip"))
	}

	ip := ips.PublicIPAddress[0]

	d.Set("ip", ip.IPAddress)
	d.Set("zone", ip.ZoneName)

	return nil
}

func deleteElasticIP(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	req := &egoscale.DisassociateIPAddress{
		ID: d.Id(),
	}
	err := client.BooleanAsyncRequest(req, async)
	if err != nil {
		return err
	}

	log.Printf("Deleted ip id: %s\n", d.Id())
	return nil
}

func importElasticIP(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := readElasticIP(d, meta); err != nil {
		return nil, err
	}

	resources := make([]*schema.ResourceData, 1)
	resources[0] = d
	return resources, nil
}
