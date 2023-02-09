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
	filterStringPropName = "filter_string"
	filterRegexPropName  = "filter_regex"
	filterLabelsPropName = "labels"
	attributePropName    = "attribute"
	keyPropName          = "key"
	valuePropName        = "value"
)

type matchStringFunc = func(given string) bool

type createStringMatchFunc = func(filterValue string) (matchStringFunc, error)

func createStringFilterFuncs(stringFilterProp interface{}, createMatch createStringMatchFunc) ([]filterFunc, error) {
	set := stringFilterProp.(*schema.Set)

	var filters []filterFunc

	for _, v := range set.List() {
		m := v.(map[string]interface{})

		match, err := createMatch(m[valuePropName].(string))
		if err != nil {
			return nil, err
		}

		filters = append(filters, createStrFilterFunc(m[attributePropName].(string), match))
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
				valuePropName: {
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
			filterRegexPropName:  filterStringSchema(),
			filterLabelsPropName: filterLabelsSchema(),
		},

		ReadContext: dataSourceComputeInstanceListRead,
	}
}

type filterFunc = func(map[string]interface{}) bool

func matchExact(expected string) (matchStringFunc, error) {
	return func(given string) bool {
		return given == expected
	}, nil
}

func matchRegex(expectedRegex string) (matchStringFunc, error) {
	r, err := regexp.Compile(expectedRegex)
	if err != nil {
		return nil, err
	}

	return func(given string) bool {
		return r.MatchString(given)
	}, nil
}

func createStrFilterFunc(filterAttribute string, match matchStringFunc) filterFunc {
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

func createLabelFilterFunc(ctx context.Context, labelsFilterProp interface{}) filterFunc {
	labelsFilter := make(map[string]string)
	labels := labelsFilterProp.(map[string]interface{})
	for k, v := range labels {
		labelsFilter[k] = v.(string)
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

		for filterKey, filterValue := range labelsFilter {
			value, ok := labels[filterKey]
			if !ok || value != filterValue {
				return false
			}
		}

		return true
	}
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
		newFilters, err := createStringFilterFuncs(strFilterProp, matchExact)
		if err != nil {
			return diag.Errorf("failed to create filter: %w", err)
		}

		filters = append(filters, newFilters...)
	}

	regexFilterProp, regexFiltersSpecified := d.GetOk(filterRegexPropName)
	if regexFiltersSpecified {
		newFilters, err := createStringFilterFuncs(regexFilterProp, matchRegex)
		if err != nil {
			return diag.Errorf("failed to create filter: %w", err)
		}

		filters = append(filters, newFilters...)
	}

	labelsFilterProp, labelFiltersSpecified := d.GetOk(filterLabelsPropName)
	if labelFiltersSpecified {
		filters = append(filters, createLabelFilterFunc(ctx, labelsFilterProp))
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

		if !checkForMatch(instanceData, filters) {
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
