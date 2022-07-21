---
page_title: "Exoscale: exoscale_anti_affinity_group"
description: |-
  Manage Exoscale Anti-Affinity Groups.
---

# exoscale\_anti\_affinity\_group

Manage Exoscale [Anti-Affinity Groups](https://community.exoscale.com/documentation/compute/anti-affinity-groups/).


## Usage

```hcl
resource "exoscale_anti_affinity_group" "my_anti_affinity_group" {
  name        = "my-anti-affinity-group"
  description = "Prevent compute instances to run on the same host"
}
```

Please refer to the [examples](../../examples/) directory for complete configuration examples.


## Arguments Reference

* `name` - (Required) The name of the anti-affinity group.

* `description` - A free-form text describing the anti-affinity group.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the anti-affinity group.


## Import

An existing anti-affinity group may be imported by `<ID>`:

```console
$ terraform import \
  exoscale_anti_affinity_group.my_anti_affinity_group \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6
```
