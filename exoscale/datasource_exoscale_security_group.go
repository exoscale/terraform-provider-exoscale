package exoscale

import (
	"context"
	"log"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	dsSecurityGroupAttrID   = "id"
	dsSecurityGroupAttrName = "name"
)

func dataSourceSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			dsSecurityGroupAttrID: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsSecurityGroupAttrName},
			},
			dsSecurityGroupAttrName: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsSecurityGroupAttrID},
			},
		},

		ReadContext: dataSourceSecurityGroupRead,
	}
}

func dataSourceSecurityGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning read", resourceSecurityGroupIDString(d))

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	securityGroupID, bySecurityGroupID := d.GetOk(dsSecurityGroupAttrID)
	securityGroupName, bySecurityGroupName := d.GetOk(dsSecurityGroupAttrName)
	if !bySecurityGroupID && !bySecurityGroupName {
		return diag.Errorf(
			"either %s or %s must be specified",
			dsSecurityGroupAttrName,
			dsSecurityGroupAttrID,
		)
	}

	securityGroup, err := client.FindSecurityGroup(
		ctx,
		zone, func() string {
			if bySecurityGroupID {
				return securityGroupID.(string)
			}
			return securityGroupName.(string)
		}(),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*securityGroup.ID)

	if err := d.Set(dsSecurityGroupAttrName, *securityGroup.Name); err != nil {
		return diag.FromErr(err)
	}

	log.Printf("[DEBUG] %s: read finished successfully", resourceSecurityGroupIDString(d))

	return nil
}
