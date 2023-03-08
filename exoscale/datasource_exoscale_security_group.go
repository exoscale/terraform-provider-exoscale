package exoscale

import (
	"context"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	dsSecurityGroupAttrID   = "id"
	dsSecurityGroupAttrName = "name"
)

func dataSourceSecurityGroup() *schema.Resource {
	return &schema.Resource{
		Description: `Fetch Exoscale [Security Groups](https://community.exoscale.com/documentation/compute/security-groups/) data.

Corresponding resource: [exoscale_security_group](../resources/security_group.md).`,
		Schema: map[string]*schema.Schema{
			dsSecurityGroupAttrID: {
				Description:   "The security group ID to match (conflicts with `name`)",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsSecurityGroupAttrName},
			},
			dsSecurityGroupAttrName: {
				Description:   "The name to match (conflicts with `id`)",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsSecurityGroupAttrID},
			},
		},

		ReadContext: dataSourceSecurityGroupRead,
	}
}

func dataSourceSecurityGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceSecurityGroupIDString(d),
	})

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

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceSecurityGroupIDString(d),
	})

	return nil
}
