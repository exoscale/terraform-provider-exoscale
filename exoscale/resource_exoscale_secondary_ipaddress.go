package exoscale

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceSecondaryIPAddressIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_secondary_ipaddress")
}

func resourceSecondaryIPAddress() *schema.Resource {
	return &schema.Resource{
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
				ValidateFunc: ValidateIPv4String(),
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

		Create: resourceSecondaryIPAddressCreate,
		Read:   resourceSecondaryIPAddressRead,
		Delete: resourceSecondaryIPAddressDelete,
		Exists: resourceSecondaryIPAddressExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceSecondaryIPAddressCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning create", resourceSecondaryIPAddressIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	virtualMachineID, err := egoscale.ParseUUID(d.Get("compute_id").(string))
	if err != nil {
		return err
	}

	resp, err := client.RequestWithContext(ctx, &egoscale.ListNics{
		VirtualMachineID: virtualMachineID,
	})
	if err != nil {
		return err
	}

	nics := resp.(*egoscale.ListNicsResponse)
	if nics.Count == 0 {
		return fmt.Errorf("The VM has no NIC %v", virtualMachineID)
	}

	for i := range nics.Nic {
		nic := nics.Nic[i]

		if !nic.IsDefault {
			continue
		}

		resp, err = client.RequestWithContext(ctx, &egoscale.AddIPToNic{
			NicID:     nic.ID,
			IPAddress: net.ParseIP(d.Get("ip_address").(string)),
		})

		if err != nil {
			return err
		}

		ip := resp.(*egoscale.NicSecondaryIP)

		d.SetId(fmt.Sprintf("%s_%s", ip.NicID, ip.IPAddress.String()))

		if err := d.Set("compute_id", virtualMachineID.String()); err != nil {
			return err
		}
		if err := d.Set("nic_id", ip.NicID.String()); err != nil {
			return err
		}

		log.Printf("[DEBUG] %s: create finished successfully", resourceSecondaryIPAddressIDString(d))

		return resourceSecondaryIPAddressRead(d, meta)
	}

	return fmt.Errorf("No default NIC found for %v", virtualMachineID)
}

func resourceSecondaryIPAddressExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ip, err := getSecondaryIP(d, meta)
	if err != nil {
		e := handleNotFound(d, err)
		return d.Id() != "", e
	}

	return ip != nil, nil
}

func resourceSecondaryIPAddressRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning read", resourceSecondaryIPAddressIDString(d))

	ip, err := getSecondaryIP(d, meta)
	if err != nil {
		return handleNotFound(d, err)
	}

	if ip != nil {
		err = resourceSecondaryIPAddressApply(d, ip)
		if err != nil {
			return err
		}
	} else {
		d.SetId("")
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceSecondaryIPAddressIDString(d))

	return nil
}

func getSecondaryIP(d *schema.ResourceData, meta interface{}) (*egoscale.NicSecondaryIP, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	ipAddress := d.Get("ip_address").(string)
	virtualMachine := d.Get("compute_id").(string)
	nic := d.Get("nic_id").(string)

	var virtualMachineID *egoscale.UUID
	var nicID *egoscale.UUID

	if virtualMachine == "" {
		id := d.Id()
		infos := strings.SplitN(id, "_", 2)
		if len(infos) != 2 {
			return nil, errors.New("import requires <nicid>_<ipaddress>")
		}

		var errUUID error
		nicID, errUUID := egoscale.ParseUUID(infos[0])
		if errUUID != nil {
			return nil, errUUID
		}

		ipAddress = infos[1]
		addr := net.ParseIP(ipAddress)
		if addr == nil {
			return nil, fmt.Errorf("not a valid ipaddress, got %q", ipAddress)
		}

		req := &egoscale.IPAddress{
			IPAddress: addr,
			IsElastic: true,
		}

		resp, err := client.GetWithContext(ctx, req)
		if err != nil {
			return nil, err
		}
		ip := resp.(*egoscale.IPAddress)

		// This is a hack for importing a VM
		reqVms := &egoscale.ListVirtualMachines{
			ZoneID: ip.ZoneID,
		}

		var n *egoscale.Nic
		client.PaginateWithContext(ctx, reqVms, func(v interface{}, e error) bool {
			if e != nil {
				err = e
				return false
			}

			vm := v.(*egoscale.VirtualMachine)
			n = vm.NicByID(*nicID)
			return n == nil
		})

		if err != nil {
			return nil, err
		}

		virtualMachineID = n.VirtualMachineID
	} else {
		var err error
		virtualMachineID, err = egoscale.ParseUUID(virtualMachine)
		if err != nil {
			return nil, err
		}

		if nic == "" {
			return nil, nil
		}

		nicID, err = egoscale.ParseUUID(nic)
		if err != nil {
			return nil, err
		}
	}

	ns, err := client.ListWithContext(ctx, &egoscale.Nic{
		ID:               nicID,
		VirtualMachineID: virtualMachineID,
	})

	if err != nil {
		// XXX fugly hack in case the VM doesn't exist anymore
		if r, ok := err.(*egoscale.ErrorResponse); ok {
			if r.ErrorCode == egoscale.InternalError && r.ErrorText == "Virtual machine id does not exist" {
				return nil, nil
			}
		}

		e := handleNotFound(d, err)
		return nil, e
	}

	if len(ns) == 0 {
		// No nics, means the VM is gone.
		return nil, nil
	}

	for _, n := range ns {
		nic, ok := n.(*egoscale.Nic)
		if !ok {
			continue
		}

		if !nic.IsDefault {
			continue
		}

		for _, ip := range nic.SecondaryIP {
			if ip.IPAddress != nil && ipAddress == ip.IPAddress.String() {
				ip.NicID = nic.ID
				ip.NetworkID = nic.NetworkID
				ip.VirtualMachineID = virtualMachineID

				return &ip, nil
			}
		}
	}

	return nil, nil
}

func resourceSecondaryIPAddressDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning delete", resourceSecondaryIPAddressIDString(d))

	ip, err := getSecondaryIP(d, meta)
	if err != nil {
		return err
	}

	// ip is already gone
	if ip == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	if err := client.BooleanRequestWithContext(ctx, &egoscale.RemoveIPFromNic{ID: ip.ID}); err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceSecondaryIPAddressIDString(d))

	return nil
}

func resourceSecondaryIPAddressApply(d *schema.ResourceData, secondaryIP *egoscale.NicSecondaryIP) error {

	d.SetId(fmt.Sprintf("%s_%s", secondaryIP.NicID, secondaryIP.IPAddress.String()))

	if secondaryIP.VirtualMachineID != nil {
		if err := d.Set("compute_id", secondaryIP.VirtualMachineID.String()); err != nil {
			return err
		}
	}

	ipAddress := ""

	if secondaryIP.IPAddress != nil {
		ipAddress = secondaryIP.IPAddress.String()
	}
	if err := d.Set("ip_address", ipAddress); err != nil {
		return err
	}

	if err := d.Set("network_id", secondaryIP.NetworkID.String()); err != nil {
		return err
	}
	if err := d.Set("nic_id", secondaryIP.NicID.String()); err != nil {
		return err
	}

	return nil
}
