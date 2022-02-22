---
page_title: "Exoscale: exoscale_anti_affinity_group"
description: |-
  Provides an Exoscale Anti-Affinity Group resource.
---

# exoscale\_affinity

Provides an Exoscale [Anti-Affinity Group][aag-doc] resource. This can be used to create and delete Anti-Affinity Groups.


## Example Usage

```hcl
resource "exoscale_anti_affinity_group" "example" {
  name        = "cluster"
  description = "HA Cluster"
}
```


## Arguments Reference

* `name` - (Required) The name of the Anti-Affinity Group.
* `description` - An Anti-Affinity Group description.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the Anti-Affinity Group.


## Import

An existing Anti-Affinity Group can be imported as a resource its ID:

```console
$ terraform import exoscale_anti_affinity_group.my-group eb556678-ec59-4be6-8c54-0406ae0f6da6
```


[aag-doc]: https://community.exoscale.com/documentation/compute/anti-affinity-groups/

