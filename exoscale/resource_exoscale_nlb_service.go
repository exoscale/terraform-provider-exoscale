package exoscale

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
)

const (
	defaultNLBServiceHealthcheckInterval = 10
	defaultNLBServiceHealthcheckMode     = "tcp"
	defaultNLBServiceHealthcheckRetries  = 1
	defaultNLBServiceHealthcheckTimeout  = 5
	defaultNLBServiceProtocol            = "tcp"
	defaulNLBServiceStrategy             = "round-robin"

	resNLBServiceAttrDescription         = "description"
	resNLBServiceAttrHealthcheck         = "healthcheck"
	resNLBServiceAttrHealthcheckInterval = "interval"
	resNLBServiceAttrHealthcheckMode     = "mode"
	resNLBServiceAttrHealthcheckPort     = "port"
	resNLBServiceAttrHealthcheckRetries  = "retries"
	resNLBServiceAttrHealthcheckTimeout  = "timeout"
	resNLBServiceAttrHealthcheckTLSSNI   = "tls_sni"
	resNLBServiceAttrHealthcheckURI      = "uri"
	resNLBServiceAttrInstancePoolID      = "instance_pool_id"
	resNLBServiceAttrName                = "name"
	resNLBServiceAttrNLBID               = "nlb_id"
	resNLBServiceAttrPort                = "port"
	resNLBServiceAttrProtocol            = "protocol"
	resNLBServiceAttrStrategy            = "strategy"
	resNLBServiceAttrState               = "state"
	resNLBServiceAttrTargetPort          = "target_port"
	resNLBServiceAttrZone                = "zone"
)

func resourceNLBServiceIDString(d general.ResourceIDStringer) string {
	return general.ResourceIDString(d, "exoscale_nlb_service")
}

func resourceNLBService() *schema.Resource {
	s := map[string]*schema.Schema{
		resNLBServiceAttrDescription: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "A free-form text describing the NLB service.",
		},
		resNLBServiceAttrHealthcheck: {
			Description: "The service health checking configuration.",
			Type:        schema.TypeSet,
			Required:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					resNLBServiceAttrHealthcheckInterval: {
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     defaultNLBServiceHealthcheckInterval,
						Description: "The healthcheck interval in seconds (default: `10`).",
					},
					resNLBServiceAttrHealthcheckMode: {
						Type:        schema.TypeString,
						Optional:    true,
						Default:     defaultNLBServiceHealthcheckMode,
						Description: "The healthcheck mode (`tcp`|`http`|`https`; default: `tcp`).",
					},
					resNLBServiceAttrHealthcheckPort: {
						Type:        schema.TypeInt,
						Required:    true,
						Description: "The NLB service (TCP/UDP) port.",
					},
					resNLBServiceAttrHealthcheckRetries: {
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     defaultNLBServiceHealthcheckRetries,
						Description: "The healthcheck retries (default: `1`).",
					},
					resNLBServiceAttrHealthcheckTimeout: {
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     defaultNLBServiceHealthcheckTimeout,
						Description: "The healthcheck timeout (seconds; default: `5`).",
					},
					resNLBServiceAttrHealthcheckTLSSNI: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "The healthcheck TLS SNI server name (only if `mode` is `https`).",
					},
					resNLBServiceAttrHealthcheckURI: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "The healthcheck URI (must be set only if `mode` is `http(s)`).",
					},
				},
			},
		},
		resNLBServiceAttrInstancePoolID: {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "The [exoscale_instance_pool](./instance_pool.md) (ID) to forward traffic to.",
		},
		resNLBServiceAttrName: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The NLB service name.",
		},
		resNLBServiceAttrNLBID: {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "The parent [exoscale_nlb](./nlb.md) ID.",
		},
		resNLBServiceAttrPort: {
			Type:        schema.TypeInt,
			Required:    true,
			Description: "The healthcheck port.",
		},
		resNLBServiceAttrProtocol: {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     defaultNLBServiceProtocol,
			Description: "The protocol (`tcp`|`udp`; default: `tcp`).",
		},
		resNLBServiceAttrState: {
			Type:     schema.TypeString,
			Computed: true,
		},
		resNLBServiceAttrStrategy: {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     defaulNLBServiceStrategy,
			Description: "The strategy (`round-robin`|`source-hash`; default: `round-robin`).",
		},
		resNLBServiceAttrTargetPort: {
			Type:        schema.TypeInt,
			Required:    true,
			Description: "The (TCP/UDP) port to forward traffic to (on target instance pool members).",
		},
		resNLBServiceAttrZone: {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
		},
	}

	return &schema.Resource{
		Schema: s,

		Description: `Manage Exoscale [Network Load Balancer (NLB)](https://community.exoscale.com/product/networking/nlb/) Services.`,

		CreateContext: resourceNLBServiceCreate,
		ReadContext:   resourceNLBServiceRead,
		UpdateContext: resourceNLBServiceUpdate,
		DeleteContext: resourceNLBServiceDelete,

		Importer: &schema.ResourceImporter{
			StateContext: func(ctx context.Context, d *schema.ResourceData, _ interface{}) ([]*schema.ResourceData, error) {
				zonedRes, err := zonedStateContextFunc(ctx, d, nil)
				if err != nil {
					return nil, err
				}
				d = zonedRes[0]

				parts := strings.SplitN(d.Id(), "/", 2)
				if len(parts) != 2 {
					return nil, fmt.Errorf(`invalid ID %q, expected format "<NLB-ID>/<SERVICE-ID>@<ZONE>"`, d.Id())
				}

				d.SetId(parts[1])
				if err := d.Set(resNLBServiceAttrNLBID, parts[0]); err != nil {
					return nil, err
				}

				return []*schema.ResourceData{d}, nil
			},
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(config.DefaultTimeout),
			Read:   schema.DefaultTimeout(config.DefaultTimeout),
			Update: schema.DefaultTimeout(config.DefaultTimeout),
			Delete: schema.DefaultTimeout(config.DefaultTimeout),
		},
	}
}

func resourceNLBServiceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning create", map[string]interface{}{
		"id": resourceNLBServiceIDString(d),
	})

	zone := d.Get(resNLBServiceAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := getClient(meta)

	nlb, err := client.GetNetworkLoadBalancer(ctx, zone, d.Get(resNLBServiceAttrNLBID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	healthcheck := d.Get("healthcheck").(*schema.Set).List()[0].(map[string]interface{})
	nlbServiceHealthcheck := new(egoscale.NetworkLoadBalancerServiceHealthcheck)

	nlbServiceHealthcheckInterval := time.Duration(healthcheck[resNLBServiceAttrHealthcheckInterval].(int)) * time.Second
	nlbServiceHealthcheck.Interval = &nlbServiceHealthcheckInterval

	nlbServiceHealthcheckMode := healthcheck[resNLBServiceAttrHealthcheckMode].(string)
	nlbServiceHealthcheck.Mode = &nlbServiceHealthcheckMode

	nlbServiceHealthcheckPort := uint16(healthcheck[resNLBServiceAttrHealthcheckPort].(int))
	nlbServiceHealthcheck.Port = &nlbServiceHealthcheckPort

	nlbServiceHealthcheckRetries := int64(healthcheck[resNLBServiceAttrHealthcheckRetries].(int))
	nlbServiceHealthcheck.Retries = &nlbServiceHealthcheckRetries

	nlbServiceHealthcheckTimeout := time.Duration(healthcheck[resNLBServiceAttrHealthcheckTimeout].(int)) * time.Second
	nlbServiceHealthcheck.Timeout = &nlbServiceHealthcheckTimeout

	if strings.HasPrefix(nlbServiceHealthcheckMode, "http") {
		if v, ok := healthcheck[resNLBServiceAttrHealthcheckTLSSNI]; ok && v.(string) != "" {
			s := v.(string)
			nlbServiceHealthcheck.TLSSNI = &s
		}

		if v, ok := healthcheck[resNLBServiceAttrHealthcheckURI]; ok {
			s := v.(string)
			nlbServiceHealthcheck.URI = &s
		}
	}

	nlbService := new(egoscale.NetworkLoadBalancerService)

	nlbServiceName := d.Get(resNLBServiceAttrName).(string)
	nlbService.Name = &nlbServiceName

	if v, ok := d.GetOk(resNLBServiceAttrDescription); ok {
		s := v.(string)
		nlbService.Description = &s
	}

	nlbService.Healthcheck = nlbServiceHealthcheck

	nlbServiceInstancePoolID := d.Get(resNLBServiceAttrInstancePoolID).(string)
	nlbService.InstancePoolID = &nlbServiceInstancePoolID

	nlbServicePort := uint16(d.Get(resNLBServiceAttrPort).(int))
	nlbService.Port = &nlbServicePort

	nlbServiceProtocol := d.Get(resNLBServiceAttrProtocol).(string)
	nlbService.Protocol = &nlbServiceProtocol

	nlbServiceStrategy := d.Get(resNLBServiceAttrStrategy).(string)
	nlbService.Strategy = &nlbServiceStrategy

	nlbServiceTargetPort := uint16(d.Get(resNLBServiceAttrTargetPort).(int))
	nlbService.TargetPort = &nlbServiceTargetPort

	nlbService, err = client.CreateNetworkLoadBalancerService(ctx, zone, nlb, nlbService)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*nlbService.ID)

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": resourceNLBServiceIDString(d),
	})

	return resourceNLBServiceRead(ctx, d, meta)
}

func resourceNLBServiceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceNLBServiceIDString(d),
	})

	zone := d.Get(resNLBServiceAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := getClient(meta)

	nlb, err := client.GetNetworkLoadBalancer(ctx, zone, d.Get(resNLBServiceAttrNLBID).(string))
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Parent NLB doesn't exist anymore, so does the NLB service.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	var nlbService *egoscale.NetworkLoadBalancerService
	for _, s := range nlb.Services {
		if *s.ID == d.Id() {
			nlbService = s
			break
		}
	}
	if nlbService == nil {
		// Resource doesn't exist anymore, signaling the core to remove it from the state.
		d.SetId("")
		return nil
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceNLBServiceIDString(d),
	})

	return diag.FromErr(resourceNLBServiceApply(ctx, d, nlbService))
}

func resourceNLBServiceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning update", map[string]interface{}{
		"id": resourceNLBServiceIDString(d),
	})

	zone := d.Get(resNLBServiceAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutUpdate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := getClient(meta)

	nlb, err := client.GetNetworkLoadBalancer(ctx, zone, d.Get(resNLBServiceAttrNLBID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	var nlbService *egoscale.NetworkLoadBalancerService
	for _, s := range nlb.Services {
		if *s.ID == d.Id() {
			nlbService = s
			break
		}
	}
	if nlbService == nil {
		return diag.Errorf("Network Load Balancer Service %q not found", d.Id())
	}

	var updated bool

	if d.HasChange(resNLBServiceAttrName) {
		v := d.Get(resNLBServiceAttrName).(string)
		nlbService.Name = &v
		updated = true
	}

	if d.HasChange(resNLBServiceAttrDescription) {
		v := d.Get(resNLBServiceAttrDescription).(string)
		nlbService.Description = &v
		updated = true
	}

	if d.HasChange(resNLBServiceAttrPort) {
		v := uint16(d.Get(resNLBServiceAttrPort).(int))
		nlbService.Port = &v
		updated = true
	}

	if d.HasChange(resNLBServiceAttrProtocol) {
		v := d.Get(resNLBServiceAttrProtocol).(string)
		nlbService.Protocol = &v
		updated = true
	}

	if d.HasChange(resNLBServiceAttrStrategy) {
		v := d.Get(resNLBServiceAttrStrategy).(string)
		nlbService.Strategy = &v
		updated = true
	}

	if d.HasChange(resNLBServiceAttrTargetPort) {
		v := uint16(d.Get(resNLBServiceAttrTargetPort).(int))
		nlbService.TargetPort = &v
		updated = true
	}

	if d.HasChange("healthcheck") {
		healthcheck := d.Get("healthcheck").(*schema.Set).List()[0].(map[string]interface{})

		nlbServiceHealthcheckInterval := time.Duration(healthcheck[resNLBServiceAttrHealthcheckInterval].(int)) * time.Second
		nlbService.Healthcheck.Interval = &nlbServiceHealthcheckInterval

		nlbServiceHealthcheckMode := healthcheck[resNLBServiceAttrHealthcheckMode].(string)
		nlbService.Healthcheck.Mode = &nlbServiceHealthcheckMode

		nlbServiceHealthcheckPort := uint16(healthcheck[resNLBServiceAttrHealthcheckPort].(int))
		nlbService.Healthcheck.Port = &nlbServiceHealthcheckPort

		nlbServiceHealthcheckRetries := int64(healthcheck[resNLBServiceAttrHealthcheckRetries].(int))
		nlbService.Healthcheck.Retries = &nlbServiceHealthcheckRetries

		nlbServiceHealthcheckTimeout := time.Duration(healthcheck[resNLBServiceAttrHealthcheckTimeout].(int)) * time.Second
		nlbService.Healthcheck.Timeout = &nlbServiceHealthcheckTimeout

		if strings.HasPrefix(nlbServiceHealthcheckMode, "http") {
			if v, ok := healthcheck[resNLBServiceAttrHealthcheckTLSSNI]; ok && v.(string) != "" {
				s := v.(string)
				nlbService.Healthcheck.TLSSNI = &s
			}

			if v, ok := healthcheck[resNLBServiceAttrHealthcheckURI]; ok {
				s := v.(string)
				nlbService.Healthcheck.URI = &s
			}
			// We need a need struct to remove URI and TLSSNI
		} else {
			*nlbService.Healthcheck = egoscale.NetworkLoadBalancerServiceHealthcheck{
				Interval: &nlbServiceHealthcheckInterval,
				Mode:     &nlbServiceHealthcheckMode,
				Port:     &nlbServiceHealthcheckPort,
				Retries:  &nlbServiceHealthcheckRetries,
				Timeout:  &nlbServiceHealthcheckTimeout,
			}
		}

		updated = true
	}

	if updated {
		nlb, err := client.GetNetworkLoadBalancer(ctx, zone, d.Get(resNLBServiceAttrNLBID).(string))
		if err != nil {
			return diag.FromErr(err)
		}

		if err = client.UpdateNetworkLoadBalancerService(ctx, zone, nlb, nlbService); err != nil {
			return diag.FromErr(err)
		}
	}

	tflog.Debug(ctx, "update finished successfully", map[string]interface{}{
		"id": resourceNLBServiceIDString(d),
	})

	return resourceNLBServiceRead(ctx, d, meta)
}

func resourceNLBServiceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning delete", map[string]interface{}{
		"id": resourceNLBServiceIDString(d),
	})

	zone := d.Get(resNLBServiceAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := getClient(meta)

	nlb, err := client.GetNetworkLoadBalancer(ctx, zone, d.Get(resNLBServiceAttrNLBID).(string))
	if err != nil {
		return diag.FromErr(err)
	}

	nlbServiceID := d.Id()
	err = client.DeleteNetworkLoadBalancerService(ctx, zone, nlb, &egoscale.NetworkLoadBalancerService{ID: &nlbServiceID})
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": resourceNLBServiceIDString(d),
	})

	return nil
}

func resourceNLBServiceApply(
	_ context.Context,
	d *schema.ResourceData,
	nlbService *egoscale.NetworkLoadBalancerService,
) error {
	if err := d.Set(resNLBServiceAttrDescription, defaultString(nlbService.Description, "")); err != nil {
		return err
	}

	healthcheck := d.Get(resNLBServiceAttrHealthcheck).(*schema.Set)
	if err := d.Set(resNLBServiceAttrHealthcheck, schema.NewSet(healthcheck.F, []interface{}{
		map[string]interface{}{
			resNLBServiceAttrHealthcheckInterval: int(nlbService.Healthcheck.Interval.Seconds()),
			resNLBServiceAttrHealthcheckMode:     *nlbService.Healthcheck.Mode,
			resNLBServiceAttrHealthcheckPort:     int(*nlbService.Healthcheck.Port),
			resNLBServiceAttrHealthcheckRetries:  int(defaultInt64(nlbService.Healthcheck.Retries, 0)),
			resNLBServiceAttrHealthcheckTLSSNI:   defaultString(nlbService.Healthcheck.TLSSNI, ""),
			resNLBServiceAttrHealthcheckTimeout:  int(nlbService.Healthcheck.Timeout.Seconds()),
			resNLBServiceAttrHealthcheckURI:      defaultString(nlbService.Healthcheck.URI, ""),
		},
	})); err != nil {
		return err
	}

	if err := d.Set(resNLBServiceAttrInstancePoolID, *nlbService.InstancePoolID); err != nil {
		return err
	}

	if err := d.Set(resNLBServiceAttrName, *nlbService.Name); err != nil {
		return err
	}

	if err := d.Set(resNLBServiceAttrPort, *nlbService.Port); err != nil {
		return err
	}

	if err := d.Set(resNLBServiceAttrProtocol, *nlbService.Protocol); err != nil {
		return err
	}

	if err := d.Set(resNLBServiceAttrState, *nlbService.State); err != nil {
		return err
	}

	if err := d.Set(resNLBServiceAttrStrategy, *nlbService.Strategy); err != nil {
		return err
	}

	if err := d.Set(resNLBServiceAttrTargetPort, *nlbService.TargetPort); err != nil {
		return err
	}

	return nil
}
