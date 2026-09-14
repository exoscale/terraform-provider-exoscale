package exoscale

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
	"github.com/exoscale/terraform-provider-exoscale/pkg/utils"
)

// NOTE: the `exoscale_sks_nodepool` resource and data source have been migrated
// to the terraform-plugin-framework (see pkg/resources/sks_cluster). The
// `exoscale_sks_nodepool_list` data source is still implemented with the
// SDKv2 and reuses the nodepool schema + `nodepoolToDataMap` below through the
// generic `list.FilterableListDataSource` helper. This file retains just
// enough of the former SDKv2 implementation to keep that list data source
// working until it is migrated in turn.

const (
	dsSKSNodepoolIdentifier = "exoscale_sks_nodepool"
	dsSKSNodepoolID         = "id"
)

const (
	defaultSKSNodepoolDiskSize       int64 = 50
	defaultSKSNodepoolInstancePrefix       = "pool"

	sksNodepoolAddonStorageLVM = "storage-lvm"

	resSKSNodepoolAttrAntiAffinityGroupIDs   = "anti_affinity_group_ids"
	resSKSNodepoolAttrClusterID              = "cluster_id"
	resSKSNodepoolAttrCreatedAt              = "created_at"
	resSKSNodepoolAttrDeployTargetID         = "deploy_target_id"
	resSKSNodepoolAttrDescription            = "description"
	resSKSNodepoolAttrDiskSize               = "disk_size"
	resSKSNodepoolAttrInstancePoolID         = "instance_pool_id"
	resSKSNodepoolAttrInstancePrefix         = "instance_prefix"
	resSKSNodepoolAttrInstanceType           = "instance_type"
	resSKSNodepoolAttrKubeletGC              = "kubelet_image_gc"
	resSKSNodepoolAttrKubeletGCMinAge        = "min_age"
	resSKSNodepoolAttrKubeletGCHighThreshold = "high_threshold"
	resSKSNodepoolAttrKubeletGCLowThreshold  = "low_threshold"
	resSKSNodepoolAttrKubeletMaxPods         = "kubelet_max_pods"
	resSKSNodepoolAttrLabels                 = "labels"
	resSKSNodepoolAttrID                     = "id"
	resSKSNodepoolAttrName                   = "name"
	resSKSNodepoolAttrNvidiaMigProfile       = "nvidia_mig_profile"
	resSKSNOdepoolAttrIPV6Enabled            = "ipv6"
	resSKSNodepoolAttrPrivateNetworkIDs      = "private_network_ids"
	resSKSNodepoolAttrSecurityGroupIDs       = "security_group_ids"
	resSKSNodepoolAttrSize                   = "size"
	resSKSNodepoolAttrState                  = "state"
	resSKSNodepoolAttrStorageLVM             = "storage_lvm"
	resSKSNodepoolAttrTaints                 = "taints"
	resSKSNodepoolAttrTemplateID             = "template_id"
	resSKSNodepoolAttrVersion                = "version"
	resSKSNodepoolAttrZone                   = "zone"
)

// sksNodepoolResourceSchema is a verbatim copy of the former SDKv2
// `resourceSKSNodepool()` schema map. It is only used to build the
// `exoscale_sks_nodepool_list` element schema.
func sksNodepoolResourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		resSKSNodepoolAttrAntiAffinityGroupIDs: {
			Type:        schema.TypeSet,
			Optional:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "A list of [exoscale_anti_affinity_group](./anti_affinity_group.md) (IDs) to be attached to the managed instances.",
		},
		resSKSNodepoolAttrClusterID: {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "The parent [exoscale_sks_cluster](./sks_cluster.md) ID.",
		},
		resSKSNodepoolAttrCreatedAt: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The pool creation date.",
		},
		resSKSNodepoolAttrDeployTargetID: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "A deploy target ID.",
		},
		resSKSNodepoolAttrDescription: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "A free-form text describing the pool.",
		},
		resSKSNodepoolAttrDiskSize: {
			Type:        schema.TypeInt,
			Optional:    true,
			Default:     defaultSKSNodepoolDiskSize,
			Description: "The managed instances disk size (GiB; default: `50`).",
		},
		resSKSNodepoolAttrInstancePoolID: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The underlying [exoscale_instance_pool](./instance_pool.md) ID.",
		},
		resSKSNodepoolAttrInstancePrefix: {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     defaultSKSNodepoolInstancePrefix,
			Description: "The string used to prefix the managed instances name (default `pool`).",
		},
		resSKSNodepoolAttrInstanceType: {
			Type:             schema.TypeString,
			Required:         true,
			ValidateDiagFunc: validateComputeInstanceType,
			// Ignore case differences
			DiffSuppressFunc: suppressCaseDiff,
			Description:      "The managed compute instances type (`<family>.<size>`, e.g. `standard.medium`; use the [Exoscale CLI](https://github.com/exoscale/cli/) - `exo compute instance-type list` - for the list of available types).",
		},
		resSKSNodepoolAttrKubeletGC: {
			Type:        schema.TypeSet,
			Optional:    true,
			Description: "Configuration for this nodepool's kubelet image garbage collector",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					resSKSNodepoolAttrKubeletGCMinAge: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "The minimum age for an unused image before it is garbage collected (k8s duration format, eg. 1h)",
					},
					resSKSNodepoolAttrKubeletGCHighThreshold: {
						Type:        schema.TypeInt,
						Optional:    true,
						Description: "The percent of disk usage after which image garbage collection is always run",
					},
					resSKSNodepoolAttrKubeletGCLowThreshold: {
						Type:        schema.TypeInt,
						Optional:    true,
						Description: "The percent of disk usage before which image garbage collection is never run",
					},
				},
			},
		},
		resSKSNodepoolAttrKubeletMaxPods: {
			Type:        schema.TypeInt,
			Optional:    true,
			Computed:    true,
			Description: "The maximum number of pods per node (default is 110).",
		},
		resSKSNodepoolAttrLabels: {
			Type:        schema.TypeMap,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Description: "A map of key/value labels.",
		},
		resSKSNodepoolAttrID: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The SKS node pool ID.",
		},
		resSKSNodepoolAttrName: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The SKS node pool name.",
		},
		resSKSNodepoolAttrNvidiaMigProfile: {
			Type:     schema.TypeString,
			Optional: true,
			Description: "The NVIDIA [Multi-Instance GPU (MIG)](https://docs.nvidia.com/datacenter/tesla/mig-user-guide/) profile to enable on the managed GPUs. " +
				"The GPU family is inferred from `instance_type`: `gpua30.*` accepts `2g.12gb`, `1g.6gb+me`, `1g.6gb`, `2g.12gb+me`, `4g.24gb`; " +
				"`gpurtx6000pro.*` accepts `1g.24gb-me`, `1g.24gb`, `2g.48gb-me`, `2g.48gb`, `4g.96gb+gfx`, `1g.24gb+me`, `2g.48gb+me.all`, `1g.24gb+gfx`, `1g.24gb+me.all`, `4g.96gb`, `2g.48gb+gfx`.",
		},
		resSKSNOdepoolAttrIPV6Enabled: {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Enable IPV6 for the nodepool nodes",
		},
		resSKSNodepoolAttrPrivateNetworkIDs: {
			Type:        schema.TypeSet,
			Optional:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "A list of [exoscale_private_network](./private_network.md) (IDs) to be attached to the managed instances.",
		},
		resSKSNodepoolAttrSecurityGroupIDs: {
			Type:        schema.TypeSet,
			Optional:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "A list of [exoscale_security_group](./security_group.md) (IDs) to be attached to the managed instances.",
		},
		resSKSNodepoolAttrSize: {
			Type:     schema.TypeInt,
			Required: true,
		},
		resSKSNodepoolAttrState: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The current pool state.",
		},
		resSKSNodepoolAttrStorageLVM: {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Create nodes with non-standard partitioning for persistent storage (requires min 100G of disk space) (may only be set at creation time).",
		},
		resSKSNodepoolAttrTaints: {
			Type:        schema.TypeMap,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Description: `A map of key/value Kubernetes [taints](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) ('taints = { <key> = "<value>:<effect>" }').`,
		},
		resSKSNodepoolAttrTemplateID: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The managed instances template ID.",
		},
		resSKSNodepoolAttrVersion: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The managed instances version.",
		},
		resSKSNodepoolAttrZone: {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
		},
	}
}

// nodepoolToDataMap is a verbatim copy of the former SDKv2 helper, used by the
// `exoscale_sks_nodepool_list` data source.
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
	if nodepool.KubeletMaxPods != nil {
		ret[resSKSNodepoolAttrKubeletMaxPods] = int(*nodepool.KubeletMaxPods)
	}
	if nodepool.InstancePool != nil {
		ret[resSKSNodepoolAttrInstancePoolID] = nodepool.InstancePool.ID.String()
	}
	if nodepool.InstanceType != nil {
		ret[resSKSNodepoolAttrInstanceType] = nodepool.InstanceType.ID.String()
	}
	if profile := sksNodepoolMIGProfile(nodepool.NvidiaMigProfiles); profile != "" {
		ret[resSKSNodepoolAttrNvidiaMigProfile] = profile
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

// sksNodepoolMIGProfile is a verbatim copy of the former SDKv2 helper (flatten
// only), used by `nodepoolToDataMap` above.
func sksNodepoolMIGProfile(profiles *v3.NvidiaMigProfiles) string {
	if profiles == nil {
		return ""
	}
	if profiles.A3024gb != "" {
		return string(profiles.A3024gb)
	}
	if profiles.Rtxpro600096gb != "" {
		return string(profiles.Rtxpro600096gb)
	}

	return ""
}
