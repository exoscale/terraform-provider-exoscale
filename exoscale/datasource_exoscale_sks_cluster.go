package exoscale

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
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

	general.AddAttributes(ret, resourceSKSCluster().Schema)

	return ret
}

func clusterToDataMap(cluster *v3.SKSCluster) general.TerraformObject {
	ret := make(general.TerraformObject)

	ret[resSKSClusterAttrAddons] = cluster.Addons
	ret[resSKSClusterAttrCNI] = string(cluster.Cni)
	ret[resSKSClusterAttrCreatedAt] = cluster.CreatedAT.Format(time.RFC3339)
	ret[resSKSClusterAttrDescription] = cluster.Description
	ret[resSKSClusterAttrEndpoint] = cluster.Endpoint
	ret[resSKSClusterAttrFeatureGates] = cluster.FeatureGates
	ret[resSKSClusterAttrLabels] = map[string]string(cluster.Labels)
	ret[resSKSClusterAttrName] = cluster.Name
	ret[dsSKSClusterID] = cluster.ID.String()
	ret[resSKSClusterAttrServiceLevel] = string(cluster.Level)
	ret[resSKSClusterAttrState] = string(cluster.State)
	ret[resSKSClusterAttrVersion] = cluster.Version

	if cluster.AutoUpgrade != nil {
		ret[resSKSClusterAttrAutoUpgrade] = *cluster.AutoUpgrade
	}
	if cluster.DefaultSecurityGroupID != nil {
		ret[resSKSClusterAttrDefaultSecurityGroupID] = cluster.DefaultSecurityGroupID.String()
	}
	if cluster.EnableKubeProxy != nil {
		ret[resSKSClusterAttrEnableKubeProxy] = *cluster.EnableKubeProxy
	}

	nodepools := make([]string, len(cluster.Nodepools))
	for i, np := range cluster.Nodepools {
		nodepools[i] = np.ID.String()
	}
	ret[resSKSClusterAttrNodepools] = nodepools

	return ret
}

func dataSourceSKSClusterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]any{
		"id": resourceSKSClusterIDString(d),
	})

	zone := d.Get(resSKSClusterAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancel()

	client, err := config.GetClientV3WithZone(ctx, meta, zone)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID, searchByClusterID := d.GetOk(dsSKSClusterID)
	clusterName, searchByClusterName := d.GetOk(resSKSClusterAttrName)

	var cluster *v3.SKSCluster
	switch {
	case searchByClusterID:
		clusterIDStr := clusterID.(string)

		cluster, err = client.GetSKSCluster(ctx, v3.UUID(clusterIDStr))
		if err != nil {
			return diag.Errorf("error getting cluster %q: %s", clusterIDStr, err)
		}
	case searchByClusterName:
		clusterNameStr := clusterName.(string)

		clusters, err := client.ListSKSClusters(ctx)
		if err != nil {
			return diag.Errorf("error listing clusters: %s", err)
		}
		found, err := clusters.FindSKSCluster(clusterNameStr)
		if err != nil {
			return diag.Errorf("error finding cluster %q: %s", clusterNameStr, err)
		}
		cluster = &found
	default:
		return diag.Errorf(
			"one of %s or %s must be specified",
			dsSKSClusterID,
			resSKSClusterAttrName,
		)
	}

	d.SetId(cluster.ID.String())

	if err := general.Apply(clusterToDataMap(cluster), d, dataSourceSKSCluster().Schema); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
