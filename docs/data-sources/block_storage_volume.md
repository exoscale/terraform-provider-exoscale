---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "exoscale_block_storage_volume Data Source - terraform-provider-exoscale"
subcategory: ""
description: |-
  Fetch Exoscale Block Storage https://community.exoscale.com/product/storage/block-storage/ Volume.
  Block Storage offers persistent externally attached volumes for your workloads.
  Corresponding resource: exoscaleblockstorage_volume ../resources/block_storage_volume.md.
---

# exoscale_block_storage_volume (Data Source)

Fetch [Exoscale Block Storage](https://community.exoscale.com/product/storage/block-storage/) Volume.

Block Storage offers persistent externally attached volumes for your workloads.

Corresponding resource: [exoscale_block_storage_volume](../resources/block_storage_volume.md).



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `id` (String) Volume ID to match.
- `zone` (String) The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.

### Optional

- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))

### Read-Only

- `blocksize` (Number) Volume block size.
- `created_at` (String) Volume creation date.
- `instance` (Attributes) Volume attached instance. (see [below for nested schema](#nestedatt--instance))
- `labels` (Map of String) Resource labels.
- `name` (String) Volume name.
- `size` (Number) Volume size in GB.
- `snapshots` (Attributes Set) Volume snapshots. (see [below for nested schema](#nestedatt--snapshots))
- `state` (String) Volume state.

<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `read` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours). Read operations occur during any refresh or planning operation when refresh is enabled.


<a id="nestedatt--instance"></a>
### Nested Schema for `instance`

Read-Only:

- `id` (String) Instance ID.


<a id="nestedatt--snapshots"></a>
### Nested Schema for `snapshots`

Read-Only:

- `id` (String) Snapshot ID.


