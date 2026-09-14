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
	dsSKSNodepoolsListIdentifier          = "exoscale_sks_nodepool_list"
	dsSKSNodepoolsListAttributeIdentifier = "nodepools"
)

// dataSourceSKSNodepoolListGetElementScheme reproduces the schema that the
// (now framework-based) `exoscale_sks_nodepool` data source used to expose, so
// the `exoscale_sks_nodepool_list` data source keeps behaving identically until
// it is migrated in turn. See exoscale/sks_nodepool_sdkv2.go.
func dataSourceSKSNodepoolListGetElementScheme() general.SchemaMap {
	ret := &schema.Resource{
		Schema: map[string]*schema.Schema{
			resSKSNodepoolAttrZone: {
				Type:     schema.TypeString,
				Required: true,
			},
			dsSKSNodepoolID: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{resSKSNodepoolAttrName},
			},
			resSKSNodepoolAttrName: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{dsSKSNodepoolID},
			},
			resSKSNodepoolAttrClusterID: {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}

	general.AddAttributes(ret, sksNodepoolResourceSchema())

	return ret.Schema
}

func dataSourceSKSNodepoolList() *schema.Resource {
	return list.FilterableListDataSource(dsSKSNodepoolsListIdentifier, dsSKSNodepoolsListAttributeIdentifier, resSKSNodepoolAttrZone, getNodepoolList, nodepoolToDataMap, generateSKSNodepoolListID, dataSourceSKSNodepoolListGetElementScheme)
}

func generateSKSNodepoolListID(nodepools []*v3.SKSNodepool) string {
	ids := make([]string, 0, len(nodepools))

	for _, np := range nodepools {
		ids = append(ids, np.ID.String())
	}

	sort.Strings(ids)

	return fmt.Sprintf("%x", md5.Sum([]byte(strings.Join(ids, ""))))
}

func getNodepoolList(ctx context.Context, d *schema.ResourceData, meta any) ([]*v3.SKSNodepool, error) {
	zone := d.Get(resSKSNodepoolAttrZone).(string)

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

	var nodepools []*v3.SKSNodepool
	for ci := range resp.SKSClusters {
		cluster := &resp.SKSClusters[ci]
		for ni := range cluster.Nodepools {
			nodepools = append(nodepools, &cluster.Nodepools[ni])
		}
	}

	return nodepools, nil
}
