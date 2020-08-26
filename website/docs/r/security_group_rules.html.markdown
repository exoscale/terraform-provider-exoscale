---
layout: "exoscale"
page_title: "Exoscale: exoscale_security_group_rules"
sidebar_current: "docs-exoscale-security-group-rules"
description: |-
  Provides a resource for assigning multiple rules to an existing Exoscale Security Group.
---

# exoscale\_security\_group\_rules

Provides a resource for assigning multiple rules to an existing Exoscale [Security Group][r-security_group].


## Example usage

```hcl
resource "exoscale_security_group" "webservers" {
  # ...
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


## Arguments Reference

* `security_group` - (Required) The Security Group name the rules apply to (conflicts with `security_group_id`).
* `security_group_id` - (Required) The Security Group ID the rules apply to (conficts with `security_group)`.
* `ingress`/`egress` - A Security Group rule definition.

`ingress`/`egress`:

* `protocol` - (Required) The network protocol to match. Supported values are: `TCP`, `UDP`, `ICMP`, `ICMPv6`, `AH`, `ESP`, `GRE`, `IPIP` and `ALL`.
* `description` - A free-form text describing the Security Group rule purpose.
* `ports` - A list of ports or port ranges (`start_port-end_port`).
* `icmp_type`/`icmp_code` - An ICMP/ICMPv6 [type/code][icmp] to match.
* `cidr_list` - A list of source (for ingress)/destination (for egress) IP subnet (in [CIDR notation][cidr]) to match.
* `user_security_group_list` - A source (for ingress)/destination (for egress) of the traffic identified by a Security Group.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* n/a


[cidr]: https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing#CIDR_notation
[icmp]: https://en.wikipedia.org/wiki/Internet_Control_Message_Protocol#Control_messages
[r-security_group]: security_group.html

