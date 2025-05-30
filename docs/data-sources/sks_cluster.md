---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "exoscale_sks_cluster Data Source - terraform-provider-exoscale"
subcategory: ""
description: |-
  
---

# exoscale_sks_cluster (Data Source)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `zone` (String)

### Optional

- `addons` (Set of String, Deprecated)
- `aggregation_ca` (String) The CA certificate (in PEM format) for TLS communications between the control plane and the aggregation layer (e.g. `metrics-server`).
- `auto_upgrade` (Boolean) Enable automatic upgrading of the control plane version.
- `cni` (String) The CNI plugin that is to be used. Available options are "calico" or "cilium". Defaults to "calico". Setting empty string will result in a cluster with no CNI.
- `control_plane_ca` (String) The CA certificate (in PEM format) for TLS communications between control plane components.
- `created_at` (String) The cluster creation date.
- `description` (String) A free-form text describing the cluster.
- `enable_kube_proxy` (Boolean) ❗ Indicates whether to deploy the Kubernetes network proxy. (may only be set at creation time)
- `endpoint` (String) The cluster API endpoint.
- `exoscale_ccm` (Boolean) Deploy the Exoscale [Cloud Controller Manager](https://github.com/exoscale/exoscale-cloud-controller-manager/) in the control plane (boolean; default: `true`; may only be set at creation time).
- `exoscale_csi` (Boolean) Deploy the Exoscale [Container Storage Interface](https://github.com/exoscale/exoscale-csi-driver/) on worker nodes (boolean; default: `false`; requires the CCM to be enabled).
- `feature_gates` (Set of String) Feature gates options for the cluster.
- `kubelet_ca` (String) The CA certificate (in PEM format) for TLS communications between kubelets and the control plane.
- `labels` (Map of String) A map of key/value labels.
- `metrics_server` (Boolean) Deploy the [Kubernetes Metrics Server](https://github.com/kubernetes-sigs/metrics-server/) in the control plane (boolean; default: `true`; may only be set at creation time).
- `name` (String)
- `nodepools` (Set of String) The list of [exoscale_sks_nodepool](./sks_nodepool.md) (IDs) attached to the cluster.
- `oidc` (Block List, Max: 1) An OpenID Connect configuration to provide to the Kubernetes API server (may only be set at creation time). Structure is documented below. (see [below for nested schema](#nestedblock--oidc))
- `service_level` (String) The service level of the control plane (`pro` or `starter`; default: `pro`; may only be set at creation time).
- `state` (String) The cluster state.
- `version` (String) The version of the control plane (default: latest version available from the API; see `exo compute sks versions` for reference; may only be set at creation time).

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--oidc"></a>
### Nested Schema for `oidc`

Required:

- `client_id` (String) The OpenID client ID.
- `issuer_url` (String) The OpenID provider URL.

Optional:

- `groups_claim` (String) An OpenID JWT claim to use as the user's group.
- `groups_prefix` (String) An OpenID prefix prepended to group claims.
- `required_claim` (Map of String) A map of key/value pairs that describes a required claim in the OpenID Token.
- `username_claim` (String) An OpenID JWT claim to use as the user name.
- `username_prefix` (String) An OpenID prefix prepended to username claims.


