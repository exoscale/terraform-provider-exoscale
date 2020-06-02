package exoscale

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/exoscale/egoscale"
	apiv2 "github.com/exoscale/egoscale/api/v2"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

func resourceNLBServiceIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_nlb_service")
}

func resourceNLBService() *schema.Resource {
	s := map[string]*schema.Schema{
		"zone": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"nlb_id": {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
		"name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"instance_pool_id": {
			Type:     schema.TypeString,
			Required: true,
		},
		"port": {
			Type:     schema.TypeInt,
			Required: true,
		},
		"target_port": {
			Type:     schema.TypeInt,
			Required: true,
		},
		"healthcheck": {
			Type:     schema.TypeSet,
			Required: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"port": {
						Type:     schema.TypeInt,
						Required: true,
					},
					"mode": {
						Type:     schema.TypeString,
						Optional: true,
						Default:  "tcp",
					},
					"interval": {
						Type:     schema.TypeInt,
						Optional: true,
						Default:  10,
					},
					"timeout": {
						Type:     schema.TypeInt,
						Optional: true,
						Default:  5,
					},
					"retries": {
						Type:     schema.TypeInt,
						Optional: true,
						Default:  1,
					},
					"uri": {
						Type:     schema.TypeString,
						Optional: true,
					},
				},
			},
		},
		"protocol": {
			Type:     schema.TypeString,
			Optional: true,
			Default:  "tcp",
		},
		"strategy": {
			Type:     schema.TypeString,
			Optional: true,
			Default:  "round-robin",
		},
		"description": {
			Type:     schema.TypeString,
			Optional: true,
		},
	}

	return &schema.Resource{
		Schema: s,

		Create: resourceNLBServiceCreate,
		Read:   resourceNLBServiceRead,
		Update: resourceNLBServiceUpdate,
		Delete: resourceNLBServiceDelete,
		Exists: resourceNLBServiceExists,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceNLBServiceCreate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning create", resourceNLBServiceIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutCreate))
	defer cancel()

	client := GetComputeClient(meta)

	zone := d.Get("zone").(string)

	ctx = apiv2.WithEndpoint(ctx, apiv2.NewReqEndpoint(getEnvironment(meta), zone))
	nlb, err := client.GetNetworkLoadBalancer(
		ctx,
		zone,
		d.Get("nlb_id").(string),
	)
	if err != nil {
		return err
	}

	raw := d.Get("healthcheck").(*schema.Set)
	healthcheck := raw.List()[0].(map[string]interface{})

	service, err := nlb.AddService(
		ctx,
		&egoscale.NetworkLoadBalancerService{
			Name:           d.Get("name").(string),
			Description:    d.Get("description").(string),
			InstancePoolID: d.Get("instance_pool_id").(string),
			Protocol:       d.Get("protocol").(string),
			Port:           uint16(d.Get("port").(int)),
			TargetPort:     uint16(d.Get("target_port").(int)),
			Strategy:       d.Get("strategy").(string),
			Healthcheck: egoscale.NetworkLoadBalancerServiceHealthcheck{
				Mode:     healthcheck["mode"].(string),
				Port:     uint16(healthcheck["port"].(int)),
				Interval: time.Duration(healthcheck["interval"].(int)) * time.Second,
				Timeout:  time.Duration(healthcheck["timeout"].(int)) * time.Second,
				Retries:  int64(healthcheck["retries"].(int)),
				URI:      healthcheck["uri"].(string),
			},
		},
	)
	if err != nil {
		return err
	}

	d.SetId(service.ID)

	log.Printf("[DEBUG] %s: create finished successfully", resourceNLBServiceIDString(d))

	return resourceNLBServiceRead(d, meta)
}

func resourceNLBServiceRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning read", resourceNLBServiceIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	service, err := findNLBService(ctx, d, meta)
	if err != nil {
		return err
	}

	if service == nil {
		return fmt.Errorf("Network Load Balancer Service %q not found", d.Id())
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceNLBServiceIDString(d))

	return resourceNLBServiceApply(d, service)
}

func resourceNLBServiceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutRead))
	defer cancel()

	service, err := findNLBService(ctx, d, meta)
	if err != nil {
		return false, err
	}

	if service == nil {
		return false, nil
	}

	return true, nil
}

func resourceNLBServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning update", resourceNLBServiceIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutUpdate))
	defer cancel()

	client := GetComputeClient(meta)

	service, err := findNLBService(ctx, d, meta)
	if err != nil {
		return err
	}

	if service == nil {
		return fmt.Errorf("Network Load Balancer Service %q not found", d.Id())
	}

	if d.HasChange("name") {
		service.Name = d.Get("name").(string)
	}

	if d.HasChange("description") {
		service.Description = d.Get("description").(string)
	}

	if d.HasChange("protocol") {
		service.Protocol = d.Get("protocol").(string)
	}

	if d.HasChange("strategy") {
		service.Strategy = d.Get("strategy").(string)
	}

	if d.HasChange("port") {
		service.Port = uint16(d.Get("port").(int))
	}

	if d.HasChange("target_port") {
		service.TargetPort = uint16(d.Get("target_port").(int))
	}

	if d.HasChange("healthcheck") {
		raw := d.Get("healthcheck").(*schema.Set)
		healthcheck := raw.List()[0].(map[string]interface{})

		service.Healthcheck.Mode = healthcheck["mode"].(string)
		service.Healthcheck.Port = uint16(healthcheck["port"].(int))
		service.Healthcheck.Retries = int64(healthcheck["retries"].(int))
		service.Healthcheck.URI = healthcheck["uri"].(string)
		service.Healthcheck.Interval = time.Duration(healthcheck["interval"].(int)) * time.Second
		service.Healthcheck.Timeout = time.Duration(healthcheck["timeout"].(int)) * time.Second
	}

	zone := d.Get("zone").(string)

	ctx = apiv2.WithEndpoint(ctx, apiv2.NewReqEndpoint(getEnvironment(meta), zone))
	nlb, err := client.GetNetworkLoadBalancer(
		ctx,
		zone,
		d.Get("nlb_id").(string),
	)
	if err != nil {
		return err
	}

	err = nlb.UpdateService(
		ctx,
		service,
	)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: update finished successfully", resourceNLBServiceIDString(d))

	return resourceNLBServiceRead(d, meta)
}

func resourceNLBServiceDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[DEBUG] %s: beginning delete", resourceNLBServiceIDString(d))

	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout(schema.TimeoutDelete))
	defer cancel()

	client := GetComputeClient(meta)

	zone := d.Get("zone").(string)

	ctx = apiv2.WithEndpoint(ctx, apiv2.NewReqEndpoint(getEnvironment(meta), zone))
	nlb, err := client.GetNetworkLoadBalancer(
		ctx,
		zone,
		d.Get("nlb_id").(string),
	)
	if err != nil {
		return err
	}

	err = nlb.DeleteService(
		ctx,
		&egoscale.NetworkLoadBalancerService{
			ID: d.Id(),
		},
	)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceNLBServiceIDString(d))

	return nil
}

func resourceNLBServiceApply(d *schema.ResourceData, service *egoscale.NetworkLoadBalancerService) error {
	if err := d.Set("name", service.Name); err != nil {
		return err
	}

	if err := d.Set("description", service.Description); err != nil {
		return err
	}

	if err := d.Set("instance_pool_id", service.InstancePoolID); err != nil {
		return err
	}

	if err := d.Set("protocol", service.Protocol); err != nil {
		return err
	}

	if err := d.Set("port", service.Port); err != nil {
		return err
	}

	if err := d.Set("target_port", service.TargetPort); err != nil {
		return err
	}

	if err := d.Set("strategy", service.Strategy); err != nil {
		return err
	}

	healthcheck := map[string]interface{}{
		"mode":     service.Healthcheck.Mode,
		"port":     int(service.Healthcheck.Port),
		"interval": int(service.Healthcheck.Interval.Seconds()),
		"timeout":  int(service.Healthcheck.Timeout.Seconds()),
		"retries":  int(service.Healthcheck.Retries),
		"uri":      service.Healthcheck.URI,
	}

	raw := d.Get("healthcheck").(*schema.Set)
	set := schema.NewSet(raw.F, []interface{}{healthcheck})

	return d.Set("healthcheck", set)
}

func findNLBService(ctx context.Context, d *schema.ResourceData, meta interface{}) (*egoscale.NetworkLoadBalancerService, error) {
	client := GetComputeClient(meta)

	zone, okZone := d.GetOk("zone")
	nlbID, okNLBID := d.GetOk("nlb_id")
	if okZone && okNLBID {
		ctx = apiv2.WithEndpoint(ctx, apiv2.NewReqEndpoint(getEnvironment(meta), zone.(string)))
		nlb, err := client.GetNetworkLoadBalancer(ctx, zone.(string), nlbID.(string))
		if err != nil {
			return nil, err
		}
		for _, s := range nlb.Services {
			if s.ID == d.Id() {
				return s, nil
			}
		}
	}

	resp, err := client.RequestWithContext(ctx, egoscale.ListZones{})
	if err != nil {
		return nil, err
	}
	zones := resp.(*egoscale.ListZonesResponse).Zone

	for _, zone := range zones {
		nlbs, err := client.ListNetworkLoadBalancers(
			apiv2.WithEndpoint(ctx, apiv2.NewReqEndpoint(getEnvironment(meta), zone.Name)),
			zone.Name)
		if err != nil {
			return nil, err
		}

		for _, nlb := range nlbs {
			for _, s := range nlb.Services {
				if s.ID == d.Id() {
					if err := d.Set("zone", zone.Name); err != nil {
						return nil, err
					}
					if err := d.Set("nlb_id", nlb.ID); err != nil {
						return nil, err
					}

					return s, nil
				}
			}
		}
	}

	return nil, nil
}
