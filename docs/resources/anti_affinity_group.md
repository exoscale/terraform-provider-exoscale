---
page_title: "Exoscale: exoscale_anti_affinity_group"
description: |-
  Manage Exoscale Anti-Affinity Groups.
---

# exoscale\_anti\_affinity\_group

Manage Exoscale [Anti-Affinity Groups](https://community.exoscale.com/documentation/compute/anti-affinity-groups/).

Corresponding data source: [exoscale_anti_affinity_group](../data-sources/anti_affinity_group.md).


## Usage

```hcl
resource "exoscale_anti_affinity_group" "my_anti_affinity_group" {
  name        = "my-anti-affinity-group"
  description = "Prevent compute instances to run on the same host"
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

* `name` - (Required) The anti-affinity group name.

* `description` - A free-form text describing the group.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The anti-affinity group ID.


## Import

An existing anti-affinity group may be imported by `<ID>`:

```console
$ terraform import \
  exoscale_anti_affinity_group.my_anti_affinity_group \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6
```
