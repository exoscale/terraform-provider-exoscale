package exoscale

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
)

const (
	filterStringPropName = "filter_string"
	filterLabelsPropName = "filter_labels"
	attributePropName    = "attribute"
	keyPropName          = "key"
	valuePropName        = "value"
)

type stringFilter struct {
	Attribute string
	Value     string
}

type labelFilter struct {
	Key   string
	Value string
}

func buildStringFilters(set *schema.Set) []stringFilter {
	var filters []stringFilter

	for _, v := range set.List() {
		m := v.(map[string]interface{})

		filters = append(filters,
			stringFilter{
				Attribute: m[attributePropName].(string),
				Value:     m[valuePropName].(string),
			},
		)
	}

	return filters
}

func buildLabelFilters(set *schema.Set) []labelFilter {
	var filters []labelFilter

	for _, v := range set.List() {
		m := v.(map[string]interface{})

		filters = append(filters,
			labelFilter{
				Key:   m[keyPropName].(string),
				Value: m[valuePropName].(string),
			},
		)
	}

	return filters
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

	var strFilters []stringFilter
	strFilterProp, stringFiltersSpecified := d.GetOk(filterStringPropName)
	if stringFiltersSpecified {
		strFilters = buildStringFilters(strFilterProp.(*schema.Set))
	}

	var labelsFilters []labelFilter
	labelsFilterProp, labelFiltersSpecified := d.GetOk(filterLabelsPropName)
	if labelFiltersSpecified {
		labelsFilters = buildLabelFilters(labelsFilterProp.(*schema.Set))
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

		matched := false

		if stringFiltersSpecified {
			for _, filter := range strFilters {
				instanceAttr, ok := instanceData[filter.Attribute]
				if !ok {
					continue
				}

				switch v := instanceAttr.(type) {
				case string:
					if v == filter.Value {
						matched = true
					}
				case *string:
					if *v == filter.Value {
						matched = true
					}
				}
			}
		}

		if !matched && labelFiltersSpecified {
			labelsAttr, ok := instanceData["labels"]
			if !ok {
				continue
			}

			labels, isMap := labelsAttr.(map[string]string)
			if !isMap {
				tflog.Info(ctx, fmt.Sprintf("attribute of compute instance has unexpected type %T for labels", labelsAttr))

				continue
			}

			for _, filter := range labelsFilters {
				value, ok := labels[filter.Key]
				if !ok {
					continue
				}

				if value == filter.Value {
					matched = true
				}
			}
		}

		if !matched {
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

func pp(ctx context.Context, msg string) {
	tflog.Info(ctx, msg)
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
		Type:     schema.TypeSet,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				keyPropName: {
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
