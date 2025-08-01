---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "exoscale_block_storage_volume_snapshot Resource - terraform-provider-exoscale"
subcategory: ""
description: |-
  Manage Exoscale Block Storage https://community.exoscale.com/product/storage/block-storage/ Volume Snapshot.
  Block Storage offers persistent externally attached volumes for your workloads.
---

# exoscale_block_storage_volume_snapshot (Resource)

Manage [Exoscale Block Storage](https://community.exoscale.com/product/storage/block-storage/) Volume Snapshot.

Block Storage offers persistent externally attached volumes for your workloads.



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) Volume snapshot name.
- `volume` (Attributes) Volume from which to create a snapshot. (see [below for nested schema](#nestedatt--volume))
- `zone` (String) ❗ The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.

### Optional

- `labels` (Map of String) Resource labels.
- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))

### Read-Only

- `created_at` (String) Snapshot creation date.
- `id` (String) The ID of this resource.
- `size` (Number) Snapshot size in GB.
- `state` (String) Snapshot state.

<a id="nestedatt--volume"></a>
### Nested Schema for `volume`

Required:

- `id` (String) Snapshot ID.


<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `read` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours). Read operations occur during any refresh or planning operation when refresh is enabled.

-> The symbol ❗ in an attribute indicates that modifying it, will force the creation of a new resource.


