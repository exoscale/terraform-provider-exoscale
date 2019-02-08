---
layout: "exoscale"
page_title: "Exoscale: exoscale_security_group_rule"
sidebar_current: "docs-exoscale-security-group_rule"
description: |-
  Manages a rule to a security group.
---

# exoscale_security_group_rule

A security group rule represents a single `ingress` or `egress` rule belonging
to a `exoscale_security_group`.

## Example usage

```hcl
resource "exoscale_security_group_rule" "http" {
  security_group_id = "${exoscale_security_group.http.id}"
  protocol = "TCP"
  type = "INGRESS"
  cidr = "0.0.0.0/0"  # "::/0" for IPv6
  start_port = 80
  end_port = 80
}
```

## Argument Reference

- `security_group_id` - (Required) which security group by name the rule applies to

- `security_group` - (Required) which security group by id the rule applies to

- `protocol` - (Required) the protocol, e.g. `TCP`, `UDP`, `ICMP`, `ICMPv6`, ... or `ALL`

- `type` - (Required) traffic type, either `INGRESS` or `EGRESS`

- `description` - human description

- `start_port` and `end_port` - for `TCP`, `UDP` traffic

- `icmp_type` and `icmp_code` - for `ICMP`, `ICMPv6` traffic

- `cidr` - source/destination of the traffic as an IP subnet (conflicts with `user_security_group`)

- `user_security_group_id` - source/destination of the traffic identified by a security group by id (conflicts with `cidr`)

- `user_security_group` - source/destination of the traffic identified by a security group by name (conflicts with `cidr`)

## Attributes Reference

- `security_group` - Name of the security group

- `security_group_id` - Identifier of the security group

- `user_security_group` - Name of the source/destination security group

- `user_security_group_id` - Identifer of the source/destination security group
