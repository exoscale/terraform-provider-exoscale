package exoscale

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"

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

func resourceNLBIDString(d general.ResourceIDStringer) string {
	return general.ResourceIDString(d, "exoscale_nlb")
}

func resourceNLB() *schema.Resource {
	s := map[string]*schema.Schema{
		resNLBAttrCreatedAt: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The NLB creation date.",
		},
		resNLBAttrDescription: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "A free-form text describing the NLB.",
		},
		resNLBAttrIPAddress: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The NLB IPv4 address.",
		},
		resNLBAttrLabels: {
			Type:        schema.TypeMap,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Description: "A map of key/value labels.",
		},
		resNLBAttrName: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The network load balancer (NLB) name.",
		},
		resNLBAttrServices: {
			Type:        schema.TypeSet,
			Computed:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "The list of the [exoscale_nlb_service](./nlb_service.md) (names).",
		},
		resNLBAttrState: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The current NLB state.",
		},
		resNLBAttrZone: {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
		},
	}

	return &schema.Resource{
		Schema:      s,
		Description: "Manage Exoscale Network Load Balancers (NLB).",

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
	tflog.Debug(ctx, "beginning create", map[string]interface{}{
		"id": resourceNLBIDString(d),
	})

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

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": resourceNLBIDString(d),
	})

	return resourceNLBRead(ctx, d, meta)
}

func resourceNLBRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceNLBIDString(d),
	})

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

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceNLBIDString(d),
	})

	return diag.FromErr(resourceNLBApply(ctx, d, nlb))
}

func resourceNLBUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning update", map[string]interface{}{
		"id": resourceNLBIDString(d),
	})

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

	tflog.Debug(ctx, "update finished successfully", map[string]interface{}{
		"id": resourceNLBIDString(d),
	})

	return resourceNLBRead(ctx, d, meta)
}

func resourceNLBDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning delete", map[string]interface{}{
		"id": resourceNLBIDString(d),
	})

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

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": resourceNLBIDString(d),
	})

	return nil
}

func resourceNLBApply(_ context.Context, d *schema.ResourceData, nlb *egoscale.NetworkLoadBalancer) error {
	if err := d.Set(resNLBAttrCreatedAt, nlb.CreatedAt.String()); err != nil {
		return err
	}

	if err := d.Set(resNLBAttrDescription, defaultString(nlb.Description, "")); err != nil {
		return err
	}

	if err := d.Set(resNLBAttrIPAddress, nlb.IPAddress.String()); err != nil {
		return err
	}

	if err := d.Set(resNLBAttrLabels, nlb.Labels); err != nil {
		return err
	}

	if err := d.Set(resNLBAttrName, *nlb.Name); err != nil {
		return err
	}

	services := make([]string, len(nlb.Services))
	for i, service := range nlb.Services {
		services[i] = *service.ID
	}
	if err := d.Set(resNLBAttrServices, services); err != nil {
		return err
	}

	if err := d.Set(resNLBAttrState, *nlb.State); err != nil {
		return err
	}

	return nil
}
