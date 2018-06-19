package exoscale

import (
	"context"
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

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},

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
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	virtualMachineID := d.Get("compute_id").(string)

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

		ip := resp.(*egoscale.AddIPToNicResponse).NicSecondaryIP
		ip.NicID = nic.ID
		ip.NetworkID = nic.NetworkID

		return applySecondaryIP(d, &ip)
	}

	return fmt.Errorf("No default NIC found for %v", virtualMachineID)
}

func existsSecondaryIP(d *schema.ResourceData, meta interface{}) (bool, error) {
	ip, err := getSecondaryIP(d, meta)
	if err != nil {
		e := handleNotFound(d, err)
		return d.Id() != "", e
	}

	return ip != nil, nil
}

func readSecondaryIP(d *schema.ResourceData, meta interface{}) error {
	ip, err := getSecondaryIP(d, meta)
	if err != nil {
		return handleNotFound(d, err)
	}

	if ip != nil {
		err = applySecondaryIP(d, ip)
		if err != nil {
			return err
		}
	} else {
		d.SetId("")
	}
	return nil
}

func getSecondaryIP(d *schema.ResourceData, meta interface{}) (*egoscale.NicSecondaryIP, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	client := GetComputeClient(meta)

	ipAddress := d.Get("ip_address").(string)
	virtualMachineID := d.Get("compute_id").(string)
	nicID := d.Get("nic_id").(string)

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

				return &ip, nil
			}
		}
	}

	return nil, nil
}

func deleteSecondaryIP(d *schema.ResourceData, meta interface{}) error {
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

	return client.BooleanRequestWithContext(ctx, &egoscale.RemoveIPFromNic{
		ID: ip.ID,
	})
}

func applySecondaryIP(d *schema.ResourceData, secondaryIP *egoscale.NicSecondaryIP) error {

	d.SetId(fmt.Sprintf("%s_%s", secondaryIP.NicID, secondaryIP.IPAddress.String()))

	if secondaryIP.IPAddress != nil {
		d.Set("ip_address", secondaryIP.IPAddress.String())
	} else {
		d.Set("ip_address", "")
	}

	d.Set("network_id", secondaryIP.NetworkID)
	d.Set("nic_id", secondaryIP.NicID)

	return nil
}
