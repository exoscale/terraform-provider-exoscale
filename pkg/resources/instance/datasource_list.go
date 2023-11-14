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

	v2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"

	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/filter"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
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

var instanceListFilledFields = map[string]struct{}{
	AttrID:              {},
	AttrName:            {},
	AttrZone:            {},
	AttrType:            {},
	AttrPublicIPAddress: {},
	AttrState:           {},
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

	onlyListFieldsAreFiltered := true
	filteredFields := filter.GetFilteredFields(ctx, d, DataSourceSchema())
	for fieldName := range filteredFields {
		if _, ok := instanceListFilledFields[fieldName]; !ok {
			onlyListFieldsAreFiltered = false
		}
	}

	for _, listInst := range instances {
		// we use ID to generate a resource ID, we cannot list instances without ID.
		if listInst.ID == nil {
			continue
		}

		ids = append(ids, *listInst.ID)

		var inst *v2.Instance

		// to save time the full instance data is only fetched if a field
		// is filtered which is not already returned by ListInstances.
		// If the instance passes the filter step the full data will be fetched later.
		allInstanceDataFetched := false
		if onlyListFieldsAreFiltered {
			inst = listInst
		} else {
			inst, err = client.GetInstance(
				ctx,
				zone,
				*listInst.ID,
			)
			if err != nil {
				return diag.FromErr(err)
			}

			allInstanceDataFetched = true
		}

		instanceData, err := dsBuildData(inst)
		if err != nil {
			return diag.FromErr(err)
		}

		getInstanceReverseDNS := func() diag.Diagnostics {
			rdns, err := client.GetInstanceReverseDNS(ctx, zone, *inst.ID)
			if err != nil && !errors.Is(err, exoapi.ErrNotFound) {
				return diag.Errorf("unable to retrieve instance reverse-dns: %s", err)
			}
			instanceData[AttrReverseDNS] = rdns

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
		if inst.InstanceTypeID != nil {
			tid := *inst.InstanceTypeID
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

			instanceType = instanceTypes[tid]

			instanceData[AttrType] = instanceType
		}

		if !filter.CheckForMatch(instanceData, filters) {
			continue
		}

		if !allInstanceDataFetched {
			inst, err = client.GetInstance(
				ctx,
				zone,
				*listInst.ID,
			)
			if err != nil {
				return diag.FromErr(err)
			}

			instanceData, err = dsBuildData(inst)
			if err != nil {
				return diag.FromErr(err)
			}

			instanceData[AttrType] = instanceType
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
