package exoscale

import (
	"context"
	"crypto/md5"
	"fmt"
	"sort"
	"strings"

	v2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
	"github.com/exoscale/terraform-provider-exoscale/pkg/list"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

func generateSKSClusterListID(clusters []*v2.SKSCluster) string {
	ids := make([]string, 0, len(clusters))

	for _, cluster := range clusters {
		ids = append(ids, *cluster.ID)
	}

	sort.Strings(ids)

	return fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(ids, ""))))
}

func getClusterList(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*v2.SKSCluster, error) {
	zone := d.Get(resSKSClusterAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	clusters, err := client.ListSKSClusters(ctx, zone)
	if err != nil {
		return nil, fmt.Errorf("error getting cluster list from zone %q: %s", zone, err)
	}

	return clusters, nil
}
