package exoscale

import (
	"context"
	"log"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	dsPrivateNetworkAttrDescription = "description"
	dsPrivateNetworkAttrEndIP       = "end_ip"
	dsPrivateNetworkAttrID          = "id"
	dsPrivateNetworkAttrName        = "name"
	dsPrivateNetworkAttrNetmask     = "netmask"
	dsPrivateNetworkAttrStartIP     = "start_ip"
	dsPrivateNetworkAttrZone        = "zone"
)

func dataSourcePrivateNetwork() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			dsPrivateNetworkAttrDescription: {
				Type:     schema.TypeString,
				Optional: true,
			},
			dsPrivateNetworkAttrEndIP: {
				Type:     schema.TypeString,
				Computed: true,
			},
			dsPrivateNetworkAttrID: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsPrivateNetworkAttrName},
			},
			dsPrivateNetworkAttrName: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsPrivateNetworkAttrID},
			},
			dsPrivateNetworkAttrNetmask: {
				Type:     schema.TypeString,
				Computed: true,
			},
			dsPrivateNetworkAttrStartIP: {
				Type:     schema.TypeString,
				Computed: true,
			},
			dsPrivateNetworkAttrZone: {
				Type:     schema.TypeString,
				Required: true,
			},
		},

		ReadContext: dataSourcePrivateNetworkRead,
	}
}

func dataSourcePrivateNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning read", resourcePrivateNetworkIDString(d))

	zone := d.Get(dsPrivateNetworkAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	privateNetworkID, byPrivateNetworkID := d.GetOk(dsPrivateNetworkAttrID)
	privateNetworkName, byPrivateNetworkName := d.GetOk(dsPrivateNetworkAttrName)
	if !byPrivateNetworkID && !byPrivateNetworkName {
		return diag.Errorf(
			"either %s or %s must be specified",
			dsPrivateNetworkAttrName,
			dsPrivateNetworkAttrID,
		)
	}

	privateNetwork, err := client.FindPrivateNetwork(
		ctx,
		zone, func() string {
			if byPrivateNetworkID {
				return privateNetworkID.(string)
			}
			return privateNetworkName.(string)
		}(),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*privateNetwork.ID)

	if err := d.Set(dsPrivateNetworkAttrDescription, defaultString(privateNetwork.Description, "")); err != nil {
		return diag.FromErr(err)
	}

	if privateNetwork.EndIP != nil {
		if err := d.Set(dsPrivateNetworkAttrEndIP, privateNetwork.EndIP.String()); err != nil {
			return diag.FromErr(err)
		}
	}

	if privateNetwork.Netmask != nil {
		if err := d.Set(dsPrivateNetworkAttrNetmask, privateNetwork.Netmask.String()); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(dsPrivateNetworkAttrName, *privateNetwork.Name); err != nil {
		return diag.FromErr(err)
	}

	if privateNetwork.StartIP != nil {
		if err := d.Set(dsPrivateNetworkAttrStartIP, privateNetwork.StartIP.String()); err != nil {
			return diag.FromErr(err)
		}
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourcePrivateNetworkIDString(d))

	return nil
}
