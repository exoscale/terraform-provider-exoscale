package exoscale

import (
	"context"
	"crypto/md5"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
	"github.com/exoscale/terraform-provider-exoscale/pkg/list"
)

const (
	dsSKSClustersListIdentifier = "exoscale_sks_cluster_list"
	dsSKSClustersListClusters   = "clusters"
)

func dataSourceSKSClusterListGetElementScheme() general.SchemaMap {
	return dataSourceSKSCluster().Schema
}

func dataSourceSKSClusterList() *schema.Resource {
	return list.FilterableListDataSource(dsSKSClustersListIdentifier, dsSKSClustersListClusters, resSKSClusterAttrZone, getClusterList, clusterToDataMap, generateSKSClusterListID, dataSourceSKSClusterListGetElementScheme)
}

func generateSKSClusterListID(clusters []*v3.SKSCluster) string {
	ids := make([]string, 0, len(clusters))

	for _, cluster := range clusters {
		ids = append(ids, cluster.ID.String())
	}

	sort.Strings(ids)

	return fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(ids, ""))))
}

func getClusterList(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*v3.SKSCluster, error) {
	zone := d.Get(resSKSClusterAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancel()

	client, err := config.GetClientV3WithZone(ctx, meta, zone)
	if err != nil {
		return nil, fmt.Errorf("error getting client for zone %q: %s", zone, err)
	}

	resp, err := client.ListSKSClusters(ctx)
	if err != nil {
		return nil, fmt.Errorf("error getting cluster list from zone %q: %s", zone, err)
	}

	clusters := make([]*v3.SKSCluster, len(resp.SKSClusters))
	for i := range resp.SKSClusters {
		clusters[i] = &resp.SKSClusters[i]
	}

	return clusters, nil
}
