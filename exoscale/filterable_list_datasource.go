package exoscale

import (
	"context"

	"github.com/exoscale/terraform-provider-exoscale/pkg/filter"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type schemaMap map[string]*schema.Schema

func filterableListDataSource[T any](
	dataSourceIdentifier, listAttributeIdentifier, zoneAttribute string,
	getList getListFunc[T],
	toDataMap toDataMapFunc[T],
	generateListID generateListIDFunc[T],
	getScheme func() schemaMap) *schema.Resource {
	// TODO make zone required
	elemScheme := getScheme()
	ret := &schema.Resource{
		Schema: map[string]*schema.Schema{
			listAttributeIdentifier: {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: elemScheme,
				},
			},
		},

		ReadContext: createDataSourceReadFunc(
			dataSourceIdentifier, listAttributeIdentifier, zoneAttribute,
			getList,
			toDataMap,
			generateListID,
			elemScheme),
	}

	filter.AddFilterAttributes(ret, elemScheme)

	return ret
}

type getListFunc[T any] func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*T, error)

type generateListIDFunc[T any] func([]*T) string

type toDataMapFunc[T any] func(*T) terraformObject

func createDataSourceReadFunc[T any](
	dataSourceIdentifier, listAttributeIdentifier, zoneAttribute string,
	getList getListFunc[T],
	toDataMap toDataMapFunc[T],
	generateListID generateListIDFunc[T],
	elemScheme schemaMap) func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		tflog.Debug(ctx, "beginning read", map[string]interface{}{
			"id": resourceIDString(d, dataSourceIdentifier),
		})

		// TODO
		zone := d.Get(zoneAttribute).(string)

		clusters, err := getList(ctx, d, meta)
		if err != nil {
			return diag.FromErr(err)
		}

		filters, err := filter.CreateFilters(ctx, d, elemScheme)
		if err != nil {
			return diag.Errorf("failed to create filter: %q", err)
		}

		data := make([]interface{}, 0, len(clusters))
		for _, cluster := range clusters {
			clusterData := toDataMap(cluster)
			clusterData[zoneAttribute] = zone

			if !filter.CheckForMatch(clusterData, filters) {
				continue
			}

			data = append(data, clusterData)
		}

		d.SetId(generateListID(clusters))

		err = d.Set(listAttributeIdentifier, data)
		if err != nil {
			return diag.FromErr(err)
		}

		tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
			"id": resourceIDString(d, dataSourceIdentifier),
		})

		return nil
	}
}
