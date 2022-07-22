---
page_title: "Exoscale: exoscale_affinity"
subcategory: "Deprecated"
description: |-
  Manage Exoscale Anti-Affinity Groups.
---

# exoscale\_affinity

!> **WARNING:** This resource is **DEPRECATED** and will be removed in the next major version. Please use [exoscale_anti_affinity_group](./anti_affinity_group.md) instead.


## Arguments Reference

* `name` - (Required) The name of the anti-affinity group.

* `description` - A free-form text describing the anti-affinity group.
* `type` - The type of the anti-affinity group (`host anti-affinity` is the only supported value).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the anti-affinity group.
* `virtual_machine_ids` - The IDs of the compute instance members of the anti-affinity group.
