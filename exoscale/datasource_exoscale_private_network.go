package exoscale

import (
	"context"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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

	dsPrivateNetworkStartEndIPDescription = "The first/last IPv4 addresses used by the DHCP service for dynamic leases."
)

func dataSourcePrivateNetwork() *schema.Resource {
	return &schema.Resource{
		Description: `Fetch Exoscale [Private Networks](https://community.exoscale.com/documentation/compute/private-networks/) data.

Corresponding resource: [exoscale_private_network](../resources/private_network.md).`,
		Schema: map[string]*schema.Schema{
			dsPrivateNetworkAttrDescription: {
				Description: "The private network description.",
				Type:        schema.TypeString,
				Optional:    true,
			},
			dsPrivateNetworkAttrEndIP: {
				Description: dsPrivateNetworkStartEndIPDescription,
				Type:        schema.TypeString,
				Computed:    true,
			},
			dsPrivateNetworkAttrID: {
				Description:   "The private network ID to match (conflicts with `name`).",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsPrivateNetworkAttrName},
			},
			dsPrivateNetworkAttrName: {
				Description:   "The network name to match (conflicts with `id`).",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsPrivateNetworkAttrID},
			},
			dsPrivateNetworkAttrNetmask: {
				Description: "The network mask defining the IPv4 network allowed for static leases.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			dsPrivateNetworkAttrStartIP: {
				Description: dsPrivateNetworkStartEndIPDescription,
				Type:        schema.TypeString,
				Computed:    true,
			},
			dsPrivateNetworkAttrZone: {
				Description: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Type:        schema.TypeString,
				Required:    true,
			},
		},

		ReadContext: dataSourcePrivateNetworkRead,
	}
}

func dataSourcePrivateNetworkRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourcePrivateNetworkIDString(d),
	})

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

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourcePrivateNetworkIDString(d),
	})

	return nil
}
