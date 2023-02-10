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

func optionalAttribute(typ schema.ValueType) *schema.Schema {
	return &schema.Schema{
		Type:     typ,
		Optional: true,
	}
}

func optionalMapOfStrToStrAtribute() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Elem:     &schema.Schema{Type: schema.TypeString},
		Optional: true,
	}
}

func dataSourceComputeInstanceList() *schema.Resource {
	ret := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"instances": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: getDataSourceComputeInstanceSchema(),
				},
			},
		},

		ReadContext: dataSourceComputeInstanceListRead,
	}

	for attrIdentifier, attrSpec := range getDataSourceComputeInstanceSchema() {
		switch attrSpec.Type {
		case schema.TypeBool, schema.TypeInt, schema.TypeString:
			ret.Schema[attrIdentifier] = optionalAttribute(attrSpec.Type)
		case schema.TypeMap:
			elem, ok := attrSpec.Elem.(*schema.Schema)
			if ok && elem.Type == schema.TypeString {
				ret.Schema[attrIdentifier] = optionalMapOfStrToStrAtribute()
			}
		}
	}

	return ret
}

type filterFunc = func(map[string]interface{}) bool

func createEqualityFilter[T comparable](argIdentifier string, expected T) (filterFunc, error) {
	return func(data map[string]interface{}) bool {
		attr, ok := data[argIdentifier]
		if !ok {
			return false
		}

		switch v := attr.(type) {
		case T:
			if v == expected {
				return true
			}
		case *T:
			if *v == expected {
				return true
			}
		}

		return false
	}, nil
}

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

func createMapStrToStrFilterFunc(ctx context.Context, filterProp interface{}) (filterFunc, error) {
	filters := make(map[string]matchStringFunc)
	maps := filterProp.(map[string]interface{})
	for k, v := range maps {
		filter, err := createMatchStringFunc(v.(string))
		if err != nil {
			return nil, err
		}

		filters[k] = filter
	}

	return func(data map[string]interface{}) bool {
		mapAttr, ok := data["labels"]
		if !ok {
			return false
		}

		mapToFilter, isMap := mapAttr.(map[string]string)
		if !isMap {
			tflog.Info(ctx, fmt.Sprintf("attribute of compute instance has unexpected type %T for labels", mapAttr))

			return false
		}

		for filterKey, filterValue := range filters {
			value, ok := mapToFilter[filterKey]
			if !ok || !filterValue(value) {
				return false
			}
		}

		return true
	}, nil
}

func checkForMatch(data map[string]interface{}, filters []filterFunc) bool {
	for _, filter := range filters {
		if !filter(data) {
			return false
		}
	}

	return true
}

func createStringFilter(argIdentifier, expected string) (filterFunc, error) {
	matchFn, err := createMatchStringFunc(expected)
	if err != nil {
		return nil, err
	}

	return createStringFilterFunc(argIdentifier, matchFn), nil
}

func createFilters(ctx context.Context, d *schema.ResourceData) ([]filterFunc, error) {
	var filters []filterFunc

	for argIdentifier, argSpec := range getDataSourceComputeInstanceSchema() {
		argValue, ok := d.GetOk(argIdentifier)
		if !ok {
			continue
		}

		switch argSpec.Type {
		case schema.TypeBool:
			newFilterFunc, err := createEqualityFilter(argIdentifier, argValue.(bool))
			if err != nil {
				return nil, err
			}

			filters = append(filters, newFilterFunc)
		case schema.TypeInt:
			newFilterFunc, err := createEqualityFilter(argIdentifier, int64(argValue.(int)))
			if err != nil {
				return nil, err
			}

			filters = append(filters, newFilterFunc)
		case schema.TypeString:
			newFilterFunc, err := createStringFilter(argIdentifier, argValue.(string))
			if err != nil {
				return nil, err
			}

			filters = append(filters, newFilterFunc)
		case schema.TypeMap:
			newFilter, err := createMapStrToStrFilterFunc(ctx, argValue)
			if err != nil {
				return nil, err
			}

			filters = append(filters, newFilter)
		default:
			continue
		}
	}

	return filters, nil
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

	filters, err := createFilters(ctx, d)
	if err != nil {
		return diag.Errorf("failed to create filter: %q", err)
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
