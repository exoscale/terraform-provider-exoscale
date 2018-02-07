package exoscale

import (
	"fmt"
	"net"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
)

func secondaryIPResource() *schema.Resource {
	return &schema.Resource{
		Create: createSecondaryIP,
		Exists: existsSecondaryIP,
		Read:   readSecondaryIP,
		Delete: deleteSecondaryIP,

		Schema: map[string]*schema.Schema{
			"compute_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ip_address": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Elastic IP address",
				ValidateFunc: ValidateIPv4String,
			},
			"nic_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"network_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func createSecondaryIP(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	virtualMachineID := d.Get("compute_id").(string)

	resp, err := client.Request(&egoscale.ListNics{
		VirtualMachineID: virtualMachineID,
	})
	if err != nil {
		return err
	}

	nics := resp.(*egoscale.ListNicsResponse)
	if nics.Count == 0 {
		return fmt.Errorf("The VM has no NIC %v", virtualMachineID)
	}

	// XXX Fragile
	nic := nics.Nic[0]
	resp, err = client.AsyncRequest(&egoscale.AddIPToNic{
		NicID:     nic.ID,
		IPAddress: net.ParseIP(d.Get("ip_address").(string)),
	}, async)
	if err != nil {
		return err
	}

	secondaryIP := resp.(*egoscale.AddIPToNicResponse).NicSecondaryIP

	d.SetId(secondaryIP.ID)
	// XXX this is fragile
	d.Set("nic_id", nic.ID)
	return nil
}

func existsSecondaryIP(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := GetComputeClient(meta)

	nicID := d.Get("nic_id").(string)
	virtualMachineID := d.Get("compute_id").(string)
	resp, err := client.Request(&egoscale.ListNics{
		NicID:            nicID,
		VirtualMachineID: virtualMachineID,
	})

	if err != nil {
		// XXX Check the root cause of that error to tell
		//     using pkg/errors.
		return err != nil, err
	}

	nics := resp.(*egoscale.ListNicsResponse)
	if nics.Count == 0 {
		return false, nil
	}

	return true, nil
}

func readSecondaryIP(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)

	nicID := d.Get("nic_id").(string)
	virtualMachineID := d.Get("compute_id").(string)
	resp, err := client.Request(&egoscale.ListNics{
		NicID:            nicID,
		VirtualMachineID: virtualMachineID,
	})

	nics := resp.(*egoscale.ListNicsResponse)
	if nics.Count == 0 {
		// No nics, means the VM is gone.
		d.SetId("")
		return nil
	}

	if err != nil {
		return err
	}

	// XXX why Nic[0]?
	for _, ip := range nics.Nic[0].SecondaryIP {
		if ip.NicID == nicID {
			d.SetId(ip.ID)
			if ip.IPAddress != nil {
				d.Set("ip_address", ip.IPAddress.String())
			} else {
				d.Set("ip_address", "")
			}

			return nil
		}
	}

	d.SetId("")
	return nil
}

func deleteSecondaryIP(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	return client.BooleanAsyncRequest(&egoscale.RemoveIPFromNic{
		ID: d.Id(),
	}, async)
}

func applySecondaryIP(d *schema.ResourceData, secondaryIP egoscale.NicSecondaryIP) {
	d.SetId(secondaryIP.ID)
	d.Set("ip_address", secondaryIP.IPAddress.String())
	d.Set("network_id", secondaryIP.NetworkID)
	d.Set("nic_id", secondaryIP.NicID)
}
