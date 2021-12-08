package exoscale

import (
	"context"
	"log"

	exoapi "github.com/exoscale/egoscale/v2/api"
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
		Schema: map[string]*schema.Schema{
			dsAntiAffinityGroupAttrID: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsAntiAffinityGroupAttrName},
			},
			dsAntiAffinityGroupAttrInstances: {
				Type:     schema.TypeSet,
				Computed: true,
				Set:      schema.HashString,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			dsAntiAffinityGroupAttrName: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{dsAntiAffinityGroupAttrID},
			},
		},

		ReadContext: dataSourceAntiAffinityGroupRead,
	}
}

func dataSourceAntiAffinityGroupRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[DEBUG] %s: beginning read", resourceAntiAffinityGroupIDString(d))

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

	log.Printf("[DEBUG] %s: read finished successfully", resourceAntiAffinityGroupIDString(d))

	return nil
}
