---
layout: "exoscale"
page_title: "Exoscale: exoscale_security_group_rules"
sidebar_current: "docs-exoscale-security-group-rules"
description: |-
  Provides a resource for assigning multiple rules to an existing Exoscale Security Group.
---

# exoscale\_security\_group\_rules

Provides a resource for assigning multiple rules to an existing Exoscale [Security Group][sg].

[sg]: security_group.html

## Example usage

```hcl
resource "exoscale_security_group" "webservers" {
  ...
}

resource "exoscale_security_group_rules" "admin" {
  security_group = exoscale_security_group.webservers.name

  ingress {
    protocol                 = "ICMP"
    icmp_type                = 8
    user_security_group_list = ["bastion"]
  }

  ingress {
    protocol                 = "TCP"
    ports                    = ["22"]
    user_security_group_list = ["bastion"]
  }
}

resource "exoscale_security_group_rules" "web" {
  security_group_id = exoscale_security_group.webservers.id

  ingress {
    protocol  = "TCP"
    ports     = ["80", "443"]
    cidr_list = ["0.0.0.0/0", "::/0"]
  }
}
```

## Argument Reference

The following attributes are exported:

* `security_group` - (Required) The Security Group name the rules apply to.
* `security_group_id` - (Required) The Security Group ID the rules apply to.

`egress` and `ingress` support the following:

* `protocol` - (Required) The network protocol to match. Supported values are: `TCP`, `UDP`, `ICMP`, `ICMPv6`, `AH`, `ESP`, `GRE`, `IPIP` and `ALL`.
* `description` - A free-form text describing the Security Group Rule purpose.
* `ports` - A list of ports or port ranges (`start_port-end_port`).
* `icmp_type`/`icmp_code` - An `ICMP`/`ICMPv6` [type/code][icmp] to match.
* `cidr_list` - A list of source (for ingress)/destination (for egress) IP subnet to match (conflicts with `user_security_group`).
* `user_security_group_list` - A source (for ingress)/destination (for egress) of the traffic identified by a security group

[icmp]: https://en.wikipedia.org/wiki/Internet_Control_Message_Protocol#Control_messages

## Attributes Reference

The following attributes are exported:

* `security_group` - The name of the Security Group the rules apply to.
* `security_group_id` - The ID of the Security Group the rules apply to.
