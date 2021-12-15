package exoscale

import (
	"context"
	"log"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
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
	dsElasticIPAttrZone                     = "zone"
)

func dataSourceElasticIP() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
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
				ConflictsWith: []string{dsElasticIPAttrIPAddress},
			},
			dsElasticIPAttrIPAddress: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsElasticIPAttrID},
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
	log.Printf("[DEBUG] %s: beginning read", resourceElasticIPIDString(d))

	zone := d.Get(dsElasticIPAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	elasticIPID, byElasticIPID := d.GetOk(dsElasticIPAttrID)
	elasticIPAddress, byElasticIPAddress := d.GetOk(dsElasticIPAttrIPAddress)
	if !byElasticIPID && !byElasticIPAddress {
		return diag.Errorf(
			"either %s or %s must be specified",
			dsElasticIPAttrIPAddress,
			dsElasticIPAttrID,
		)
	}

	elasticIP, err := client.FindElasticIP(
		ctx,
		zone, func() string {
			if byElasticIPID {
				return elasticIPID.(string)
			}
			return elasticIPAddress.(string)
		}(),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*elasticIP.ID)

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

	log.Printf("[DEBUG] %s: read finished successfully", resourceElasticIPIDString(d))

	return nil
}
