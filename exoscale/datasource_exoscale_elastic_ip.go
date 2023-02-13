package exoscale

import (
	"context"
	"errors"
	"fmt"
	"strings"

	egoscale "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	dsElasticIPAttrAddressFamily            = "address_family"
	dsElasticIPAttrCIDR                     = "cidr"
	dsElasticIPAttrDescription              = "description"
	dsElasticIPAttrHealthcheckInterval      = "interval"
	dsElasticIPAttrHealthcheckMode          = "mode"
	dsElasticIPAttrHealthcheckPort          = "port"
	dsElasticIPAttrHealthcheckStrikesFail   = "strikes_fail"
	dsElasticIPAttrHealthcheckStrikesOK     = "strikes_ok"
	dsElasticIPAttrHealthcheckTLSSNI        = "tls_sni"
	dsElasticIPAttrHealthcheckTLSSkipVerify = "tls_skip_verify"
	dsElasticIPAttrHealthcheckTimeout       = "timeout"
	dsElasticIPAttrHealthcheckURI           = "uri"
	dsElasticIPAttrID                       = "id"
	dsElasticIPAttrIPAddress                = "ip_address"
	dsElasticIPAttrReverseDNS               = "reverse_dns"
	dsElasticIPAttrLabels                   = "labels"
	dsElasticIPAttrZone                     = "zone"
)

func dataSourceElasticIP() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			dsElasticIPAttrAddressFamily: {
				Type:     schema.TypeString,
				Computed: true,
			},
			dsElasticIPAttrCIDR: {
				Type:     schema.TypeString,
				Computed: true,
			},
			dsElasticIPAttrDescription: {
				Type:     schema.TypeString,
				Computed: true,
			},
			"healthcheck": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						dsElasticIPAttrHealthcheckInterval: {
							Type:     schema.TypeInt,
							Computed: true,
						},
						dsElasticIPAttrHealthcheckMode: {
							Type:     schema.TypeString,
							Computed: true,
						},
						dsElasticIPAttrHealthcheckPort: {
							Type:     schema.TypeInt,
							Computed: true,
						},
						dsElasticIPAttrHealthcheckTimeout: {
							Type:     schema.TypeInt,
							Computed: true,
						},
						dsElasticIPAttrHealthcheckStrikesFail: {
							Type:     schema.TypeInt,
							Computed: true,
						},
						dsElasticIPAttrHealthcheckStrikesOK: {
							Type:     schema.TypeInt,
							Computed: true,
						},
						dsElasticIPAttrHealthcheckTLSSkipVerify: {
							Type:     schema.TypeBool,
							Computed: true,
						},
						dsElasticIPAttrHealthcheckTLSSNI: {
							Type:     schema.TypeString,
							Computed: true,
						},
						dsElasticIPAttrHealthcheckURI: {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			dsElasticIPAttrID: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsElasticIPAttrIPAddress, dsElasticIPAttrLabels},
			},
			dsElasticIPAttrIPAddress: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsElasticIPAttrID, dsElasticIPAttrLabels},
			},
			dsElasticIPAttrReverseDNS: {
				Type:     schema.TypeString,
				Computed: true,
			},
			dsElasticIPAttrLabels: {
				Type:          schema.TypeMap,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{dsElasticIPAttrID, dsElasticIPAttrIPAddress},
			},
			dsElasticIPAttrZone: {
				Type:     schema.TypeString,
				Required: true,
			},
		},

		ReadContext: dataSourceElasticIPRead,
	}
}

func dataSourceElasticIPRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceElasticIPIDString(d),
	})

	zone := d.Get(dsElasticIPAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	elasticIPID, searchByElasticIPID := d.GetOk(dsElasticIPAttrID)
	elasticIPAddress, searchByElasticIPAddress := d.GetOk(dsElasticIPAttrIPAddress)
	elasticIPLabels, searchByElasticIPLabels := d.GetOk(dsElasticIPAttrLabels)
	if !searchByElasticIPID && !searchByElasticIPAddress && !searchByElasticIPLabels {
		return diag.Errorf(
			"one of %s, %s or %s must be specified",
			dsElasticIPAttrIPAddress,
			dsElasticIPAttrID,
			dsElasticIPAttrLabels,
		)
	}

	// search by address by default
	filterElasticIP := func(eip *egoscale.ElasticIP) bool {
		if eip.IPAddress.String() == elasticIPAddress {
			return true
		}
		return false
	}

	if searchByElasticIPID {
		filterElasticIP = func(eip *egoscale.ElasticIP) bool {
			if *eip.ID == elasticIPID {
				return true
			}
			return false
		}
	}

	if searchByElasticIPLabels {
		filterElasticIP = func(eip *egoscale.ElasticIP) bool {
			if eip.Labels == nil {
				return false
			}

			for searchKey, searchValue := range elasticIPLabels.(map[string]interface{}) {
				v, ok := (*eip.Labels)[searchKey]
				if !ok || v != searchValue {
					return false
				}
			}

			return true
		}
	}

	elasticIPs, err := client.ListElasticIPs(ctx, zone)
	if err != nil {
		return diag.FromErr(err)
	}

	var elasticIP *egoscale.ElasticIP
	for _, eip := range elasticIPs {
		if filterElasticIP(eip) {
			elasticIP = eip
			break
		}
	}

	if elasticIP == nil {
		return diag.FromErr(fmt.Errorf("Unable to find matching ElasticIP"))
	}

	d.SetId(*elasticIP.ID)

	if err := d.Set(dsElasticIPAttrAddressFamily, defaultString(elasticIP.AddressFamily, "")); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(dsElasticIPAttrCIDR, defaultString(elasticIP.CIDR, "")); err != nil {
		return diag.FromErr(err)
	}
	if err := d.Set(dsElasticIPAttrDescription, defaultString(elasticIP.Description, "")); err != nil {
		return diag.FromErr(err)
	}

	if elasticIP.Healthcheck != nil {
		elasticIPHealthcheck := map[string]interface{}{
			dsElasticIPAttrHealthcheckInterval:      elasticIP.Healthcheck.Interval.Seconds(),
			dsElasticIPAttrHealthcheckMode:          *elasticIP.Healthcheck.Mode,
			dsElasticIPAttrHealthcheckPort:          int(*elasticIP.Healthcheck.Port),
			dsElasticIPAttrHealthcheckStrikesFail:   int(*elasticIP.Healthcheck.StrikesFail),
			dsElasticIPAttrHealthcheckStrikesOK:     int(*elasticIP.Healthcheck.StrikesOK),
			dsElasticIPAttrHealthcheckTLSSNI:        defaultString(elasticIP.Healthcheck.TLSSNI, ""),
			dsElasticIPAttrHealthcheckTLSSkipVerify: defaultBool(elasticIP.Healthcheck.TLSSkipVerify, false),
			dsElasticIPAttrHealthcheckTimeout:       elasticIP.Healthcheck.Timeout.Seconds(),
			dsElasticIPAttrHealthcheckURI:           defaultString(elasticIP.Healthcheck.URI, ""),
		}

		if err := d.Set("healthcheck", []interface{}{elasticIPHealthcheck}); err != nil {
			return diag.FromErr(err)
		}
	}

	if elasticIP.IPAddress != nil {
		if err := d.Set(dsElasticIPAttrIPAddress, elasticIP.IPAddress.String()); err != nil {
			return diag.FromErr(err)
		}
	}

	rdns, err := client.GetElasticIPReverseDNS(ctx, zone, *elasticIP.ID)
	if err != nil && !errors.Is(err, exoapi.ErrNotFound) {
		return diag.Errorf("unable to retrieve instance reverse-dns: %s", err)
	}
	if err := d.Set(dsElasticIPAttrReverseDNS, strings.TrimSuffix(rdns, ".")); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(dsElasticIPAttrLabels, elasticIP.Labels); err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceElasticIPIDString(d),
	})

	return nil
}
