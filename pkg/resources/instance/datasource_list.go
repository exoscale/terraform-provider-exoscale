package instance

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"sort"
	"strings"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/filter"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

func DataSourceList() *schema.Resource {
	ret := &schema.Resource{
		Description: `List Exoscale [Compute Instances](https://community.exoscale.com/documentation/compute/).

Corresponding resource: [exoscale_compute_instance](../resources/compute_instance.md).`,
		Schema: map[string]*schema.Schema{
			"instances": {
				Description: "The list of [exoscale_compute_instance](./compute_instance.md).",
				Type:        schema.TypeList,
				Computed:    true,
				Elem: &schema.Resource{
					Schema: DataSourceSchema(),
				},
			},
		},

		ReadContext: dsListRead,
	}

	filter.AddFilterAttributes(ret, DataSourceSchema())

	return ret
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

	instances, err := client.ListInstances(
		ctx,
		zone,
	)
	if err != nil {
		return diag.FromErr(err)
	}

	data := make([]interface{}, 0, len(instances))
	ids := make([]string, 0, len(instances))
	instanceTypes := map[string]string{}

	filters, err := filter.CreateFilters(ctx, d, DataSourceSchema())
	if err != nil {
		return diag.Errorf("failed to create filter: %q", err)
	}

	for _, item := range instances {
		// we use ID to generate a resource ID, we cannot list instances without ID.
		if item.ID == nil {
			continue
		}

		ids = append(ids, *item.ID)

		testInstance, err := client.FindInstance(
			ctx,
			zone,
			*item.ID,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		instanceData, err := dsBuildData(testInstance)
		if err != nil {
			return diag.FromErr(err)
		}

		rdns, err := client.GetInstanceReverseDNS(ctx, zone, *testInstance.ID)
		if err != nil && !errors.Is(err, exoapi.ErrNotFound) {
			return diag.Errorf("unable to retrieve instance reverse-dns: %s", err)
		}
		instanceData[AttrReverseDNS] = rdns

		if testInstance.InstanceTypeID != nil {
			tid := *testInstance.InstanceTypeID
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

			instanceData[AttrType] = instanceTypes[tid]
		}

		if !filter.CheckForMatch(instanceData, filters) {
			continue
		}

		data = append(data, instanceData)
	}

	err = d.Set("instances", data)
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
