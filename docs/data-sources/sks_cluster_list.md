---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "exoscale_sks_cluster_list Data Source - terraform-provider-exoscale"
subcategory: ""
description: |-
  
---

# exoscale_sks_cluster_list (Data Source)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `zone` (String) The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.

### Optional

- `aggregation_ca` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `auto_upgrade` (Boolean) Match against this bool
- `cni` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `control_plane_ca` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `created_at` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `description` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `enable_kube_proxy` (Boolean) Match against this bool
- `endpoint` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `exoscale_ccm` (Boolean) Match against this bool
- `exoscale_csi` (Boolean) Match against this bool
- `id` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `kubelet_ca` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `labels` (Map of String) Match against key/values. Keys are matched exactly, while values may be matched as a regex if you supply a string that begins and ends with "/"
- `metrics_server` (Boolean) Match against this bool
- `name` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `service_level` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `state` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `version` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.

### Read-Only

- `clusters` (List of Object) (see [below for nested schema](#nestedatt--clusters))

<a id="nestedatt--clusters"></a>
### Nested Schema for `clusters`

Read-Only:

- `addons` (Set of String)
- `aggregation_ca` (String)
- `auto_upgrade` (Boolean)
- `cni` (String)
- `control_plane_ca` (String)
- `created_at` (String)
- `description` (String)
- `enable_kube_proxy` (Boolean)
- `endpoint` (String)
- `exoscale_ccm` (Boolean)
- `exoscale_csi` (Boolean)
- `feature_gates` (Set of String)
- `id` (String)
- `kubelet_ca` (String)
- `labels` (Map of String)
- `metrics_server` (Boolean)
- `name` (String)
- `nodepools` (Set of String)
- `oidc` (List of Object) (see [below for nested schema](#nestedobjatt--clusters--oidc))
- `service_level` (String)
- `state` (String)
- `version` (String)
- `zone` (String)

<a id="nestedobjatt--clusters--oidc"></a>
### Nested Schema for `clusters.oidc`

Read-Only:

- `client_id` (String)
- `groups_claim` (String)
- `groups_prefix` (String)
- `issuer_url` (String)
- `required_claim` (Map of String)
- `username_claim` (String)
- `username_prefix` (String)


