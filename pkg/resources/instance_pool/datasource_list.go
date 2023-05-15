package instance_pool

import (
	"context"
	"crypto/md5"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	exoapi "github.com/exoscale/egoscale/v2/api"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

func DataSourceList() *schema.Resource {
	return &schema.Resource{
		Description: `List Exoscale [Instance Pools](https://community.exoscale.com/documentation/compute/instance-pools/).

Corresponding resource: [exoscale_instance_pool](../resources/instance_pool.md).`,
		Schema: map[string]*schema.Schema{
			AttrZone: {
				Description: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"pools": {
				Description: "The list of [exoscale_instance_pool](./instance_pool.md).",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: DataSourceSchema(),
				},
			},
		},

		ReadContext: dsListRead,
	}
}

func dsListRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": utils.IDString(d, NameList),
	})

	zone := d.Get(AttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(config.GetEnvironment(meta), zone))
	defer cancel()

	client, err := config.GetClient(meta)
	if err != nil {
		return diag.FromErr(err)
	}

	pools, err := client.ListInstancePools(
		ctx,
		zone,
	)
	if err != nil {
		return diag.FromErr(err)
	}

	data := make([]interface{}, 0, len(pools))
	ids := make([]string, 0, len(pools))
	instanceTypes := map[string]string{}

	for _, item := range pools {
		// we use ID to generate a resource ID, we cannot list instance pools without ID.
		if item.ID == nil {
			continue
		}

		ids = append(ids, *item.ID)

		pool, err := client.FindInstancePool(
			ctx,
			zone,
			*item.ID,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		poolData, err := dsBuildData(pool)
		if err != nil {
			return diag.FromErr(err)
		}

		if pool.InstanceTypeID != nil {
			tid := *pool.InstanceTypeID
			if _, ok := instanceTypes[tid]; !ok {
				instanceType, err := client.GetInstanceType(
					ctx,
					zone,
					tid,
				)
				if err != nil {
					return diag.Errorf("unable to retrieve instance type: %s", err)
				}
				instanceTypes[tid] = fmt.Sprintf(
					"%s.%s",
					strings.ToLower(*instanceType.Family),
					strings.ToLower(*instanceType.Size),
				)
			}

			poolData[AttrInstanceType] = instanceTypes[tid]
		}

		if pool.InstanceIDs != nil {
			instancesData := make([]interface{}, len(*pool.InstanceIDs))
			for i, id := range *pool.InstanceIDs {
				instance, err := client.GetInstance(ctx, zone, id)
				if err != nil {
					return diag.FromErr(err)
				}

				var ipv6, publicIp string
				if instance.IPv6Address != nil {
					ipv6 = instance.IPv6Address.String()
				}
				if instance.PublicIPAddress != nil {
					publicIp = instance.PublicIPAddress.String()
				}

				instancesData[i] = map[string]interface{}{
					AttrInstanceID:              id,
					AttrInstanceIPv6Address:     ipv6,
					AttrInstanceName:            instance.Name,
					AttrInstancePublicIPAddress: publicIp,
				}
			}

			poolData[AttrInstances] = instancesData
		}

		data = append(data, poolData)
	}

	err = d.Set("pools", &data)
	if err != nil {
		return diag.FromErr(err)
	}

	// by sorting instance IDs we can generate the same resource ID regardless of the order in which
	// API returns instances in thelist.
	sort.Strings(ids)

	d.SetId(fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(ids, "")))))

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": utils.IDString(d, NameList),
	})

	return nil
}
