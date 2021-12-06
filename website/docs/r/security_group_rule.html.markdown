---
layout: "exoscale"
page_title: "Exoscale: exoscale_security_group_rule"
sidebar_current: "docs-exoscale-security-group-rule"
description: |-
  Provides an Exoscale Security Group Rule.
---

# exoscale\_security\_group\_rule

Provides an Exoscale [Security Group][r-security_group] rule resource. This can be used to create and delete Security Group rules.


## Example usage

```hcl
resource "exoscale_security_group" "webservers" {
  # ...
}

resource "exoscale_security_group_rule" "http" {
  security_group_id = exoscale_security_group.webservers.id
  type              = "INGRESS"
  protocol          = "TCP"
  cidr              = "0.0.0.0/0" # "::/0" for IPv6
  start_port        = 80
  end_port          = 80
}
```


## Arguments Reference

* `security_group` - (Required) The Security Group name the rule applies to.
* `security_group_id` - (Required) The Security Group ID the rule applies to.
* `type` - (Required) The traffic direction to match (`INGRESS` or `EGRESS`).
* `protocol` - (Required) The network protocol to match. Supported values are: `TCP`, `UDP`, `ICMP`, `ICMPv6`, `AH`, `ESP`, `GRE`, `IPIP` and `ALL`.
* `description` - A free-form text describing the Security Group rule purpose.
* `start_port`/`end_port` - A `TCP`/`UDP` port range to match.
* `icmp_type`/`icmp_code` - An ICMP/ICMPv6 [type/code][icmp] to match.
* `cidr` - A source (for ingress)/destination (for egress) IP subnet (in [CIDR notation][cidr]) to match (conflicts with `user_security_group`/`security_group_id`).
* `user_security_group_id` - A source (for ingress)/destination (for egress) Security Group ID to match (conflicts with `cidr`/`security_group)`).
* `user_security_group` - A source (for ingress)/destination (for egress) Security Group name to match (conflicts with `cidr`/`security_group_id`).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the Security Group rule.


## Import

An existing Security Group rule can be imported as a resource by `<SECURITY-GROUP-ID>/<SECURITY-GROUP-RULE-ID>`:

```console
$ terraform import exoscale_security_group_rule.http eb556678-ec59-4be6-8c54-0406ae0f6da6/846831cb-a0fc-454b-9abd-cb526559fcf9
```

~> **NOTE:** This resource is automatically imported when importing an `exoscale_security_group` resource.


[cidr]: https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing#CIDR_notation
[icmp]: https://en.wikipedia.org/wiki/Internet_Control_Message_Protocol#Control_messages
[r-security_group]: security_group.html
