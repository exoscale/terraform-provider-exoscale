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
		Description: `Fetch Exoscale [Elastic IPs (EIP)](https://community.exoscale.com/documentation/compute/eip/) data.

Corresponding resource: [exoscale_elastic_ip](../resources/elastic_ip.md).`,
		Schema: map[string]*schema.Schema{
			dsElasticIPAttrAddressFamily: {
				Description: "The Elastic IP (EIP) address family (`inet4` or `inet6`).",
				Type:        schema.TypeString,
				Computed:    true,
			},
			dsElasticIPAttrCIDR: {
				Description: "The Elastic IP (EIP) CIDR.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			dsElasticIPAttrDescription: {
				Description: "The Elastic IP (EIP) description.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"healthcheck": {
				Description: "The *managed* EIP healthcheck configuration.",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						dsElasticIPAttrHealthcheckInterval: {
							Description: "The healthcheck interval in seconds.",
							Type:        schema.TypeInt,
							Computed:    true,
						},
						dsElasticIPAttrHealthcheckMode: {
							Description: "The healthcheck mode.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						dsElasticIPAttrHealthcheckPort: {
							Description: "The healthcheck target port.",
							Type:        schema.TypeInt,
							Computed:    true,
						},
						dsElasticIPAttrHealthcheckTimeout: {
							Description: "The time in seconds before considering a healthcheck probing failed.",
							Type:        schema.TypeInt,
							Computed:    true,
						},
						dsElasticIPAttrHealthcheckStrikesFail: {
							Description: "The number of failed healthcheck attempts before considering the target unhealthy.",
							Type:        schema.TypeInt,
							Computed:    true,
						},
						dsElasticIPAttrHealthcheckStrikesOK: {
							Description: "The number of successful healthcheck attempts before considering the target healthy.",
							Type:        schema.TypeInt,
							Computed:    true,
						},
						dsElasticIPAttrHealthcheckTLSSkipVerify: {
							Description: "Disable TLS certificate verification for healthcheck in `https` mode.",
							Type:        schema.TypeBool,
							Computed:    true,
						},
						dsElasticIPAttrHealthcheckTLSSNI: {
							Description: "The healthcheck server name to present with SNI in `https` mode.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						dsElasticIPAttrHealthcheckURI: {
							Description: "The healthcheck URI.",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
			dsElasticIPAttrID: {
				Description:   "The Elastic IP (EIP) ID to match (conflicts with `ip_address` and `labels`).",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsElasticIPAttrIPAddress, dsElasticIPAttrLabels},
			},
			dsElasticIPAttrIPAddress: {
				Description:   "The EIP IPv4 or IPv6 address to match (conflicts with `id` and `labels`).",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsElasticIPAttrID, dsElasticIPAttrLabels},
			},
			dsElasticIPAttrReverseDNS: {
				Description: "Domain name for reverse DNS record.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			dsElasticIPAttrLabels: {
				Description:   "The EIP labels to match (conflicts with `ip_address` and `id`).",
				Type:          schema.TypeMap,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Optional:      true,
				ConflictsWith: []string{dsElasticIPAttrID, dsElasticIPAttrIPAddress},
			},
			dsElasticIPAttrZone: {
				Description: "The Exocale [Zone](https://www.exoscale.com/datacenters/) name.",
				Type:        schema.TypeString,
				Required:    true,
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

	client := getClient(meta)

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
		return eip.IPAddress.String() == elasticIPAddress
	}

	if searchByElasticIPID {
		filterElasticIP = func(eip *egoscale.ElasticIP) bool {
			return *eip.ID == elasticIPID
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
