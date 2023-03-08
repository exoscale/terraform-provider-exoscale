package exoscale

import (
	"context"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	dsAntiAffinityGroupAttrID        = "id"
	dsAntiAffinityGroupAttrInstances = "instances"
	dsAntiAffinityGroupAttrName      = "name"
)

func dataSourceAntiAffinityGroup() *schema.Resource {
	return &schema.Resource{
		Description: `Fetch Exoscale [Anti-Affinity Groups](https://community.exoscale.com/documentation/compute/anti-affinity-groups/) data.

Corresponding resource: [exoscale_anti_affinity_group](../resources/anti_affinity_group.md).`,
		Schema: map[string]*schema.Schema{
			dsAntiAffinityGroupAttrID: {
				Description:   "The anti-affinity group ID to match (conflicts with `name`).",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsAntiAffinityGroupAttrName},
			},
			dsAntiAffinityGroupAttrInstances: {
				Description: "The list of attached [exoscale_compute_instance](../resources/compute_instance.md) (IDs).",
				Type:        schema.TypeSet,
				Computed:    true,
				Set:         schema.HashString,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			dsAntiAffinityGroupAttrName: {
				Description:   "The group name to match (conflicts with `id`).",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsAntiAffinityGroupAttrID},
			},
		},

		ReadContext: dataSourceAntiAffinityGroupRead,
	}
}

func dataSourceAntiAffinityGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceAntiAffinityGroupIDString(d),
	})

	zone := defaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	antiAffinityGroupID, byAntiAffinityGroupID := d.GetOk(dsAntiAffinityGroupAttrID)
	antiAffinityGroupName, byAntiAffinityGroupName := d.GetOk(dsAntiAffinityGroupAttrName)
	if !byAntiAffinityGroupID && !byAntiAffinityGroupName {
		return diag.Errorf(
			"either %s or %s must be specified",
			dsAntiAffinityGroupAttrName,
			dsAntiAffinityGroupAttrID,
		)
	}

	antiAffinityGroup, err := client.FindAntiAffinityGroup(
		ctx,
		zone, func() string {
			if byAntiAffinityGroupID {
				return antiAffinityGroupID.(string)
			}
			return antiAffinityGroupName.(string)
		}(),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*antiAffinityGroup.ID)

	if antiAffinityGroup.InstanceIDs != nil {
		instanceIDs := make([]string, len(*antiAffinityGroup.InstanceIDs))
		for i, id := range *antiAffinityGroup.InstanceIDs {
			instanceIDs[i] = id
		}

		if err := d.Set(dsAntiAffinityGroupAttrInstances, instanceIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(dsAntiAffinityGroupAttrName, *antiAffinityGroup.Name); err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceAntiAffinityGroupIDString(d),
	})

	return nil
}
