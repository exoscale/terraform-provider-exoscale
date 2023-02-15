package exoscale

import (
	"context"

	v2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	dsSKSClusterID = "id"
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

	// TODO do we really need all attributes?
	for attributeIdentifier, attributeValue := range resourceSKSCluster().Schema {
		_, attributeAlreadySet := ret.Schema[attributeIdentifier]
		if !attributeAlreadySet {
			ret.Schema[attributeIdentifier] = attributeValue
		}
	}

	return ret
}

type terraformObject = map[string]interface{}

func clusterToDataMap(cluster *v2.SKSCluster) terraformObject {
	ret := make(terraformObject)

	assign(ret, resSKSClusterAttrAddons, cluster.AddOns)
	assign(ret, resSKSClusterAttrAutoUpgrade, cluster.AutoUpgrade)
	assign(ret, resSKSClusterAttrCNI, cluster.CNI)
	assignTime(ret, resSKSClusterAttrCreatedAt, cluster.CreatedAt)
	assign(ret, resSKSClusterAttrDescription, cluster.Description)
	assign(ret, resSKSClusterAttrEndpoint, cluster.Endpoint)
	assign(ret, resSKSClusterAttrLabels, cluster.Labels)
	assign(ret, resSKSClusterAttrName, cluster.Name)
	assign(ret, dsSKSClusterID, cluster.ID)

	nodepools := make([]string, len(cluster.Nodepools))
	for i, nodepool := range cluster.Nodepools {
		nodepools[i] = *nodepool.ID
	}
	assign(ret, resSKSClusterAttrNodepools, &nodepools)

	assign(ret, resSKSClusterAttrServiceLevel, cluster.ServiceLevel)
	assign(ret, resSKSClusterAttrState, cluster.State)
	assign(ret, resSKSClusterAttrVersion, cluster.Version)

	return ret
}

func applyClusterDataToDataSource(data terraformObject, d *schema.ResourceData, schema map[string]*schema.Schema) error {
	for attrIdentifier, attrVal := range data {
		_, hasAttribute := schema[attrIdentifier]
		if hasAttribute {
			if err := d.Set(attrIdentifier, attrVal); err != nil {
				return err
			}
		}
	}

	return nil
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
	if err := applyClusterDataToDataSource(clusterData, d, dataSourceSKSCluster().Schema); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
