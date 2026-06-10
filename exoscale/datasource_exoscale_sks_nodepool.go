package exoscale

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/config"
	"github.com/exoscale/terraform-provider-exoscale/pkg/filter"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
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

func nodepoolToDataMap(nodepool *v3.SKSNodepool) general.TerraformObject {
	ret := make(general.TerraformObject)

	ret[dsSKSNodepoolID] = nodepool.ID.String()
	ret[resSKSNodepoolAttrCreatedAt] = nodepool.CreatedAT.Format(time.RFC3339)
	ret[resSKSNodepoolAttrDescription] = nodepool.Description
	ret[resSKSNodepoolAttrDiskSize] = nodepool.DiskSize
	ret[resSKSNodepoolAttrInstancePrefix] = nodepool.InstancePrefix
	ret[resSKSNodepoolAttrLabels] = map[string]string(nodepool.Labels)
	ret[resSKSNodepoolAttrName] = nodepool.Name
	ret[resSKSNodepoolAttrSize] = nodepool.Size
	ret[resSKSNodepoolAttrState] = string(nodepool.State)
	ret[resSKSNodepoolAttrVersion] = nodepool.Version

	if len(nodepool.AntiAffinityGroups) > 0 {
		ret[resSKSNodepoolAttrAntiAffinityGroupIDs] = utils.AntiAffiniGroupsToAntiAffinityGroupIDs(nodepool.AntiAffinityGroups)
	}
	if nodepool.DeployTarget != nil {
		ret[resSKSNodepoolAttrDeployTargetID] = nodepool.DeployTarget.ID.String()
	}
	if nodepool.InstancePool != nil {
		ret[resSKSNodepoolAttrInstancePoolID] = nodepool.InstancePool.ID.String()
	}
	if nodepool.InstanceType != nil {
		ret[resSKSNodepoolAttrInstanceType] = nodepool.InstanceType.ID.String()
	}
	if len(nodepool.PrivateNetworks) > 0 {
		ret[resSKSNodepoolAttrPrivateNetworkIDs] = utils.PrivateNetworksToPrivateNetworkIDs(nodepool.PrivateNetworks)
	}
	if len(nodepool.SecurityGroups) > 0 {
		ret[resSKSNodepoolAttrSecurityGroupIDs] = utils.SecurityGroupsToSecurityGroupIDs(nodepool.SecurityGroups)
	}
	if len(nodepool.Taints) > 0 {
		taints := make(map[string]string, len(nodepool.Taints))
		for k, v := range nodepool.Taints {
			taints[k] = fmt.Sprintf("%s:%s", v.Value, v.Effect)
		}
		ret[resSKSNodepoolAttrTaints] = taints
	}
	if nodepool.Template != nil {
		ret[resSKSNodepoolAttrTemplateID] = nodepool.Template.ID.String()
	}

	return ret
}

func dataSourceSKSNodepoolRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	tflog.Debug(ctx, "beginning read", map[string]any{
		"id": resourceSKSNodepoolIDString(d),
	})

	zone := d.Get(resSKSClusterAttrZone).(string)

	ctx, cancel := context.WithTimeout(ctx, d.Timeout(schema.TimeoutRead))
	defer cancel()

	client, err := config.GetClientV3WithZone(ctx, meta, zone)
	if err != nil {
		return diag.FromErr(err)
	}

	clusterID := d.Get(resSKSNodepoolAttrClusterID).(string)
	cluster, err := client.GetSKSCluster(ctx, v3.UUID(clusterID))
	if err != nil {
		return diag.Errorf("error getting cluster %q: %s", clusterID, err)
	}

	filters, err := filter.CreateFilters(ctx, d, dataSourceSKSNodepool().Schema)
	if err != nil {
		return diag.Errorf("failed to create filter: %q", err)
	}

	var matchingNodePool general.TerraformObject
	nMatches := 0

	for i := range cluster.Nodepools {
		nodepool := &cluster.Nodepools[i]
		nodepoolData := nodepoolToDataMap(nodepool)
		nodepoolData[resSKSNodepoolAttrClusterID] = clusterID
		nodepoolData[resSKSNodepoolAttrZone] = zone
		if filter.CheckForMatch(nodepoolData, filters) {
			if nMatches < 1 {
				d.SetId(nodepool.ID.String())

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
