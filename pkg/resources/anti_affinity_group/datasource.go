package anti_affinity_group

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	exoapi "github.com/exoscale/egoscale/v2/api"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

func DataSource() *schema.Resource {
	return &schema.Resource{
		Description: `Fetch Exoscale [Anti-Affinity Groups](https://community.exoscale.com/product/compute/instances/how-to/anti-affinity/) data.

Corresponding resource: [exoscale_anti_affinity_group](../resources/anti_affinity_group.md).`,
		Schema: map[string]*schema.Schema{
			AttrID: {
				Description:   "The anti-affinity group ID to match (conflicts with `name`).",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{AttrName},
			},
			AttrInstances: {
				Description: "The list of attached [exoscale_compute_instance](../resources/compute_instance.md) (IDs).",
				Type:        schema.TypeSet,
				Computed:    true,
				Set:         schema.HashString,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
			AttrName: {
				Description:   "The group name to match (conflicts with `id`).",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{AttrID},
			},
		},

		ReadContext: dsRead,
	}
}

func dsRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": utils.IDString(d, Name),
	})

	zone := config.DefaultZone

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(config.GetEnvironment(meta), zone))
	defer cancel()

	client, err := config.GetClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	id, byID := d.GetOk(AttrID)
	name, byName := d.GetOk(AttrName)
	if !byID && !byName {
		return diag.Errorf(
			"either %s or %s must be specified",
			AttrName,
			AttrID,
		)
	}

	res, err := client.FindAntiAffinityGroup(
		ctx,
		zone, func() string {
			if byID {
				return id.(string)
			}
			return name.(string)
		}(),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(*res.ID)

	if res.InstanceIDs != nil {
		instanceIDs := make([]string, len(*res.InstanceIDs))
		copy(instanceIDs, *res.InstanceIDs)

		if err := d.Set(AttrInstances, instanceIDs); err != nil {
			return diag.FromErr(err)
		}
	}

	if err := d.Set(AttrName, *res.Name); err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": utils.IDString(d, Name),
	})

	return nil
}
