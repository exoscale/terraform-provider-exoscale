---
layout: "exoscale"
page_title: "Exoscale: exoscale_affinity_group"
sidebar_current: "docs-exoscale-affinity-group"
description: |-
  Manages an affinity group.
---

# exoscale_affinity_group

Define an Affinity Group. `host anti-affinity` groups make sure than the virtual machines are not running on the same physical host.

## Example Usage

```hcl
resource "exoscale_affinity" "affinitylabel" {
  name = "affinity name"
  description = "long text"
  type = "host anti-affinity"
}
```

## Argument Reference

- `name` - (Required) name of the (anti-)Affinity Group.

- `description` - longer description.

- `type` - type of the Affinity Group. By default: `host anti-affinity`.

## Attributes Reference

The following attributes are exported:

- `id` - The id of the Affinity Group.

- `virtual_machine_ids` - The id of the compute resources member of the Affinity Group.

## Import

Importing an Affinity Group resource is possible by name or id.

```shell
# by name
$ terraform import exoscale_affinity_group.mygroup mygroup

# by id
$ terraform import exoscale_affinity_group.mygroup eb556678-ec59-4be6-8c54-0406ae0f6da6
```
