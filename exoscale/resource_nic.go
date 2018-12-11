package exoscale

import (
	"context"
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
		Update: updateNic,
		Delete: deleteNic,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
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
			"ip_address": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "IP address",
				ValidateFunc: ValidateIPv4String,
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
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	var ip net.IP
	if i, ok := d.GetOk("ip_address"); ok {
		ip = net.ParseIP(i.(string))
	}

	networkID, err := egoscale.ParseUUID(d.Get("network_id").(string))
	if err != nil {
		return err
	}

	vmID, err := egoscale.ParseUUID(d.Get("compute_id").(string))
	if err != nil {
		return err
	}

	resp, err := client.RequestWithContext(ctx, &egoscale.AddNicToVirtualMachine{
		NetworkID:        networkID,
		VirtualMachineID: vmID,
		IPAddress:        ip,
	})

	if err != nil {
		return err
	}

	vm := resp.(*egoscale.VirtualMachine)
	nic := vm.NicByNetworkID(*networkID)
	if nic == nil {
		return fmt.Errorf("Nic addition didn't create a NIC for Network %s", networkID)
	}

	d.SetId(nic.ID.String())
	return readNic(d, meta)
}

func readNic(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	nic := &egoscale.Nic{ID: id}

	resp, err := client.GetWithContext(ctx, nic)
	if err != nil {
		return handleNotFound(d, err)
	}

	n := resp.(*egoscale.Nic)
	return applyNic(d, *n)
}

func existsNic(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return false, err
	}

	nic := &egoscale.Nic{ID: id}

	_, err = client.GetWithContext(ctx, nic)
	if err != nil {
		e := handleNotFound(d, err)
		return d.Id() != "", e
	}

	return true, nil
}

func updateNic(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	if d.HasChange("ip_address") {
		o, n := d.GetChange("ip_address")

		if o.(string) != "" && n.(string) == "" {
			return fmt.Errorf("[ERROR] new value of %q cannot be empty. old value was %s. The resource must be recreated instead", "ip_address", o.(string))
		}

		ipAddress := net.ParseIP(n.(string))

		d.SetPartial("ip_address")

		_, err := client.RequestWithContext(ctx, egoscale.UpdateVMNicIP{
			NicID:     id,
			IPAddress: ipAddress,
		})

		if err != nil {
			return err
		}
	}

	err = readNic(d, meta)

	d.Partial(false)

	return err
}

func deleteNic(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	vmID, err := egoscale.ParseUUID(d.Get("compute_id").(string))
	if err != nil {
		return err
	}

	networkID, err := egoscale.ParseUUID(d.Get("network_id").(string))
	if err != nil {
		return err
	}

	resp, err := client.RequestWithContext(ctx, &egoscale.RemoveNicFromVirtualMachine{
		NicID:            id,
		VirtualMachineID: vmID,
	})

	if err != nil {
		return err
	}

	vm := resp.(*egoscale.VirtualMachine)
	nic := vm.NicByNetworkID(*networkID)
	if nic != nil {
		return fmt.Errorf("Failed removing NIC %s from instance %s", d.Id(), vm.ID)
	}

	d.SetId("")
	return nil
}

func applyNic(d *schema.ResourceData, nic egoscale.Nic) error {
	d.SetId(nic.ID.String())
	if err := d.Set("compute_id", nic.VirtualMachineID.String()); err != nil {
		return err
	}
	if err := d.Set("network_id", nic.NetworkID.String()); err != nil {
		return err
	}
	if err := d.Set("mac_address", nic.MACAddress.String()); err != nil {
		return err
	}

	ipAddress := ""
	if nic.IPAddress != nil {
		ipAddress = nic.IPAddress.String()
	}
	if err := d.Set("ip_address", ipAddress); err != nil {
		return err
	}

	netmask := ""
	if nic.Netmask != nil {
		netmask = nic.Netmask.String()
	}
	if err := d.Set("netmask", netmask); err != nil {
		return err
	}

	gateway := ""
	if nic.Gateway != nil {
		gateway = nic.Gateway.String()
	}
	if err := d.Set("gateway", gateway); err != nil {
		return err
	}

	return nil
}
