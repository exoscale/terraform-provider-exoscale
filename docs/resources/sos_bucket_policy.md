---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "exoscale_sos_bucket_policy Resource - terraform-provider-exoscale"
subcategory: ""
description: |-
  Manage Exoscale SOS Bucket Policies https://community.exoscale.com/product/storage/object-storage/how-to/bucketpolicy/.
---

# exoscale_sos_bucket_policy (Resource)

Manage Exoscale [SOS Bucket Policies](https://community.exoscale.com/product/storage/object-storage/how-to/bucketpolicy/).



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `bucket` (String) ❗ The name of the bucket to which the policy is to be applied.
- `policy` (String) The content of the policy
- `zone` (String) ❗ The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.

### Optional

- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))

<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours).
- `delete` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours). Setting a timeout for a Delete operation is only applicable if changes are saved into state before the destroy operation occurs.
- `read` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours). Read operations occur during any refresh or planning operation when refresh is enabled.
- `update` (String) A string that can be [parsed as a duration](https://pkg.go.dev/time#ParseDuration) consisting of numbers and unit suffixes, such as "30s" or "2h45m". Valid time units are "s" (seconds), "m" (minutes), "h" (hours).

-> The symbol ❗ in an attribute indicates that modifying it, will force the creation of a new resource.


