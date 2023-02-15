package exoscale

import (
	"context"
	"fmt"
	"net"

	"github.com/exoscale/egoscale"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceNICIDString(d general.ResourceIDStringer) string {
	return general.ResourceIDString(d, "exoscale_nic")
}

func resourceNIC() *schema.Resource {
	return &schema.Resource{
		Description: "Manage Exoscale Compute Instance Private Network Interfaces (NIC).",
		Schema: map[string]*schema.Schema{
			"compute_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The compute instance ID.",
			},
			"network_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The private network ID.",
			},
			"ip_address": {
				Type:             schema.TypeString,
				Optional:         true,
				Computed:         true,
				Description:      "The IPv4 address to request as static DHCP lease if the NIC is attached to a *managed* private network.",
				ValidateDiagFunc: validation.ToDiagFunc(validation.IsIPv4Address),
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
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The NIC MAC address.",
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
	tflog.Debug(context.Background(), "beginning create", map[string]interface{}{
		"id": resourceNICIDString(d),
	})

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

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": resourceNICIDString(d),
	})

	return resourceNICRead(d, meta)
}

func resourceNICRead(d *schema.ResourceData, meta interface{}) error {
	tflog.Debug(context.Background(), "beginning read", map[string]interface{}{
		"id": resourceNICIDString(d),
	})

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

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceNICIDString(d),
	})

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
	tflog.Debug(context.Background(), "beginning update", map[string]interface{}{
		"id": resourceNICIDString(d),
	})

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

			_, err := client.RequestWithContext(ctx, egoscale.UpdateVMNicIP{
				NicID:     id,
				IPAddress: ipAddress,
			})
			if err != nil {
				return err
			}
		}
	}

	tflog.Debug(ctx, "update finished successfully", map[string]interface{}{
		"id": resourceNICIDString(d),
	})

	return resourceNICRead(d, meta)
}

func resourceNICDelete(d *schema.ResourceData, meta interface{}) error {
	tflog.Debug(context.Background(), "beginning delete", map[string]interface{}{
		"id": resourceNICIDString(d),
	})

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

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": resourceNICIDString(d),
	})

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
