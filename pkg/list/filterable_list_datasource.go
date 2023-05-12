package list

import (
	"context"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/exoscale/terraform-provider-exoscale/pkg/filter"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
)

const (
	ZoneAttributeIdentifier = "zone"
)

func FilterableListDataSource[T any](
	dataSourceIdentifier, listAttributeIdentifier, zoneAttribute string,
	getList getListFunc[T],
	toTFObj toTerraformObjectFunc[T],
	generateListID generateListIDFunc[T],
	getScheme func() general.SchemaMap) *schema.Resource {
	elemScheme := getScheme()
	ret := &schema.Resource{
		Schema: map[string]*schema.Schema{
			ZoneAttributeIdentifier: {
				Type:     schema.TypeString,
				Required: true,
			},
			listAttributeIdentifier: {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: elemScheme,
				},
			},
		},

		ReadContext: createDataSourceReadFunc(
			dataSourceIdentifier, listAttributeIdentifier,
			getList,
			toTFObj,
			generateListID,
			elemScheme),
	}

	filter.AddFilterAttributes(ret, elemScheme)

	return ret
}

type getListFunc[T any] func(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*T, error)

type generateListIDFunc[T any] func([]*T) string

type toTerraformObjectFunc[T any] func(*T) general.TerraformObject

func createDataSourceReadFunc[T any](
	dataSourceIdentifier, listAttributeIdentifier string,
	getList getListFunc[T],
	toTFObj toTerraformObjectFunc[T],
	generateListID generateListIDFunc[T],
	elemScheme general.SchemaMap) func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	return func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
		tflog.Debug(ctx, "beginning read", map[string]interface{}{
			"id": general.ResourceIDString(d, dataSourceIdentifier),
		})

		zone := d.Get(ZoneAttributeIdentifier).(string)

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
			clusterData := toTFObj(cluster)
			clusterData[ZoneAttributeIdentifier] = zone

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
			"id": general.ResourceIDString(d, dataSourceIdentifier),
		})

		return nil
	}
}
