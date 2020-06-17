package exoscale

import (
	"context"
	"fmt"
	"log"

	apiv2 "github.com/exoscale/egoscale/api/v2"

	"github.com/exoscale/egoscale"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceNLBIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_nlb")
}

func resourceNLB() *schema.Resource {
	s := map[string]*schema.Schema{
		"zone": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"description": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"created_at": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"state": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"ip_address": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"services": {
			Type:     schema.TypeSet,
			Computed: true,
			Set:      schema.HashString,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
	}

	return &schema.Resource{
		Schema: s,

		Create: resourceNLBCreate,
		Read:   resourceNLBRead,
		Update: resourceNLBUpdate,
		Delete: resourceNLBDelete,
		Exists: resourceNLBExists,

		Importer: &schema.ResourceImporter{
			State: resourceNLBImport,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceNLBCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning create", resourceNLBIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	zone := d.Get("zone").(string)

	ctx = apiv2.WithEndpoint(ctx, apiv2.NewReqEndpoint(getEnvironment(meta), zone))
	nlb, err := client.CreateNetworkLoadBalancer(
		ctx,
		zone,
		&egoscale.NetworkLoadBalancer{
			Name:        d.Get("name").(string),
			Description: d.Get("description").(string),
		},
	)
	if err != nil {
		return err
	}

	d.SetId(nlb.ID)

	log.Printf("[DEBUG] %s: create finished successfully", resourceNLBIDString(d))

	return resourceNLBRead(d, meta)
}

func resourceNLBRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning read", resourceNLBIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	nlb, err := findNLB(ctx, d, meta)
	if err != nil {
		return err
	}

	if nlb == nil {
		return fmt.Errorf("Network Load Balancer %q not found", d.Id())
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceNLBIDString(d))

	return resourceNLBApply(d, nlb)
}

func resourceNLBExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	nlb, err := findNLB(ctx, d, meta)
	if err != nil {
		return false, err
	}

	if nlb == nil {
		return false, nil
	}

	return true, nil
}

func resourceNLBUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning update", resourceNLBIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	client := GetComputeClient(meta)

	nlb, err := findNLB(ctx, d, meta)
	if err != nil {
		return err
	}

	if nlb == nil {
		return fmt.Errorf("Network Load Balancer %q not found", d.Id())
	}

	if d.HasChange("name") {
		nlb.Name = d.Get("name").(string)
	}

	if d.HasChange("description") {
		nlb.Description = d.Get("description").(string)
	}

	zone := d.Get("zone").(string)

	ctx = apiv2.WithEndpoint(ctx, apiv2.NewReqEndpoint(getEnvironment(meta), zone))
	_, err = client.UpdateNetworkLoadBalancer(ctx, zone, nlb)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: update finished successfully", resourceNLBIDString(d))

	return resourceNLBRead(d, meta)
}

func resourceNLBDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning delete", resourceNLBIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	zone := d.Get("zone").(string)

	ctx = apiv2.WithEndpoint(ctx, apiv2.NewReqEndpoint(getEnvironment(meta), zone))
	err := client.DeleteNetworkLoadBalancer(ctx, zone, d.Id())
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceNLBIDString(d))

	return nil
}

func resourceNLBImport(d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	log.Printf("[DEBUG] %s: beginning import", resourceNLBIDString(d))
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	nlb, err := findNLB(ctx, d, meta)
	if err != nil {
		return nil, err
	}

	if nlb == nil {
		return nil, fmt.Errorf("Network Load Balancer %q not found", d.Id())
	}

	if err := resourceNLBApply(d, nlb); err != nil {
		return nil, err
	}

	resources := []*schema.ResourceData{d}

	for _, service := range nlb.Services {
		resource := resourceNLBService()
		d := resource.Data(nil)
		d.SetType("exoscale_nlb_service")
		d.SetId(service.ID)
		err := resourceNLBServiceApply(d, service)
		if err != nil {
			return nil, err
		}

		resources = append(resources, d)
	}

	log.Printf("[DEBUG] %s: import finished successfully", resourceNLBIDString(d))

	return resources, nil
}

func resourceNLBApply(d *schema.ResourceData, nlb *egoscale.NetworkLoadBalancer) error {
	if err := d.Set("name", nlb.Name); err != nil {
		return err
	}

	if err := d.Set("description", nlb.Description); err != nil {
		return err
	}

	if err := d.Set("created_at", nlb.CreatedAt.String()); err != nil {
		return err
	}

	if err := d.Set("state", nlb.State); err != nil {
		return err
	}

	if err := d.Set("ip_address", nlb.IPAddress.String()); err != nil {
		return err
	}

	services := make([]string, len(nlb.Services))
	for i, service := range nlb.Services {
		services[i] = service.ID
	}

	if err := d.Set("services", services); err != nil {
		return err
	}

	return nil
}

func findNLB(ctx context.Context, d *schema.ResourceData, meta interface{}) (*egoscale.NetworkLoadBalancer, error) {
	client := GetComputeClient(meta)

	if zone, ok := d.GetOk("zone"); ok {
		ctx = apiv2.WithEndpoint(ctx, apiv2.NewReqEndpoint(getEnvironment(meta), zone.(string)))
		nlb, err := client.GetNetworkLoadBalancer(ctx, zone.(string), d.Id())
		if err != nil {
			return nil, err
		}

		return nlb, nil
	}

	resp, err := client.RequestWithContext(ctx, egoscale.ListZones{})
	if err != nil {
		return nil, err
	}
	zones := resp.(*egoscale.ListZonesResponse).Zone

	var nlb *egoscale.NetworkLoadBalancer
	for _, zone := range zones {
		n, err := client.GetNetworkLoadBalancer(
			apiv2.WithEndpoint(ctx, apiv2.NewReqEndpoint(getEnvironment(meta), zone.Name)),
			zone.Name,
			d.Id())
		if err != nil {
			if err == egoscale.ErrNotFound {
				continue
			}

			return nil, err
		}

		nlb = n
		if err := d.Set("zone", zone.Name); err != nil {
			return nil, err
		}
	}

	return nlb, nil
}
