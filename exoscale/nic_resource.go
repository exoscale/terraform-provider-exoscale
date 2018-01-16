package exoscale

import (
	"fmt"
	"net"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform/helper/schema"
)

func nicResource() *schema.Resource {
	return &schema.Resource{
		Create: createNic,
		Exists: existsNic,
		Read:   readNic,
		Delete: deleteNic,

		Importer: &schema.ResourceImporter{
			State: importNic,
		},

		Schema: map[string]*schema.Schema{
			"compute_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"network_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"nic_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ip_address": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "IP address",
				ValidateFunc: StringIPv4,
			},
			"netmask": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"gateway": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"mac_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func createNic(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	var ip net.IP
	if i, ok := d.GetOk("ip_address"); ok {
		ip = net.ParseIP(i.(string))
	}

	networkID := d.Get("network_id").(string)

	resp, err := client.AsyncRequest(&egoscale.AddNicToVirtualMachine{
		NetworkID:        networkID,
		VirtualMachineID: d.Get("virtual_machine_id").(string),
		IPAddress:        ip,
	}, async)

	if err != nil {
		return err
	}

	vm := resp.(*egoscale.AddNicToVirtualMachineResponse).VirtualMachine
	nic := vm.NicByNetworkID(networkID)

	d.SetId(nic.ID)
	return readNic(d, meta)
}

func readNic(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	resp, err := client.Request(&egoscale.ListNics{
		NicID: d.Id(),
	})

	if err != nil {
		return err
	}

	nics := resp.(*egoscale.ListNicsResponse)
	if nics.Count == 0 {
		return fmt.Errorf("No nic found for ID: %s", d.Id())
	}

	nic := nics.Nic[0]
	return applyNic(d, *nic)
}

func existsNic(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := GetComputeClient(meta)
	resp, err := client.Request(&egoscale.ListNics{
		NicID: d.Id(),
	})

	if err != nil {
		// XXX handle 431
		return false, err
	}

	nics := resp.(*egoscale.ListNicsResponse)
	if nics.Count == 0 {
		d.SetId("")
		return false, nil
	}

	return true, nil
}

func deleteNic(d *schema.ResourceData, meta interface{}) error {
	client := GetComputeClient(meta)
	async := meta.(BaseConfig).async

	resp, err := client.AsyncRequest(&egoscale.RemoveNicFromVirtualMachine{
		NicID:            d.Id(),
		VirtualMachineID: d.Get("virtual_machine_id").(string),
	}, async)

	if err != nil {
		return err
	}

	vm := resp.(*egoscale.RemoveNicFromVirtualMachineResponse).VirtualMachine
	nic := vm.NicByNetworkID(d.Get("network_id").(string))
	if nic != nil {
		return fmt.Errorf("Failed removing NIC %s from instance %s", d.Id(), vm.ID)
	}

	d.SetId("")
	return nil
}

func importNic(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	if err := readNic(d, meta); err != nil {
		return nil, err
	}

	resources := make([]*schema.ResourceData, 1)
	resources[0] = d
	return resources, nil
}

func applyNic(d *schema.ResourceData, nic egoscale.Nic) error {
	d.SetId(nic.ID)
	d.Set("virtual_machine_id", nic.VirtualMachineID)
	d.Set("network_id", nic.NetworkID)
	d.Set("ip_address", nic.IPAddress.String())
	d.Set("netmask", nic.Netmask.String())
	d.Set("gateway", nic.Gateway.String())
	d.Set("mac_address", nic.MacAddress)

	return nil
}
