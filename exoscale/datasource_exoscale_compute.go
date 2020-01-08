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
				Description:   "ID of the Compute",
				Optional:      true,
				ConflictsWith: []string{"name", "tag"},
			},
			"name": {
				Type:          schema.TypeString,
				Description:   "Name of the Compute",
				Optional:      true,
				ConflictsWith: []string{"id", "tag"},
			},
			"tags": {
				Type: schema.TypeMap,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
				Description:   "Map of tags (key: value)",
				Optional:      true,
				ConflictsWith: []string{"id", "name"},
			},
			"created": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Date when the compute was created",
			},
			"zone": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the availability zone for the compute",
			},
			"template": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the template for the compute",
			},
			"size": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current size of the compute",
			},
			"disk": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Size of the compute disk",
			},
			"cpu": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Number of cpu the compute is running with",
			},
			"memory": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "memory allocated for the compute",
			},
			"state": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "state of the compute",
			},

			"ipv4": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "compute public ipv4 address",
			},
			"ipv6": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "compute public ipv6 address",
			},
			"privnet_ipv4": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "compute private ipv4 address",
			},
			"privnet_ipv6": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "compute private ipv6 address",
			},
		},

		Read: dataSourceComputeRead,
	}
}

func dataSourceComputeRead(d *schema.ResourceData, meta interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	req := egoscale.ListVirtualMachines{}

	computeName, byName := d.GetOk("name")
	computeID, byID := d.GetOk("id")
	computeTag, byTag := d.GetOk("tag")
	switch {
	case !byName && !byID && !byTag:
		return errors.New("either name, id or tag must be specified")
	case computeID != "":
		var err error
		if req.ID, err = egoscale.ParseUUID(computeID.(string)); err != nil {
			return fmt.Errorf("invalid value for id: %s", err)
		}
	case byTag:
		for key, value := range computeTag.(map[string]string) {
			req.Tags = append(req.Tags, egoscale.ResourceTag{
				Key:   key,
				Value: value,
			})
		}
	default:
		req.Name = computeName.(string)
	}

	resp, err := client.RequestWithContext(ctx, &req)
	if err != nil {
		return fmt.Errorf("compute list query failed: %s", err)
	}

	var c egoscale.VirtualMachine
	nt := resp.(*egoscale.ListVirtualMachinesResponse).Count
	switch {
	case nt == 0:
		return errors.New("compute not found")

	case nt > 1:
		return errors.New("multiple results returned, expected only one")

	default:
		c = resp.(*egoscale.ListVirtualMachinesResponse).VirtualMachine[0]
	}

	resp, err = client.GetWithContext(ctx, &egoscale.Volume{
		VirtualMachineID: c.ID,
		Type:             "ROOT",
	})
	if err != nil {
		return err
	}

	ds := resp.(*egoscale.Volume).Size

	resp, err = client.RequestWithContext(ctx, &egoscale.ListNics{})
	if err != nil {
		return err
	}
	n := resp.(*egoscale.ListNicsResponse).Nic

	return dataSourceComputeApply(d, c, n, ds)
}

func dataSourceComputeApply(d *schema.ResourceData, compute egoscale.VirtualMachine, nics []egoscale.Nic, diskSize uint64) error {
	d.SetId(compute.ID.String())

	if err := d.Set("id", d.Id()); err != nil {
		return err
	}
	if err := d.Set("name", compute.Name); err != nil {
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
	if err := d.Set("disk", diskSize); err != nil {
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
	if err := d.Set("ipv4", compute.IP().String()); err != nil {
		return err
	}

	tags := make(map[string]interface{})
	for _, tag := range compute.Tags {
		tags[tag.Key] = tag.Value
	}
	if err := d.Set("tags", tags); err != nil {
		return err
	}

	privateIPv4 := make([]string, 0)
	privateIPv6 := make([]string, 0)

	for _, nic := range nics {
		switch {
		case nic.IsDefault && nic.IP6Address != nil:
			if err := d.Set("ipv6", nic.IP6Address.String()); err != nil {
				return err
			}
		case !nic.IsDefault:
			if nic.IPAddress != nil {
				privateIPv4 = append(privateIPv4, nic.IPAddress.String())
			}
			if nic.IP6Address != nil {
				privateIPv6 = append(privateIPv6, nic.IP6Address.String())
			}

		}

		if len(privateIPv4) > 0 {
			if err := d.Set("privnet_ipv4", privateIPv4); err != nil {
				return err
			}
		}

		if len(privateIPv6) > 0 {
			if err := d.Set("privnet_ipv6", privateIPv6); err != nil {
				return err
			}
		}
	}

	return nil
}