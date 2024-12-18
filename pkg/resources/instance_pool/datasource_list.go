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

	v3 "github.com/exoscale/egoscale/v3"

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

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutCreate))
	defer cancel()

	defaultClientV3, err := config.GetClientV3(meta)
	if err != nil {
		return diag.FromErr(err)
	}
	client, err := utils.SwitchClientZone(
		ctx,
		defaultClientV3,
		v3.ZoneName(zone),
	)
	if err != nil {
		return diag.FromErr(err)
	}

	poolResponse, err := client.ListInstancePools(
		ctx,
	)
	if err != nil {
		return diag.FromErr(err)
	}
	pools := poolResponse.InstancePools

	data := make([]interface{}, 0, len(pools))
	ids := make([]string, 0, len(pools))
	instanceTypes := map[string]string{}

	for _, item := range pools {
		// we use ID to generate a resource ID, we cannot list instance pools without ID.
		if item.ID == "" {
			continue
		}

		ids = append(ids, item.ID.String())

		pool, err := client.GetInstancePool(
			ctx,
			item.ID,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		poolData, err := dsBuildData(pool, zone)
		if err != nil {
			return diag.FromErr(err)
		}

		if pool.InstanceType != nil {
			tid := pool.InstanceType.ID.String()
			if _, ok := instanceTypes[tid]; !ok {
				instanceType, err := client.GetInstanceType(
					ctx,
					v3.UUID(tid),
				)
				if err != nil {
					return diag.Errorf("unable to retrieve instance type: %s", err)
				}
				instanceTypes[tid] = fmt.Sprintf(
					"%s.%s",
					strings.ToLower(string(instanceType.Family)),
					strings.ToLower(string(instanceType.Size)),
				)
			}

			poolData[AttrInstanceType] = instanceTypes[tid]
		}

		if pool.Instances != nil {
			instancesData := make([]interface{}, len(pool.Instances))
			for k, i := range pool.Instances {
				instance, err := client.GetInstance(ctx, i.ID)
				if err != nil {
					return diag.FromErr(err)
				}

				var ipv6, publicIp string
				if instance.Ipv6Address != "" {
					ipv6 = instance.Ipv6Address
				}
				if instance.PublicIP.String() != "" {
					publicIp = instance.PublicIP.String()
				}

				instancesData[k] = map[string]interface{}{
					AttrInstanceID:              i.ID,
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
