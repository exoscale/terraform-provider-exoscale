package exoscale

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	v3 "github.com/exoscale/egoscale/v3"
	"github.com/exoscale/terraform-provider-exoscale/pkg/general"
)

// NOTE: the `exoscale_sks_cluster` resource and data source have been migrated to
// the terraform-plugin-framework (see pkg/resources/sks_cluster). The
// `exoscale_sks_cluster_list` data source is still implemented with the SDKv2
// and reuses the cluster schema + `clusterToDataMap` below through the generic
// `list.FilterableListDataSource` helper. This file retains just enough of the
// former SDKv2 implementation to keep that list data source working until it is
// migrated in turn.

const (
	dsSKSClusterIdentifier = "exoscale_sks_cluster"
	dsSKSClusterID         = "id"
)

const (
	defaultSKSClusterCNI              = "calico"
	defaultSKSClusterServiceLevel     = "pro"
	defaultSKSClusterAuditInitBackoff = "10s"

	resSKSClusterAttrAddons                     = "addons"
	resSKSClusterAttrAggregationLayerCA         = "aggregation_ca"
	resSKSClusterAttrAuditBearerToken           = "bearer_token"
	resSKSClusterAttrAuditEnabled               = "enabled"
	resSKSClusterAttrAuditEndpoint              = "endpoint"
	resSKSClusterAttrAuditInitBackoff           = "initial_backoff"
	resSKSClusterAttrAutoUpgrade                = "auto_upgrade"
	resSKSClusterAttrCNI                        = "cni"
	resSKSClusterAttrControlPlaneCA             = "control_plane_ca"
	resSKSClusterAttrCreatedAt                  = "created_at"
	resSKSClusterAttrCreateDefaultSecurityGroup = "create_default_security_group"
	resSKSClusterAttrDefaultSecurityGroupID     = "default_security_group_id"
	resSKSClusterAttrDescription                = "description"
	resSKSClusterAttrEnableKubeProxy            = "enable_kube_proxy"
	resSKSClusterAttrEnableKarpenter            = "enable_karpenter"
	resSKSClusterAttrEndpoint                   = "endpoint"
	resSKSClusterAttrExoscaleCCM                = "exoscale_ccm"
	resSKSClusterAttrExoscaleCSI                = "exoscale_csi"
	resSKSClusterAttrFeatureGates               = "feature_gates"
	resSKSClusterAttrKubeletCA                  = "kubelet_ca"
	resSKSClusterAttrLabels                     = "labels"
	resSKSClusterAttrMetricsServer              = "metrics_server"
	resSKSClusterAttrID                         = "id"
	resSKSClusterAttrName                       = "name"
	resSKSClusterAttrNodepools                  = "nodepools"
	resSKSClusterAttrOIDCClientID               = "client_id"
	resSKSClusterAttrOIDCGroupsClaim            = "groups_claim"
	resSKSClusterAttrOIDCGroupsPrefix           = "groups_prefix"
	resSKSClusterAttrOIDCIssuerURL              = "issuer_url"
	resSKSClusterAttrOIDCRequiredClaim          = "required_claim"
	resSKSClusterAttrOIDCUsernameClaim          = "username_claim"
	resSKSClusterAttrOIDCUsernamePrefix         = "username_prefix"
	resSKSClusterAttrServiceLevel               = "service_level"
	resSKSClusterAttrState                      = "state"
	resSKSClusterAttrVersion                    = "version"
	resSKSClusterAttrZone                       = "zone"
)

// sksClusterResourceSchema is a verbatim copy of the former SDKv2
// `resourceSKSCluster()` schema map. It is only used to build the
// `exoscale_sks_cluster_list` element schema.
func sksClusterResourceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		resSKSClusterAttrAddons: {
			Type:     schema.TypeSet,
			Set:      schema.HashString,
			Elem:     &schema.Schema{Type: schema.TypeString},
			Optional: true,
			Computed: true,
			Deprecated: "This attribute has been replaced by `exoscale_ccm`/`metrics_server` " +
				"attributes, it will be removed in a future release.",
		},

		resSKSClusterAttrAggregationLayerCA: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The CA certificate (in PEM format) for TLS communications between the control plane and the aggregation layer (e.g. `metrics-server`).",
			Sensitive:   true,
		},
		"audit": {
			Type:        schema.TypeList,
			MaxItems:    1,
			Optional:    true,
			Description: "Parameters for Kubernetes Audit configuration (may only be enabled at creation time)",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					resSKSClusterAttrAuditEnabled: {
						Type:        schema.TypeBool,
						Optional:    true,
						Description: "Whether to run the APIServer with the configured Kubernetes Audit",
					},
					resSKSClusterAttrAuditEndpoint: {
						Type:        schema.TypeString,
						Optional:    true, // Checked at runtime
						Description: "The Endpoint URL for the Webserver responsible of processing Audit events",
					},
					resSKSClusterAttrAuditInitBackoff: {
						Type:        schema.TypeString,
						Optional:    true,
						Default:     defaultSKSClusterAuditInitBackoff,
						Description: "The Initial Backoff to wait before sending data to the remote server (default '10s')",
					},
					resSKSClusterAttrAuditBearerToken: {
						Type:        schema.TypeString,
						Optional:    true,
						Sensitive:   true,
						Description: "The optional bearer token to include in the request header",
					},
				},
			},
		},
		resSKSClusterAttrAutoUpgrade: {
			Type:        schema.TypeBool,
			Optional:    true,
			Description: "Enable automatic upgrading of the control plane version.",
		},
		resSKSClusterAttrCNI: {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     defaultSKSClusterCNI,
			Description: fmt.Sprintf(`The CNI plugin that is to be used. Available options are "calico" or "cilium". Defaults to %q. Setting empty string will result in a cluster with no CNI.`, defaultSKSClusterCNI),
		},
		resSKSClusterAttrControlPlaneCA: {
			Type:        schema.TypeString,
			Computed:    true,
			Sensitive:   true,
			Description: "The CA certificate (in PEM format) for TLS communications between control plane components.",
		},
		resSKSClusterAttrCreatedAt: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The cluster creation date.",
		},
		resSKSClusterAttrCreateDefaultSecurityGroup: {
			Type:        schema.TypeBool,
			Optional:    true,
			ForceNew:    true,
			Description: "Creates an ad-hoc security group based on the choice of the selected CNI (may only be set at creation time).",
		},
		resSKSClusterAttrDefaultSecurityGroupID: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The ID of the cluster's ad-hoc default security group (when `create_default_security_group` was set at creation time).",
		},
		resSKSClusterAttrDescription: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "A free-form text describing the cluster.",
		},
		resSKSClusterAttrEnableKubeProxy: {
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
			Description: "Indicates whether to deploy the Kubernetes network proxy. (may only be set at creation time)",
			ForceNew:    true,
		},
		resSKSClusterAttrEnableKarpenter: {
			Type:        schema.TypeBool,
			Optional:    true,
			Computed:    true,
			Description: "Indicates whether to deploy Karpenter for cluster autoscaling.",
		},
		resSKSClusterAttrEndpoint: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The cluster API endpoint.",
		},
		resSKSClusterAttrExoscaleCCM: {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Deploy the Exoscale [Cloud Controller Manager](https://github.com/exoscale/exoscale-cloud-controller-manager/) in the control plane (boolean; default: `true`; may only be set at creation time).",
		},
		resSKSClusterAttrFeatureGates: {
			Type:        schema.TypeSet,
			Optional:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "Feature gates options for the cluster.",
		},
		resSKSClusterAttrKubeletCA: {
			Type:        schema.TypeString,
			Computed:    true,
			Sensitive:   true,
			Description: "The CA certificate (in PEM format) for TLS communications between kubelets and the control plane.",
		},
		resSKSClusterAttrMetricsServer: {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     true,
			Description: "Deploy the [Kubernetes Metrics Server](https://github.com/kubernetes-sigs/metrics-server/) in the control plane (boolean; default: `true`; may only be set at creation time).",
		},
		resSKSClusterAttrExoscaleCSI: {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Deploy the Exoscale [Container Storage Interface](https://github.com/exoscale/exoscale-csi-driver/) on worker nodes (boolean; default: `false`; requires the CCM to be enabled).",
		},
		resSKSClusterAttrLabels: {
			Type:        schema.TypeMap,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Optional:    true,
			Description: "A map of key/value labels.",
		},
		resSKSClusterAttrName: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The SKS cluster name.",
		},
		resSKSClusterAttrID: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The SKS cluster ID.",
		},
		resSKSClusterAttrNodepools: {
			Type:        schema.TypeSet,
			Computed:    true,
			Set:         schema.HashString,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "The list of [exoscale_sks_nodepool](./sks_nodepool.md) (IDs) attached to the cluster.",
		},
		"oidc": {
			Type:        schema.TypeList,
			MaxItems:    1,
			Optional:    true,
			Computed:    true,
			Description: "An OpenID Connect configuration to provide to the Kubernetes API server (may only be set at creation time). Structure is documented below.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					resSKSClusterAttrOIDCClientID: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The OpenID client ID.",
					},
					resSKSClusterAttrOIDCGroupsClaim: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "An OpenID JWT claim to use as the user's group.",
					},
					resSKSClusterAttrOIDCGroupsPrefix: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "An OpenID prefix prepended to group claims.",
					},
					resSKSClusterAttrOIDCIssuerURL: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The OpenID provider URL.",
					},
					resSKSClusterAttrOIDCRequiredClaim: {
						Type:        schema.TypeMap,
						Elem:        &schema.Schema{Type: schema.TypeString},
						Optional:    true,
						Description: "A map of key/value pairs that describes a required claim in the OpenID Token.",
					},
					resSKSClusterAttrOIDCUsernameClaim: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "An OpenID JWT claim to use as the user name.",
					},
					resSKSClusterAttrOIDCUsernamePrefix: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "An OpenID prefix prepended to username claims.",
					},
				},
			},
		},
		resSKSClusterAttrServiceLevel: {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     defaultSKSClusterServiceLevel,
			Description: "The service level of the control plane (`pro` or `starter`; default: `pro`; may only be set at creation time).",
		},
		resSKSClusterAttrState: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The cluster state.",
		},
		resSKSClusterAttrVersion: {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "The version of the control plane (default: latest version available from the API; see `exo compute sks versions` for reference; may only be set at creation time).",
		},
		resSKSClusterAttrZone: {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.",
		},
	}
}

// clusterToDataMap is a verbatim copy of the former SDKv2 helper, used by the
// `exoscale_sks_cluster_list` data source.
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
