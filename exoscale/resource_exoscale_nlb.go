package exoscale

import (
	"context"
	"errors"
	"log"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	resNLBAttrCreatedAt   = "created_at"
	resNLBAttrDescription = "description"
	resNLBAttrIPAddress   = "ip_address"
	resNLBAttrLabels      = "labels"
	resNLBAttrName        = "name"
	resNLBAttrServices    = "services"
	resNLBAttrState       = "state"
	resNLBAttrZone        = "zone"
)

func resourceNLBIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_nlb")
}

func resourceNLB() *schema.Resource {
	s := map[string]*schema.Schema{
		resNLBAttrCreatedAt: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resNLBAttrDescription: {
			Type:     schema.TypeString,
			Optional: true,
		},
		resNLBAttrIPAddress: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resNLBAttrLabels: {
			Type:     schema.TypeMap,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Optional: true,
		},
		resNLBAttrName: {
			Type:     schema.TypeString,
			Required: true,
		},
		resNLBAttrServices: {
			Type:     schema.TypeSet,
			Computed: true,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		resNLBAttrState: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resNLBAttrZone: {
			Type:     schema.TypeString,
			Required: true,
			ForceNew: true,
		},
	}

	return &schema.Resource{
		Schema: s,

		CreateContext: resourceNLBCreate,
		ReadContext:   resourceNLBRead,
		UpdateContext: resourceNLBUpdate,
		DeleteContext: resourceNLBDelete,

		Importer: &schema.ResourceImporter{
			StateContext: zonedStateContextFunc,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(defaultTimeout),
			Read:   schema.DefaultTimeout(defaultTimeout),
			Update: schema.DefaultTimeout(defaultTimeout),
			Delete: schema.DefaultTimeout(defaultTimeout),
		},
	}
}

func resourceNLBCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning create", resourceNLBIDString(d))

	zone := d.Get(resNLBAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	nlb := new(egoscale.NetworkLoadBalancer)

	if l, ok := d.GetOk(resNLBAttrLabels); ok {
		labels := make(map[string]string)
		for k, v := range l.(map[string]interface{}) {
			labels[k] = v.(string)
		}
		nlb.Labels = &labels
	}

	nlbName := d.Get(resNLBAttrName).(string)
	nlb.Name = &nlbName

	if v, ok := d.GetOk(resNLBAttrDescription); ok {
		s := v.(string)
		nlb.Description = &s
	}

	nlb, err := client.CreateNetworkLoadBalancer(ctx, zone, nlb)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*nlb.ID)

	log.Printf("[DEBUG] %s: create finished successfully", resourceNLBIDString(d))

	return resourceNLBRead(ctx, d, meta)
}

func resourceNLBRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning read", resourceNLBIDString(d))

	zone := d.Get(resNLBAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	nlb, err := client.GetNetworkLoadBalancer(ctx, zone, d.Id())
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceNLBIDString(d))

	return resourceNLBApply(ctx, d, nlb)
}

func resourceNLBUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning update", resourceNLBIDString(d))

	zone := d.Get(resNLBAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutUpdate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	nlb, err := client.GetNetworkLoadBalancer(ctx, zone, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var updated bool

	if d.HasChange(resNLBAttrLabels) {
		labels := make(map[string]string)
		for k, v := range d.Get(resNLBAttrLabels).(map[string]interface{}) {
			labels[k] = v.(string)
		}
		nlb.Labels = &labels
		updated = true
	}

	if d.HasChange(resNLBAttrName) {
		v := d.Get(resNLBAttrName).(string)
		nlb.Name = &v
		updated = true
	}

	if d.HasChange(resNLBAttrDescription) {
		v := d.Get(resNLBAttrDescription).(string)
		nlb.Description = &v
		updated = true
	}

	if updated {
		if err = client.UpdateNetworkLoadBalancer(ctx, zone, nlb); err != nil {
			return diag.FromErr(err)
		}
	}

	log.Printf("[DEBUG] %s: update finished successfully", resourceNLBIDString(d))

	return resourceNLBRead(ctx, d, meta)
}

func resourceNLBDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning delete", resourceNLBIDString(d))

	zone := d.Get(resNLBAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	nlbID := d.Id()
	err := client.DeleteNetworkLoadBalancer(ctx, zone, &egoscale.NetworkLoadBalancer{ID: &nlbID})
	if err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] %s: delete finished successfully", resourceNLBIDString(d))

	return nil
}

func resourceNLBApply(_ context.Context, d *schema.ResourceData, nlb *egoscale.NetworkLoadBalancer) diag.Diagnostics {
	if err := d.Set(resNLBAttrCreatedAt, nlb.CreatedAt.String()); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resNLBAttrDescription, defaultString(nlb.Description, "")); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resNLBAttrIPAddress, nlb.IPAddress.String()); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resNLBAttrLabels, nlb.Labels); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resNLBAttrName, *nlb.Name); err != nil {
		return diag.FromErr(err)
	}

	services := make([]string, len(nlb.Services))
	for i, service := range nlb.Services {
		services[i] = *service.ID
	}
	if err := d.Set(resNLBAttrServices, services); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(resNLBAttrState, *nlb.State); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
