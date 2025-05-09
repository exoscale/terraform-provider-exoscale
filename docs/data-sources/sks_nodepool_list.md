---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "exoscale_sks_nodepool_list Data Source - terraform-provider-exoscale"
subcategory: ""
description: |-
  
---

# exoscale_sks_nodepool_list (Data Source)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `zone` (String) The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.

### Optional

- `cluster_id` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `created_at` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `deploy_target_id` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `description` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `disk_size` (Number) Match against this int
- `id` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `instance_pool_id` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `instance_prefix` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `instance_type` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `ipv6` (Boolean) Match against this bool
- `labels` (Map of String) Match against key/values. Keys are matched exactly, while values may be matched as a regex if you supply a string that begins and ends with "/"
- `name` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `size` (Number) Match against this int
- `state` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `storage_lvm` (Boolean) Match against this bool
- `taints` (Map of String) Match against key/values. Keys are matched exactly, while values may be matched as a regex if you supply a string that begins and ends with "/"
- `template_id` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.
- `version` (String) Match against this string. If you supply a string that begins and ends with a "/" it will be matched as a regex.

### Read-Only

- `nodepools` (List of Object) (see [below for nested schema](#nestedatt--nodepools))

<a id="nestedatt--nodepools"></a>
### Nested Schema for `nodepools`

Read-Only:

- `anti_affinity_group_ids` (Set of String)
- `cluster_id` (String)
- `created_at` (String)
- `deploy_target_id` (String)
- `description` (String)
- `disk_size` (Number)
- `id` (String)
- `instance_pool_id` (String)
- `instance_prefix` (String)
- `instance_type` (String)
- `ipv6` (Boolean)
- `kubelet_image_gc` (Set of Object) (see [below for nested schema](#nestedobjatt--nodepools--kubelet_image_gc))
- `labels` (Map of String)
- `name` (String)
- `private_network_ids` (Set of String)
- `security_group_ids` (Set of String)
- `size` (Number)
- `state` (String)
- `storage_lvm` (Boolean)
- `taints` (Map of String)
- `template_id` (String)
- `version` (String)
- `zone` (String)

<a id="nestedobjatt--nodepools--kubelet_image_gc"></a>
### Nested Schema for `nodepools.kubelet_image_gc`

Read-Only:

- `high_threshold` (Number)
- `low_threshold` (Number)
- `min_age` (String)


