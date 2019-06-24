---
layout: "exoscale"
page_title: "Exoscale: exoscale_security_group"
sidebar_current: "docs-exoscale-security-group"
description: |-
  Provides an Exoscale Security Group.
---

# exoscale\_security\_group

Provides an Exoscale [Security Group][sg] resource. This can be used to create and delete Security Groups.

[sg]: https://community.exoscale.com/documentation/compute/security-groups/

## Example usage

```hcl
resource "exoscale_security_group" "web" {
  name        = "web"
  description = "Webservers"

  tags = {
    kind = "web"
  }
}
```

## Argument Reference

The following attributes are exported:

* `name` - (Required) The name of the Security Group.
* `description` - A free-form text describing the Anti-Affinity Group purpose.
* `tags` - A dictionary of tags (key/value).

## Import

An existing Security Group can be imported as a resource by name or ID:

```console
# By name
$ terraform import exoscale_security_group.http http

# By ID
$ terraform import exoscale_security_group.http eb556678-ec59-4be6-8c54-0406ae0f6da6
```

~> **NOTE:** Importing a `exoscale_security_group` resource also imports related [`exoscale_security_group_rule`][sgrule] resources.

[sgrule]: security_group_rule.html