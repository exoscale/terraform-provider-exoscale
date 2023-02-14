package exoscale

import (
	"context"
	"crypto/md5"
	"fmt"
	"sort"
	"strings"

	v2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/terraform-provider-exoscale/pkg/filter"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	dsSKSClustersListIdentifier = "exoscale_sks_cluster_list"
	dsSKSClustersListClusters   = "clusters"
)

func dataSourceSKSClusterListGetElementScheme() map[string]*schema.Schema {
	return dataSourceSKSCluster().Schema
}

func dataSourceSKSClusterList() *schema.Resource {
	// TODO make zone required
	elemSchema := dataSourceSKSClusterListGetElementScheme()
	ret := &schema.Resource{
		Schema: map[string]*schema.Schema{
			dsSKSClustersListClusters: {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: elemSchema,
				},
			},
		},

		ReadContext: dataSourceSKSClusterListRead,
	}

	filter.AddFilterAttributes(ret, elemSchema)

	return ret
}

func generateSKSClusterListID(clusters []*v2.SKSCluster) string {
	ids := make([]string, 0, len(clusters))

	for _, cluster := range clusters {
		ids = append(ids, *cluster.ID)
	}

	sort.Strings(ids)

	return fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(ids, ""))))
}

func dataSourceSKSClusterListRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceIDString(d, dsSKSClustersListIdentifier),
	})

	zone := d.Get(resSKSClusterAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	clusters, err := client.ListSKSClusters(ctx, zone)
	if err != nil {
		return diag.Errorf("error getting cluster list from zone %q: %s", zone, err)
	}

	filters, err := filter.CreateFilters(ctx, d, getDataSourceComputeInstanceSchema())
	if err != nil {
		return diag.Errorf("failed to create filter: %q", err)
	}

	data := make([]interface{}, 0, len(clusters))
	for _, cluster := range clusters {
		clusterData := clusterToDataMap(cluster)
		clusterData[resSKSClusterAttrZone] = zone

		if !filter.CheckForMatch(clusterData, filters) {
			continue
		}

		data = append(data, clusterData)
	}

	d.SetId(generateSKSClusterListID(clusters))

	err = d.Set(dsSKSClustersListClusters, data)
	if err != nil {
		return diag.FromErr(err)
	}

	tflog.Debug(ctx, "read finished successfully", map[string]interface{}{
		"id": resourceIDString(d, "exoscale_compute_instance_list"),
	})

	return nil
}
