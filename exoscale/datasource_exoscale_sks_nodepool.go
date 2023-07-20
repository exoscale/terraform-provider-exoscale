package exoscale

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	v2 "github.com/exoscale/egoscale/v2"
	exoapi "github.com/exoscale/egoscale/v2/api"
	"github.com/exoscale/terraform-provider-exoscale/pkg/filter"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
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

	general.AddAttributes(ret, resourceSKSNodepool().Schema)

	return ret
}

func nodepoolToDataMap(nodepool *v2.SKSNodepool) general.TerraformObject {
	ret := make(general.TerraformObject)

	general.Assign(ret, resSKSNodepoolAttrAntiAffinityGroupIDs, nodepool.AntiAffinityGroupIDs)
	general.AssignTime(ret, resSKSNodepoolAttrCreatedAt, nodepool.CreatedAt)
	general.Assign(ret, resSKSNodepoolAttrDeployTargetID, nodepool.DeployTargetID)
	general.Assign(ret, resSKSNodepoolAttrDescription, nodepool.Description)
	general.Assign(ret, resSKSNodepoolAttrDiskSize, nodepool.DiskSize)
	general.Assign(ret, resSKSNodepoolAttrInstancePoolID, nodepool.InstancePoolID)
	general.Assign(ret, resSKSNodepoolAttrInstancePrefix, nodepool.InstancePrefix)
	general.Assign(ret, resSKSNodepoolAttrInstanceType, nodepool.InstanceTypeID)
	general.Assign(ret, resSKSNodepoolAttrLabels, nodepool.Labels)
	general.Assign(ret, resSKSNodepoolAttrName, nodepool.Name)
	general.Assign(ret, resSKSNodepoolAttrPrivateNetworkIDs, nodepool.PrivateNetworkIDs)
	general.Assign(ret, resSKSNodepoolAttrSecurityGroupIDs, nodepool.SecurityGroupIDs)
	general.Assign(ret, resSKSNodepoolAttrSize, nodepool.Size)
	general.Assign(ret, resSKSNodepoolAttrState, nodepool.State)
	general.Assign(ret, resSKSNodepoolAttrTaints, nodepool.Taints)
	general.Assign(ret, resSKSNodepoolAttrTemplateID, nodepool.TemplateID)
	general.Assign(ret, resSKSNodepoolAttrVersion, nodepool.Version)
	general.Assign(ret, dsSKSNodepoolID, nodepool.ID)

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

	client := getClient(meta)

	clusterID := d.Get(resSKSNodepoolAttrClusterID).(string)
	cluster, err := client.GetSKSCluster(ctx, zone, clusterID)
	if err != nil {
		return diag.Errorf("error getting cluster %q: %s", clusterID, err)
	}

	filters, err := filter.CreateFilters(ctx, d, dataSourceSKSNodepool().Schema)
	if err != nil {
		return diag.Errorf("failed to create filter: %q", err)
	}

	var matchingNodePool general.TerraformObject
	nMatches := 0

	for _, nodepool := range cluster.Nodepools {
		nodepoolData := nodepoolToDataMap(nodepool)
		nodepoolData[resSKSNodepoolAttrClusterID] = clusterID
		nodepoolData[resSKSNodepoolAttrZone] = zone
		if filter.CheckForMatch(nodepoolData, filters) {
			if nMatches < 1 {
				d.SetId(*nodepool.ID)

				matchingNodePool = nodepoolData

				nMatches++
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

	if err := general.Apply(matchingNodePool, d, dataSourceSKSNodepool().Schema); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
