---
page_title: "Exoscale: exoscale_security_group"
description: |-
  Manage Exoscale Security Groups.
---

# exoscale\_security\_group

Manage Exoscale [Security Groups](https://community.exoscale.com/documentation/compute/security-groups/).


## Usage

```hcl
resource "exoscale_security_group" "my_security_group" {
  name = "my-security-group"
}
```

Next step is to attach [security group rules](./security_group_rule.md) to the security group.

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

[cidr]: https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing#CIDR_notatio

* `name` - (Required) The name of the security group.

* `description` - A free-form text describing the security group.

* `external_sources` - A list of external network sources in [CIDR][cidr] notation.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the security group.


## Import

An existing security group may be imported by `<ID>`:

```console
$ terraform import \
  exoscale_security_group.my_security_group \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6
```

~> **NOTE:** Importing a `exoscale_security_group` resource also imports related `exoscale_security_group_rule` resources.
