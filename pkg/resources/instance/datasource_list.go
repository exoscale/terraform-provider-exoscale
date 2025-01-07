package instance

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	v3 "github.com/exoscale/egoscale/v3"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/filter"
	"github.com/exoscale/terraform-provider-exoscale/utils"
)

func DataSourceList() *schema.Resource {
	ret := &schema.Resource{
		Description: `List Exoscale [Compute Instances](https://community.exoscale.com/documentation/compute/).

Corresponding resource: [exoscale_compute_instance](../resources/compute_instance.md).`,
		Schema: map[string]*schema.Schema{
			AttrZone: {
				Description: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
				Type:        schema.TypeString,
				Required:    true,
			},

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

	listInstancesResponse, err := client.ListInstances(
		ctx,
	)
	if err != nil {
		return diag.FromErr(err)
	}

	data := make([]interface{}, 0, len(listInstancesResponse.Instances))
	ids := make([]string, 0, len(listInstancesResponse.Instances))
	instanceTypes := map[v3.UUID]string{}

	filters, err := filter.CreateFilters(ctx, d, DataSourceSchema())
	if err != nil {
		return diag.Errorf("failed to create filter: %q", err)
	}

	filteredFields := filter.GetFilteredFields(ctx, d, DataSourceSchema())

	for _, listInst := range listInstancesResponse.Instances {
		// we use ID to generate a resource ID, we cannot list instances without ID.
		if listInst.ID == "" {
			continue
		}

		ids = append(ids, listInst.ID.String())

		inst, err := client.GetInstance(
			ctx,
			listInst.ID,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		instanceData, err := dsBuildData(inst, zone)
		if err != nil {
			return diag.FromErr(err)
		}

		getInstanceReverseDNS := func() diag.Diagnostics {
			rdns, err := client.GetReverseDNSInstance(ctx, inst.ID)
			if err != nil && !errors.Is(err, v3.ErrNotFound) {
				return diag.Errorf("unable to retrieve instance reverse-dns: %s", err)
			}
			instanceData[AttrReverseDNS] = string(rdns.DomainName)
			return nil
		}

		// to save time the reverse DNS is only fetched if it's filtered
		// if not that it isn't filtered it will be fetched later if the
		// instance actually passes the filter
		_, reverseDNSFiltered := filteredFields[AttrReverseDNS]
		if reverseDNSFiltered {
			if diagn := getInstanceReverseDNS(); diagn != nil {
				return diagn
			}
		}

		// The API returns the instance type as a UUID.
		// We lazily convert it to the <family>.<size> format.
		instanceType := ""
		if inst.InstanceType != nil && inst.InstanceType.ID != "" {
			tid := inst.InstanceType.ID
			if _, ok := instanceTypes[tid]; !ok {
				instanceType, err := client.GetInstanceType(
					ctx,
					tid,
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

			instanceType = instanceTypes[tid]

			instanceData[AttrType] = instanceType
		}

		if !filter.CheckForMatch(instanceData, filters) {
			continue
		}

		if !reverseDNSFiltered {
			if diagn := getInstanceReverseDNS(); diagn != nil {
				return diagn
			}
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
