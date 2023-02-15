package exoscale

import (
	"context"
	"fmt"

	v2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/terraform-provider-exoscale/pkg/filter"
	"github.com/exoscale/terraform-provider-exoscale/pkg/gen/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const (
	dsSKSNodepoolIdentifier = "exoscale_sks_nodepool"
	dsSKSNodepoolID         = "id"
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

	datasource.AddAttributes(ret, resourceSKSNodepool().Schema)

	return ret
}

func nodepoolToDataMap(nodepool *v2.SKSNodepool) datasource.TerraformObject {
	ret := make(datasource.TerraformObject)

	datasource.Assign(ret, resSKSNodepoolAttrAntiAffinityGroupIDs, nodepool.AntiAffinityGroupIDs)
	datasource.AssignTime(ret, resSKSNodepoolAttrCreatedAt, nodepool.CreatedAt)
	datasource.Assign(ret, resSKSNodepoolAttrDeployTargetID, nodepool.DeployTargetID)
	datasource.Assign(ret, resSKSNodepoolAttrDescription, nodepool.Description)
	datasource.Assign(ret, resSKSNodepoolAttrDiskSize, nodepool.DiskSize)
	datasource.Assign(ret, resSKSNodepoolAttrInstancePoolID, nodepool.InstancePoolID)
	datasource.Assign(ret, resSKSNodepoolAttrInstancePrefix, nodepool.InstancePrefix)
	datasource.Assign(ret, resSKSNodepoolAttrInstanceType, nodepool.InstanceTypeID)
	datasource.Assign(ret, resSKSNodepoolAttrLabels, nodepool.Labels)
	datasource.Assign(ret, resSKSNodepoolAttrName, nodepool.Name)
	datasource.Assign(ret, resSKSNodepoolAttrPrivateNetworkIDs, nodepool.PrivateNetworkIDs)
	datasource.Assign(ret, resSKSNodepoolAttrSecurityGroupIDs, nodepool.SecurityGroupIDs)
	datasource.Assign(ret, resSKSNodepoolAttrSize, nodepool.Size)
	datasource.Assign(ret, resSKSNodepoolAttrState, nodepool.State)
	datasource.Assign(ret, resSKSNodepoolAttrTaints, nodepool.Taints)
	datasource.Assign(ret, resSKSNodepoolAttrTemplateID, nodepool.TemplateID)
	datasource.Assign(ret, resSKSNodepoolAttrVersion, nodepool.Version)
	datasource.Assign(ret, dsSKSNodepoolID, nodepool.ID)

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

	var matchingNodePool datasource.TerraformObject
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

	if err := datasource.Apply(matchingNodePool, d, dataSourceSKSNodepool().Schema); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
