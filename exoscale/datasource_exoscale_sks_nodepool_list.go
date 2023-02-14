package exoscale

import (
	"context"
	"crypto/md5"
	"fmt"
	"sort"
	"strings"

	v2 "github.com/exoscale/egoscale/v2"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	dsSKSNodepoolsListIdentifier          = "exoscale_sks_nodepool_list"
	dsSKSNodepoolsListAttributeIdentifier = "nodepools"
)

func dataSourceSKSNodepoolListGetElementScheme() schemaMap {
	return dataSourceSKSNodepool().Schema
}

func dataSourceSKSNodepoolList() *schema.Resource {
	return filterableListDataSource(dsSKSNodepoolsListIdentifier, dsSKSNodepoolsListAttributeIdentifier, resSKSNodepoolAttrZone, getNodepoolList, nodepoolToDataMap, generateSKSNodepoolListID, dataSourceSKSNodepoolListGetElementScheme)
}

func generateSKSNodepoolListID(nodepools []*v2.SKSNodepool) string {
	ids := make([]string, 0, len(nodepools))

	for _, cluster := range nodepools {
		ids = append(ids, *cluster.ID)
	}

	sort.Strings(ids)

	return fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(ids, ""))))
}

func getNodepoolList(ctx context.Context, d *schema.ResourceData, meta interface{}) ([]*v2.SKSNodepool, error) {
	clusters, err := getClusterList(ctx, d, meta)
	if err != nil {
		return nil, err
	}

	var nodepools []*v2.SKSNodepool

	for _, cluster := range clusters {
		for _, nodepool := range cluster.Nodepools {
			// TODO use a library method
			nodepools = append(nodepools, nodepool)
		}
	}

	return nodepools, nil
}
