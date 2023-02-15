package exoscale

import (
	"context"

	v2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/terraform-provider-exoscale/pkg/gen/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	dsSKSClusterIdentifier = "exoscale_sks_cluster"
	dsSKSClusterID         = "id"
)

func dataSourceSKSCluster() *schema.Resource {
	ret := &schema.Resource{
		Schema: map[string]*schema.Schema{
			resSKSClusterAttrZone: {
				Type:     schema.TypeString,
				Required: true,
			},
			resSKSClusterAttrName: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{dsSKSClusterID},
			},
			dsSKSClusterID: {
				Type:         schema.TypeString,
				Optional:     true,
				ExactlyOneOf: []string{resSKSClusterAttrName},
			},
		},

		ReadContext: dataSourceSKSClusterRead,
	}

	datasource.AddAttributes(ret, resourceSKSCluster().Schema)

	return ret
}

func clusterToDataMap(cluster *v2.SKSCluster) datasource.TerraformObject {
	ret := make(datasource.TerraformObject)

	datasource.Assign(ret, resSKSClusterAttrAddons, cluster.AddOns)
	datasource.Assign(ret, resSKSClusterAttrAutoUpgrade, cluster.AutoUpgrade)
	datasource.Assign(ret, resSKSClusterAttrCNI, cluster.CNI)
	datasource.AssignTime(ret, resSKSClusterAttrCreatedAt, cluster.CreatedAt)
	datasource.Assign(ret, resSKSClusterAttrDescription, cluster.Description)
	datasource.Assign(ret, resSKSClusterAttrEndpoint, cluster.Endpoint)
	datasource.Assign(ret, resSKSClusterAttrLabels, cluster.Labels)
	datasource.Assign(ret, resSKSClusterAttrName, cluster.Name)
	datasource.Assign(ret, dsSKSClusterID, cluster.ID)

	nodepools := make([]string, len(cluster.Nodepools))
	for i, nodepool := range cluster.Nodepools {
		nodepools[i] = *nodepool.ID
	}
	datasource.Assign(ret, resSKSClusterAttrNodepools, &nodepools)

	datasource.Assign(ret, resSKSClusterAttrServiceLevel, cluster.ServiceLevel)
	datasource.Assign(ret, resSKSClusterAttrState, cluster.State)
	datasource.Assign(ret, resSKSClusterAttrVersion, cluster.Version)

	return ret
}

func dataSourceSKSClusterRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceSKSClusterIDString(d),
	})

	zone := d.Get(resSKSClusterAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	clusterID, searchByClusterID := d.GetOk(dsSKSClusterID)
	clusterName, searchByClusterName := d.GetOk(resSKSClusterAttrName)

	var cluster *v2.SKSCluster
	switch {
	case searchByClusterID:
		clusterIDStr := clusterID.(string)

		var err error
		if cluster, err = client.GetSKSCluster(ctx, zone, clusterIDStr); err != nil {
			return diag.Errorf("error getting cluster %q: %s", clusterIDStr, err)
		}
	case searchByClusterName:
		clusterNameStr := clusterName.(string)

		var err error
		if cluster, err = client.FindSKSCluster(ctx, zone, clusterNameStr); err != nil {
			return diag.Errorf("error getting cluster %q: %s", clusterNameStr, err)
		}
	default:
		return diag.Errorf(
			"one of %s or %s must be specified",
			dsSKSClusterID,
			resSKSClusterAttrName,
		)
	}

	d.SetId(*cluster.ID)

	clusterData := clusterToDataMap(cluster)
	if err := datasource.Apply(clusterData, d, dataSourceSKSCluster().Schema); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
