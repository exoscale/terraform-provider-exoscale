package exoscale

import (
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	filterStringPropName = "match"
	filterLabelsPropName = "labels"
	attributePropName    = "attribute"
	matchPropName        = "match"
)

type matchStringFunc = func(given string) bool

func createMatchStringFunc(expected string) (matchStringFunc, error) {
	lenExp := len(expected)
	if lenExp > 1 && expected[0] == '/' && expected[lenExp-1] == '/' {
		r, err := regexp.Compile(expected[1 : lenExp-1])
		if err != nil {
			return nil, err
		}

		return func(given string) bool {
			return r.MatchString(given)
		}, nil
	}

	return func(given string) bool {
		return given == expected
	}, nil
}

func createStringFilterFuncs(stringFilterProp interface{}) ([]filterFunc, error) {
	set := stringFilterProp.(*schema.Set)

	var filters []filterFunc

	for _, v := range set.List() {
		m := v.(map[string]interface{})

		match, err := createMatchStringFunc(m[matchPropName].(string))
		if err != nil {
			return nil, err
		}

		filters = append(filters, createStringFilterFunc(m[attributePropName].(string), match))
	}

	return filters, nil
}

func filterStringSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				attributePropName: {
					Type:     schema.TypeString,
					Required: true,
				},
				matchPropName: {
					Type:     schema.TypeString,
					Required: true,
				},
			},
		},
	}
}

func filterLabelsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Elem:     &schema.Schema{Type: schema.TypeString},
		Optional: true,
	}
}

func dataSourceComputeInstanceList() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			dsComputeInstanceAttrZone: {
				Type:     schema.TypeString,
				Required: true,
			},
			"instances": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: getDataSourceComputeInstanceSchema(),
				},
			},
			filterStringPropName: filterStringSchema(),
			filterLabelsPropName: filterLabelsSchema(),
		},

		ReadContext: dataSourceComputeInstanceListRead,
	}
}

type filterFunc = func(map[string]interface{}) bool

func createStringFilterFunc(filterAttribute string, match matchStringFunc) filterFunc {
	return func(data map[string]interface{}) bool {
		attr, ok := data[filterAttribute]
		if !ok {
			return false
		}

		switch v := attr.(type) {
		case string:
			if match(v) {
				return true
			}
		case *string:
			if match(*v) {
				return true
			}
		}

		return false
	}
}

func createLabelFilterFunc(ctx context.Context, labelsFilterProp interface{}) (filterFunc, error) {
	labelFilters := make(map[string]matchStringFunc)
	labels := labelsFilterProp.(map[string]interface{})
	for k, v := range labels {
		filter, err := createMatchStringFunc(v.(string))
		if err != nil {
			return nil, err
		}

		labelFilters[k] = filter
	}

	return func(data map[string]interface{}) bool {
		labelsAttr, ok := data["labels"]
		if !ok {
			return false
		}

		labels, isMap := labelsAttr.(map[string]string)
		if !isMap {
			tflog.Info(ctx, fmt.Sprintf("attribute of compute instance has unexpected type %T for labels", labelsAttr))

			return false
		}

		for filterKey, filterValue := range labelFilters {
			value, ok := labels[filterKey]
			if !ok || !filterValue(value) {
				return false
			}
		}

		return true
	}, nil
}

func checkForMatch(data map[string]interface{}, filters []filterFunc) bool {
	for _, filter := range filters {
		if filter(data) {
			return true
		}
	}

	return false
}

func dataSourceComputeInstanceListRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceIDString(d, "exoscale_compute_instance_list"),
	})

	zone := d.Get(dsComputeInstanceAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

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

	var filters []filterFunc

	strFilterProp, stringFiltersSpecified := d.GetOk(filterStringPropName)
	if stringFiltersSpecified {
		newFilters, err := createStringFilterFuncs(strFilterProp)
		if err != nil {
			return diag.Errorf("failed to create filter: %q", err)
		}

		filters = append(filters, newFilters...)
	}

	labelsFilterProp, labelFiltersSpecified := d.GetOk(filterLabelsPropName)
	if labelFiltersSpecified {
		newFilter, err := createLabelFilterFunc(ctx, labelsFilterProp)
		if err != nil {
			return diag.Errorf("failed to create filter: %q", err)
		}

		filters = append(filters, newFilter)
	}

	for _, item := range instances {
		// we use ID to generate a resource ID, we cannot list instances without ID.
		if item.ID == nil {
			continue
		}

		ids = append(ids, *item.ID)

		instance, err := client.FindInstance(
			ctx,
			zone,
			*item.ID,
		)
		if err != nil {
			return diag.FromErr(err)
		}

		instanceData, err := dataSourceComputeInstanceBuildData(instance)
		if err != nil {
			return diag.FromErr(err)
		}

		rdns, err := client.GetInstanceReverseDNS(ctx, zone, *instance.ID)
		if err != nil && !errors.Is(err, exoapi.ErrNotFound) {
			return diag.Errorf("unable to retrieve instance reverse-dns: %s", err)
		}
		instanceData[dsComputeInstanceAttrReverseDNS] = rdns

		if instance.InstanceTypeID != nil {
			tid := *instance.InstanceTypeID
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

			instanceData[dsComputeInstanceAttrType] = instanceTypes[tid]
		}

		if len(filters) > 0 && !checkForMatch(instanceData, filters) {
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
		"id": resourceIDString(d, "exoscale_compute_instance_list"),
	})

	return nil
}
