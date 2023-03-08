package exoscale

import (
	"context"
	"errors"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	dsNLBAttrCreatedAt   = "created_at"
	dsNLBAttrDescription = "description"
	dsNLBAttrID          = "id"
	dsNLBAttrIPAddress   = "ip_address"
	dsNLBAttrName        = "name"
	dsNLBAttrState       = "state"
	dsNLBAttrZone        = "zone"
)

func dataSourceNLB() *schema.Resource {
	return &schema.Resource{
		Description: `Fetch Exoscale [Network Load Balancers (NLB)](https://community.exoscale.com/documentation/compute/network-load-balancer/) data.

Corresponding resource: [exoscale_nlb](../resources/nlb.md).`,
		Schema: map[string]*schema.Schema{
			dsNLBAttrCreatedAt: {
				Description: "The NLB creation date.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			dsNLBAttrDescription: {
				Description: "The Network Load Balancers (NLB) description.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			dsNLBAttrID: {
				Description:   "The Network Load Balancers (NLB) ID to match (conflicts with `name`).",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsNLBAttrName},
			},
			dsNLBAttrIPAddress: {
				Description: "The NLB public IPv4 address.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			dsNLBAttrName: {
				Description:   "The NLB name to match (conflicts with `id`).",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsNLBAttrID},
			},
			dsNLBAttrState: {
				Description: "The current NLB state.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			dsNLBAttrZone: {
				Description: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Type:        schema.TypeString,
				Required:    true,
			},
		},

		ReadContext: dataSourceNLBRead,
	}
}

func dataSourceNLBRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	zone := d.Get(dsNLBAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	var x string
	_, byID := d.GetOk(dsNLBAttrID)
	_, byName := d.GetOk(dsNLBAttrName)
	switch {
	case byID:
		x = d.Get(dsNLBAttrID).(string)

	case byName:
		x = d.Get(dsNLBAttrName).(string)

	default:
		return diag.FromErr(errors.New("either name or id must be specified"))
	}

	nlb, err := client.FindNetworkLoadBalancer(ctx, zone, x)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*nlb.ID)

	if err := d.Set(dsNLBAttrID, d.Id()); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(dsNLBAttrName, nlb.Name); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(dsNLBAttrDescription, nlb.Description); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(dsNLBAttrCreatedAt, nlb.CreatedAt.String()); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(dsNLBAttrState, nlb.State); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set(dsNLBAttrIPAddress, nlb.IPAddress.String()); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
