---
page_title: "Exoscale: exoscale_affinity"
subcategory: "Deprecated"
description: |-
  Provides an Exoscale Anti-Affinity Group resource.
---

# exoscale\_affinity

Provides an Exoscale [Anti-Affinity Group][aag-doc] resource. This can be used to create and delete Anti-Affinity Groups.

!> **WARNING:** This resource is deprecated and will be removed in the next major version.


## Example Usage

```hcl
resource "exoscale_affinity" "cluster" {
  name        = "cluster"
  description = "HA Cluster"
  type        = "host anti-affinity"
}
```


## Arguments Reference

* `name` - (Required) The name of the Anti-Affinity Group.
* `description` - A free-form text describing the Anti-Affinity Group purpose.
* `type` - The type of the Anti-Affinity Group (`host anti-affinity` is the only supported value).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the Anti-Affinity Group.
* `virtual_machine_ids` - The IDs of the Compute instance resources member of the Anti-Affinity Group.


## Import

An existing Anti-Affinity Group can be imported as a resource by name or ID:

```console
# By name
$ terraform import exoscale_affinity.mygroup mygroup

# By ID
$ terraform import exoscale_affinity.mygroup eb556678-ec59-4be6-8c54-0406ae0f6da6
```


[aag-doc]: https://community.exoscale.com/documentation/compute/anti-affinity-groups/

