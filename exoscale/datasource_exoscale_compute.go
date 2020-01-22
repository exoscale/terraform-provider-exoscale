package exoscale

import (
	"context"
	"errors"
	"fmt"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func dataSourceCompute() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": {
				Type:          schema.TypeString,
				Description:   "ID of the Compute instance",
				Optional:      true,
				ConflictsWith: []string{"hostname", "tags"},
			},
			"hostname": {
				Type:          schema.TypeString,
				Description:   "Hostname of the Compute instance",
				Optional:      true,
				ConflictsWith: []string{"id", "tags"},
			},
			"tags": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description:   "Map of tags (key: value)",
				Optional:      true,
				ConflictsWith: []string{"id", "hostname"},
			},
			"created": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Date when the Compute instance was created",
			},
			"zone": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the availability zone for the Compute instance",
			},
			"template": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the template for the Compute instance",
			},
			"size": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current size of the Compute instance",
			},
			"disk_size": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Size of the Compute instance disk",
			},
			"cpu": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of cpu the Compute instance is running with",
			},
			"memory": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Memory allocated for the Compute instance",
			},
			"state": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "State of the Compute instance",
			},

			"ip_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Compute instance public ipv4 address",
			},
			"ip6_address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Compute instance public ipv6 address (if ipv6 is enabled)",
			},
			"private_network_ip_addresses": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of Compute instance private IP addresses (in managed Private Networks only)",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},

		Read: dataSourceComputeRead,
	}
}

func dataSourceComputeRead(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	req := egoscale.VirtualMachine{}

	computeName, byName := d.GetOk("hostname")
	computeID, byID := d.GetOk("id")
	computeTag, byTag := d.GetOk("tags")
	switch {
	case !byName && !byID && !byTag:
		return errors.New("either hostname, id or tags must be specified")
	case computeID != "":
		var err error
		if req.ID, err = egoscale.ParseUUID(computeID.(string)); err != nil {
			return fmt.Errorf("invalid value for id: %s", err)
		}
	case byTag:
		for key, value := range computeTag.(map[string]interface{}) {
			req.Tags = append(req.Tags, egoscale.ResourceTag{
				Key:   key,
				Value: value.(string),
			})
		}
	default:
		req.Name = computeName.(string)
	}

	resp, err := client.GetWithContext(ctx, &req)
	if err != nil {
		return err
	}

	c := resp.(*egoscale.VirtualMachine)

	resp, err = client.GetWithContext(ctx, &egoscale.Volume{
		VirtualMachineID: c.ID,
		Type:             "ROOT",
	})
	if err != nil {
		return err
	}

	ds := resp.(*egoscale.Volume).Size >> 30

	resp, err = client.RequestWithContext(ctx, &egoscale.ListNics{
		VirtualMachineID: c.ID,
	})
	if err != nil {
		return err
	}
	n := resp.(*egoscale.ListNicsResponse).Nic

	return dataSourceComputeApply(d, c, n, ds)
}

func dataSourceComputeApply(d *schema.ResourceData, compute *egoscale.VirtualMachine, nics []egoscale.Nic, diskSize uint64) error {
	d.SetId(compute.ID.String())

	if err := d.Set("id", d.Id()); err != nil {
		return err
	}
	if err := d.Set("hostname", compute.Name); err != nil {
		return err
	}
	if err := d.Set("created", compute.Created); err != nil {
		return err
	}
	if err := d.Set("zone", compute.ZoneName); err != nil {
		return err
	}
	if err := d.Set("template", compute.TemplateName); err != nil {
		return err
	}
	if err := d.Set("size", compute.ServiceOfferingName); err != nil {
		return err
	}
	if err := d.Set("disk_size", diskSize); err != nil {
		return err
	}
	if err := d.Set("cpu", compute.CPUNumber); err != nil {
		return err
	}
	if err := d.Set("memory", compute.Memory); err != nil {
		return err
	}
	if err := d.Set("state", compute.State); err != nil {
		return err
	}
	if err := d.Set("ip_address", compute.IP().String()); err != nil {
		return err
	}

	tags := make(map[string]interface{})
	for _, tag := range compute.Tags {
		tags[tag.Key] = tag.Value
	}
	if err := d.Set("tags", tags); err != nil {
		return err
	}

	privateIP := make([]string, 0)
	for _, nic := range nics {
		switch {
		case nic.IsDefault && nic.IP6Address != nil:
			if err := d.Set("ip6_address", nic.IP6Address.String()); err != nil {
				return err
			}
		case !nic.IsDefault && nic.IPAddress != nil:
			privateIP = append(privateIP, nic.IPAddress.String())
		}
	}

	if len(privateIP) > 0 {
		return d.Set("private_network_ip_addresses", privateIP)
	}

	return nil
}
