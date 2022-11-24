package exoscale

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	resElasticIPAttrAddressFamily            = "address_family"
	resElasticIPAttrCIDR                     = "cidr"
	resElasticIPAttrDescription              = "description"
	resElasticIPAttrHealthcheckInterval      = "interval"
	resElasticIPAttrHealthcheckMode          = "mode"
	resElasticIPAttrHealthcheckPort          = "port"
	resElasticIPAttrHealthcheckStrikesFail   = "strikes_fail"
	resElasticIPAttrHealthcheckStrikesOK     = "strikes_ok"
	resElasticIPAttrHealthcheckTLSSNI        = "tls_sni"
	resElasticIPAttrHealthcheckTLSSkipVerify = "tls_skip_verify"
	resElasticIPAttrHealthcheckTimeout       = "timeout"
	resElasticIPAttrHealthcheckURI           = "uri"
	resElasticIPAttrIPAddress                = "ip_address"
	resElasticIPAttrLabels                   = "labels"
	resElasticIPAttrZone                     = "zone"
)

func resourceElasticIPIDString(d resourceIDStringer) string {
	return resourceIDString(d, "exoscale_elastic_ip")
}

func resourceElasticIP() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			resElasticIPAttrAddressFamily: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},
			resElasticIPAttrCIDR: {
				Type:     schema.TypeString,
				Computed: true,
			},
			resElasticIPAttrDescription: {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},
			"healthcheck": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						resElasticIPAttrHealthcheckInterval: {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(5, 300),
							Default:      10,
						},
						resElasticIPAttrHealthcheckMode: {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringMatch(
								regexp.MustCompile("(?:tcp|https?)"),
								`must be either "tcp", "http", or "https"`,
							),
						},
						resElasticIPAttrHealthcheckPort: {
							Type:         schema.TypeInt,
							Required:     true,
							ValidateFunc: validation.IntBetween(1, 65535),
						},
						resElasticIPAttrHealthcheckTimeout: {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(2, 60),
							Default:      3,
						},
						resElasticIPAttrHealthcheckStrikesFail: {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(1, 20),
							Default:      2,
						},
						resElasticIPAttrHealthcheckStrikesOK: {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(1, 20),
							Default:      3,
						},
						resElasticIPAttrHealthcheckTLSSkipVerify: {
							Type:     schema.TypeBool,
							Optional: true,
						},
						resElasticIPAttrHealthcheckTLSSNI: {
							Type:     schema.TypeString,
							Optional: true,
						},
						resElasticIPAttrHealthcheckURI: {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			resElasticIPAttrIPAddress: {
				Type:     schema.TypeString,
				Computed: true,
			},
			resElasticIPAttrLabels: {
				Type:     schema.TypeMap,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			resElasticIPAttrZone: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},

		CreateContext: resourceElasticIPCreate,
		ReadContext:   resourceElasticIPRead,
		UpdateContext: resourceElasticIPUpdate,
		DeleteContext: resourceElasticIPDelete,

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

func resourceElasticIPCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning create", map[string]interface{}{
		"id": resourceElasticIPIDString(d),
	})

	zone := d.Get(resElasticIPAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	elasticIP := new(egoscale.ElasticIP)

	if v, ok := d.GetOk(resElasticIPAttrAddressFamily); ok {
		s := v.(string)
		if s != "" {
			elasticIP.AddressFamily = &s
		}
	}

	if v, ok := d.GetOk(resElasticIPAttrDescription); ok {
		s := v.(string)
		elasticIP.Description = &s
	}

	if healthcheckMode, ok := d.GetOk(resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckMode)); ok {
		elasticIPHealthcheck := egoscale.ElasticIPHealthcheck{
			Mode: nonEmptyStringPtr(healthcheckMode.(string)),
			Port: func() *uint16 {
				p := uint16(d.Get(resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckPort)).(int))
				return &p
			}(),
		}

		if v, ok := d.GetOk(resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckInterval)); ok {
			interval := time.Duration(v.(int)) * time.Second
			elasticIPHealthcheck.Interval = &interval
		}

		if v, ok := d.GetOk(resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckStrikesFail)); ok {
			strikesFail := int64(v.(int))
			elasticIPHealthcheck.StrikesFail = &strikesFail
		}

		if v, ok := d.GetOk(resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckStrikesOK)); ok {
			strikesOK := int64(v.(int))
			elasticIPHealthcheck.StrikesOK = &strikesOK
		}

		if v, ok := d.GetOk(resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckTLSSNI)); ok {
			elasticIPHealthcheck.TLSSNI = nonEmptyStringPtr(v.(string))
		}

		if v, ok := d.GetOk(resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckTLSSkipVerify)); ok {
			tlsSkipVerify := v.(bool)
			elasticIPHealthcheck.TLSSkipVerify = &tlsSkipVerify
		}

		if v, ok := d.GetOk(resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckTimeout)); ok {
			timeout := time.Duration(v.(int)) * time.Second
			elasticIPHealthcheck.Timeout = &timeout
		}

		if v, ok := d.GetOk(resElasticIPAttrHealthcheck(resElasticIPAttrHealthcheckURI)); ok {
			elasticIPHealthcheck.URI = nonEmptyStringPtr(v.(string))
		}

		elasticIP.Healthcheck = &elasticIPHealthcheck

		if l, ok := d.GetOk(resElasticIPAttrLabels); ok {
			labels := make(map[string]string)
			for k, v := range l.(map[string]interface{}) {
				labels[k] = v.(string)
			}
			elasticIP.Labels = &labels
		}
	}

	elasticIP, err := client.CreateElasticIP(ctx, zone, elasticIP)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*elasticIP.ID)

	tflog.Debug(ctx, "create finished successfully", map[string]interface{}{
		"id": resourceElasticIPIDString(d),
	})

	return resourceElasticIPRead(ctx, d, meta)
}

func resourceElasticIPRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceElasticIPIDString(d),
	})

	zone := d.Get(resElasticIPAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	elasticIP, err := client.GetElasticIP(ctx, zone, d.Id())
	if err != nil {
		if errors.Is(err, exoapi.ErrNotFound) {
			// Resource doesn't exist anymore, signaling the core to remove it from the state.
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceElasticIPIDString(d),
	})

	return diag.FromErr(resourceElasticIPApply(ctx, d, elasticIP))
}

func resourceElasticIPUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning update", map[string]interface{}{
		"id": resourceElasticIPIDString(d),
	})

	zone := d.Get(resElasticIPAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	elasticIP, err := client.GetElasticIP(ctx, zone, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	var updated bool

	if d.HasChange(resElasticIPAttrLabels) {
		labels := make(map[string]string)
		for k, v := range d.Get(resElasticIPAttrLabels).(map[string]interface{}) {
			labels[k] = v.(string)
		}
		elasticIP.Labels = &labels
		updated = true
	}

	if d.HasChange(resElasticIPAttrAddressFamily) {
		v := d.Get(resElasticIPAttrAddressFamily).(string)
		if v != "" {
			elasticIP.AddressFamily = &v
		}
		// note that nil value is also considered change
		updated = true
	}

	if d.HasChange(resElasticIPAttrDescription) {
		v := d.Get(resElasticIPAttrDescription).(string)
		elasticIP.Description = &v
		updated = true
	}

	if updated {
		if err = client.UpdateElasticIP(ctx, zone, elasticIP); err != nil {
			return diag.FromErr(err)
		}
	}

	tflog.Debug(ctx, "update finished successfully", map[string]interface{}{
		"id": resourceElasticIPIDString(d),
	})

	return resourceElasticIPRead(ctx, d, meta)
}

func resourceElasticIPDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning delete", map[string]interface{}{
		"id": resourceElasticIPIDString(d),
	})

	zone := d.Get(resElasticIPAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutDelete))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	elasticIPID := d.Id()
	if err := client.DeleteElasticIP(ctx, zone, &egoscale.ElasticIP{ID: &elasticIPID}); err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "delete finished successfully", map[string]interface{}{
		"id": resourceDomainIDString(d),
	})

	return nil
}

func resourceElasticIPApply(_ context.Context, d *schema.ResourceData, elasticIP *egoscale.ElasticIP) error {
	if err := d.Set(resElasticIPAttrAddressFamily, defaultString(elasticIP.AddressFamily, "")); err != nil {
		return err
	}
	if err := d.Set(resElasticIPAttrCIDR, defaultString(elasticIP.CIDR, "")); err != nil {
		return err
	}
	if err := d.Set(resElasticIPAttrDescription, defaultString(elasticIP.Description, "")); err != nil {
		return err
	}

	if elasticIP.Healthcheck != nil {
		elasticIPHealthcheck := map[string]interface{}{
			resElasticIPAttrHealthcheckInterval:      elasticIP.Healthcheck.Interval.Seconds(),
			resElasticIPAttrHealthcheckMode:          *elasticIP.Healthcheck.Mode,
			resElasticIPAttrHealthcheckPort:          int(*elasticIP.Healthcheck.Port),
			resElasticIPAttrHealthcheckStrikesFail:   int(*elasticIP.Healthcheck.StrikesFail),
			resElasticIPAttrHealthcheckStrikesOK:     int(*elasticIP.Healthcheck.StrikesOK),
			resElasticIPAttrHealthcheckTLSSNI:        defaultString(elasticIP.Healthcheck.TLSSNI, ""),
			resElasticIPAttrHealthcheckTLSSkipVerify: defaultBool(elasticIP.Healthcheck.TLSSkipVerify, false),
			resElasticIPAttrHealthcheckTimeout:       elasticIP.Healthcheck.Timeout.Seconds(),
			resElasticIPAttrHealthcheckURI:           defaultString(elasticIP.Healthcheck.URI, ""),
		}

		if err := d.Set("healthcheck", []interface{}{elasticIPHealthcheck}); err != nil {
			return err
		}
	}

	if elasticIP.IPAddress != nil {
		if err := d.Set(resElasticIPAttrIPAddress, elasticIP.IPAddress.String()); err != nil {
			return err
		}
	}

	if err := d.Set(resElasticIPAttrLabels, elasticIP.Labels); err != nil {
		return err
	}

	return nil
}

// resElasticIPAttrHealthcheck returns an elastic_ip resource attribute key formatted for a "healthcheck {}" block.
func resElasticIPAttrHealthcheck(a string) string { return fmt.Sprintf("healthcheck.0.%s", a) }
