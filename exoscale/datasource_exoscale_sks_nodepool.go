package exoscale

import (
	"context"
	"fmt"

	v2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/terraform-provider-exoscale/pkg/filter"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	dsSKSNodepoolID = "id"
)

func dataSourceSKSNodepool() *schema.Resource {
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

		ReadContext: dataSourceSKSNodepoolRead,
	}

	// TODO do we really need all attributes?
	// TODO generalize
	for attributeIdentifier, attributeValue := range resourceSKSNodepool().Schema {
		_, attributeAlreadySet := ret.Schema[attributeIdentifier]
		if !attributeAlreadySet {
			newSchema := &schema.Schema{}
			*newSchema = *attributeValue
			newSchema.Required = false
			newSchema.Optional = true
			newSchema.Default = nil

			ret.Schema[attributeIdentifier] = newSchema
		}
	}

	return ret
}

func nodepoolToDataMap(nodepool *v2.SKSNodepool) tfData {
	ret := make(tfData)

	// TODO
	// ret[resSKSNodepoolAttrAntiAffinityGroupIDs] = nodepool.AntiAffinityGroupIDs
	// ret[resSKSNodepoolAttrCreatedAt] = nodepool.CreatedAt
	ret[resSKSNodepoolAttrDeployTargetID] = nodepool.DeployTargetID
	ret[resSKSNodepoolAttrDescription] = nodepool.Description
	ret[resSKSNodepoolAttrDiskSize] = nodepool.DiskSize
	ret[resSKSNodepoolAttrInstancePoolID] = nodepool.InstancePoolID
	ret[resSKSNodepoolAttrInstancePrefix] = nodepool.InstancePrefix
	ret[resSKSNodepoolAttrInstanceType] = nodepool.InstanceTypeID
	ret[resSKSNodepoolAttrLabels] = nodepool.Labels
	ret[resSKSNodepoolAttrName] = nodepool.Name
	// TODO
	// ret[resSKSNodepoolAttrPrivateNetworkIDs] = nodepool.PrivateNetworkIDs
	// ret[resSKSNodepoolAttrSecurityGroupIDs] = nodepool.SecurityGroupIDs
	ret[resSKSNodepoolAttrSize] = nodepool.Size
	ret[resSKSNodepoolAttrState] = nodepool.State
	ret[resSKSNodepoolAttrTaints] = nodepool.Taints
	ret[resSKSNodepoolAttrTemplateID] = nodepool.TemplateID
	ret[resSKSNodepoolAttrVersion] = nodepool.Version
	ret[dsSKSNodepoolID] = nodepool.ID

	return ret
}

func dataSourceSKSNodepoolRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]interface{}{
		"id": resourceSKSNodepoolIDString(d),
	})

	zone := d.Get(resSKSClusterAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	ctx = exoapi.WithEndpoint(ctx, exoapi.NewReqEndpoint(getEnvironment(meta), zone))
	defer cancel()

	client := GetComputeClient(meta)

	clusterID := d.Get(resSKSNodepoolAttrClusterID).(string)
	cluster, err := client.GetSKSCluster(ctx, zone, clusterID)
	if err != nil {
		return diag.Errorf("error getting cluster %q: %s", clusterID, err)
	}

	filters, err := filter.CreateFilters(ctx, d, dataSourceSKSNodepool().Schema)
	if err != nil {
		return diag.Errorf("failed to create filter: %q", err)
	}

	var matchingNodePool tfData
	nMatches := 0

	for _, nodepool := range cluster.Nodepools {
		nodepoolData := nodepoolToDataMap(nodepool)
		nodepoolData[resSKSNodepoolAttrClusterID] = clusterID
		nodepoolData[resSKSNodepoolAttrZone] = zone
		if filter.CheckForMatch(nodepoolData, filters) {
			if nMatches < 1 {
				d.SetId(*nodepool.ID)

				matchingNodePool = nodepoolData
			} else {
				tflog.Info(ctx, fmt.Sprintf("nodepool %q matches multiple nodepools, this shouldn't be possible", clusterID))
			}
		}
	}

	if nMatches < 0 {
		nodepoolID, _ := d.GetOk(dsSKSNodepoolID)
		nodepoolName, _ := d.GetOk(resSKSNodepoolAttrName)
		return diag.Errorf("no nodepool matches cluster %q with name %q or id %q", clusterID, nodepoolName, nodepoolID)
	}

	if err := applyClusterDataToDataSource(matchingNodePool, d, dataSourceSKSNodepool().Schema); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
