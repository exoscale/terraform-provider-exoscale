---
layout: "exoscale"
page_title: "Exoscale: exoscale_security_group"
sidebar_current: "docs-exoscale-security-group"
description: |-
  Provides an Exoscale Security Group.
---

# exoscale\_security\_group

Provides an Exoscale [Security Group][sg-doc] resource. This can be used to create and delete Security Groups.


## Example usage

```hcl
resource "exoscale_security_group" "web" {
  name             = "web"
  description      = "Webservers"
  external_sources = ["1.2.3.4/32", "5.6.7.8/32"]
}
```


## Arguments Reference

In addition to the arguments listed above, the following attributes are exported:

* `name` - (Required) The name of the Security Group.
* `description` - A free-form text describing the Security Group purpose.
* `external_sources` - A list of external network sources in [CIDR][cidr] notation.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the Security Group.


## Import

An existing Security Group can be imported as a resource by its ID:

```console
$ terraform import exoscale_security_group.http eb556678-ec59-4be6-8c54-0406ae0f6da6
```

~> **NOTE:** Importing a `exoscale_security_group` resource also imports related [`exoscale_security_group_rule`][r-security_group_rule] resources.


[cidr]: https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing#CIDR_notatio
[r-security_group_rule]: security_group_rule.html
[sg-doc]: https://community.exoscale.com/documentation/compute/security-groups/
