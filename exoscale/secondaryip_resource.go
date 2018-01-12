package exoscale

import (
	"fmt"

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
			"nic_id": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
			"elasticip": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Elastic IP address",
				ValidateFunc: StringIPAddress(),
			},
		},
	}
}

func createSecondaryIP(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	virtualMachineID := d.Get("compute_id").(string)
	ipAddress := d.Get("elasticip").(string)

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

	secondaryIP, err := client.AddIPToNic(nics.Nic[0].ID, ipAddress, async)
	if err != nil {
		return err
	}

	d.SetId(secondaryIP.ID)
	d.Set("nic_id", nics.Nic[0].ID)
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

	for _, ip := range nics.Nic[0].SecondaryIP {
		if ip.NicID == nicID {
			d.SetId(ip.ID)
			d.Set("elasticip", ip.IPAddress)
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
