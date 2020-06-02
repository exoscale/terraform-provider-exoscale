package exoscale

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceNICIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_nic")
}

func resourceNIC() *schema.Resource {
	return &schema.Resource{
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
				Computed:     true,
				Description:  "IP address",
				ValidateFunc: ValidateIPv4String(),
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

		Create: resourceNICCreate,
		Read:   resourceNICRead,
		Update: resourceNICUpdate,
		Delete: resourceNICDelete,
		Exists: resourceNICExists,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceNICCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning create", resourceNICIDString(d))

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
		return fmt.Errorf("NIC addition didn't create a NIC for Network %s", networkID)
	}

	d.SetId(nic.ID.String())

	log.Printf("[DEBUG] %s: create finished successfully", resourceNICIDString(d))

	return resourceNICRead(d, meta)
}

func resourceNICRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning read", resourceNICIDString(d))

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

	log.Printf("[DEBUG] %s: read finished successfully", resourceNICIDString(d))

	return resourceNICApply(d, *n)
}

func resourceNICExists(d *schema.ResourceData, meta interface{}) (bool, error) {
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

func resourceNICUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning update", resourceNICIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	client := GetComputeClient(meta)

	id, err := egoscale.ParseUUID(d.Id())
	if err != nil {
		return err
	}

	if d.HasChange("ip_address") {
		_, n := d.GetChange("ip_address")

		// We should set the IP only if the new IP is not null
		// and if the IP has changed
		if n.(string) != "" {

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
	}

	d.Partial(false)

	log.Printf("[DEBUG] %s: update finished successfully", resourceNICIDString(d))

	return resourceNICRead(d, meta)
}

func resourceNICDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning delete", resourceNICIDString(d))

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
		return fmt.Errorf("failed to remove NIC %s from instance %s", d.Id(), vm.ID)
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceNICIDString(d))

	return nil
}

func resourceNICApply(d *schema.ResourceData, nic egoscale.Nic) error {
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
