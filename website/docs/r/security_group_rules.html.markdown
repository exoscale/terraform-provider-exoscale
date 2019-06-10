---
layout: "exoscale"
page_title: "Exoscale: exoscale_security_group_rules"
sidebar_current: "docs-exoscale-security-group-rules"
description: |-
  Manages a set of rules to a security group.
---

# exoscale_security_group_rules

A security group rules represents a set of `ingress` and/or `egress` rules
which has to be linked to a `exoscale_security_group`.

## Example usage

```hcl
resource "exoscale_security_group_rules" "http" {
  security_group_id = "${exoscale_security_group.http.id}"

  ingress {
    protocol = "TCP"
    cidr_list = ["0.0.0.0/0", "::/0"]
    ports = ["80", "8000-8888"]
    user_security_group_list = ["default", "etcd"]
  }

  egress {
    // ...
  }
}
```

## Argument Reference

- `security_group_id` - (Required) which security group by name the rule applies to

- `security_group` - (Required) which security group by id the rule applies to

- `egress` or `ingress` - set of rules for the incoming or outgoing traffic

    - `protocol` - (Required) the protocol, e.g. `TCP`, `UDP`, `ICMP`, `ICMPv6`, .., or `ALL`

    - `description` - human description

    - `ports` - a set of port ranges

    - `icmp_type` and `icmp_code` - for `ICMP`, `ICMPv6` traffic

    - `cidr_list` - source/destination of the traffic as an IP subnet

    - `user_security_group_list` - source/destination of the traffic identified by a security group

## Attributes Reference

- `security_group` - Name of the security group

- `security_group_id` - Identifier of the security group
