---
layout: "exoscale"
page_title: "Exoscale: exoscale_security_group"
sidebar_current: "docs-exoscale-security-group"
description: |-
  Manages a security group.
---

# exoscale_security_group

Security Groups allow you to define and compose firewall rules
making it easy to manage incoming and outgoing traffic. They
offer the best way to 

## Example usage

```hcl
resource "exoscale_security_group" "http" {
  name = "HTTP"
  description = "Long text"

  tags {
    kind = "web"
  }
}
```

## Argument Reference

- `name` - (Required) name of the security group

- `description` - longer description

- `tags` - dictionary of tags (key / value)

## Import

Importing a Security Group resource imports it and the linked
[`exoscale_security_group_rule`](security_group_rule.html).

```shell
# by name
$ terraform import exoscale_security_group.http http

# by id
$ terraform import exoscale_security_group.http eb556678-ec59-4be6-8c54-0406ae0f6da6
```
