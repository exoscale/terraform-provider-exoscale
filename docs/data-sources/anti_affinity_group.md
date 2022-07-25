---
page_title: "Exoscale: exoscale_anti_affinity_group"
description: |-
  Fetch Exoscale Anti-Affinity Groups data.
---

# exoscale\_anti\_affinity\_group

Fetch Exoscale [Anti-Affinity Groups](https://community.exoscale.com/documentation/compute/anti-affinity-groups/) data.

Corresponding resource: [exoscale_anti_affinity_group](../resources/anti_affinity_group.md).


## Usage

```hcl
data "exoscale_anti_affinity_group" "my_anti_affinity_group" {
  name = "my-anti-affinity-group"
}

output "my_anti_affinity_group_id" {
  value = data.exoscale_anti_affinity_group.my_anti_affinity_group.id
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

* `id` - The anti-affinity group ID to match (conflicts with `name`).
* `name` - The group name to match (conflicts with `id`).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `instances` - The list of attached [exoscale_compute_instance](../resources/compute_instance.md) (IDs).
