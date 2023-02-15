package exoscale

import (
	"context"
	"fmt"
	"time"

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

// TODO move to package
func assignTime(data terraformObject, attributeIdentifier string, value *time.Time) {
	if value == nil {
		return
	}

	data[attributeIdentifier] = value.Format(time.RFC3339)
}

func assign[T any](data terraformObject, attributeIdentifier string, value *T) {
	if value == nil {
		return
	}

	data[attributeIdentifier] = *value
}

func nodepoolToDataMap(nodepool *v2.SKSNodepool) terraformObject {
	ret := make(terraformObject)

	assign(ret, resSKSNodepoolAttrAntiAffinityGroupIDs, nodepool.AntiAffinityGroupIDs)
	assignTime(ret, resSKSNodepoolAttrCreatedAt, nodepool.CreatedAt)
	assign(ret, resSKSNodepoolAttrDeployTargetID, nodepool.DeployTargetID)
	assign(ret, resSKSNodepoolAttrDescription, nodepool.Description)
	assign(ret, resSKSNodepoolAttrDiskSize, nodepool.DiskSize)
	assign(ret, resSKSNodepoolAttrInstancePoolID, nodepool.InstancePoolID)
	assign(ret, resSKSNodepoolAttrInstancePrefix, nodepool.InstancePrefix)
	assign(ret, resSKSNodepoolAttrInstanceType, nodepool.InstanceTypeID)
	assign(ret, resSKSNodepoolAttrLabels, nodepool.Labels)
	assign(ret, resSKSNodepoolAttrName, nodepool.Name)
	assign(ret, resSKSNodepoolAttrPrivateNetworkIDs, nodepool.PrivateNetworkIDs)
	assign(ret, resSKSNodepoolAttrSecurityGroupIDs, nodepool.SecurityGroupIDs)
	assign(ret, resSKSNodepoolAttrSize, nodepool.Size)
	assign(ret, resSKSNodepoolAttrState, nodepool.State)
	assign(ret, resSKSNodepoolAttrTaints, nodepool.Taints)
	assign(ret, resSKSNodepoolAttrTemplateID, nodepool.TemplateID)
	assign(ret, resSKSNodepoolAttrVersion, nodepool.Version)
	assign(ret, dsSKSNodepoolID, nodepool.ID)

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

	var matchingNodePool terraformObject
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
