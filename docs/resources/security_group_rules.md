---
page_title: "Exoscale: exoscale_security_group_rules"
subcategory: "Deprecated"
description: |-
  Manage Exoscale Security Group Rules.
---

# exoscale\_security\_group\_rules

!> **WARNING:** This resource is **DEPRECATED** and will be removed in the next major version. Please use the [exoscale_security_group_rule](./security_group_rule) instead (or refer to the ad-hoc [migration guide](../guides/migration-of-security-group-rules)).


## Arguments Reference

* `security_group` - (Required) The security group name the rules apply to (conflicts with `security_group_id`).
* `security_group_id` - (Required) The security group ID the rules apply to (conficts with `security_group)`.

* `ingress`/`egress` - (Block) A security group rule definition (can be specified multiple times).

### `ingress`/`egress` block

* `protocol` - (Required) The network protocol to match (`TCP`, `UDP`, `ICMP`, `ICMPv6`, `AH`, `ESP`, `GRE`, `IPIP` or `ALL`).

* `description` - A free-form text describing the security group rules.
* `icmp_type`/`icmp_code` - An ICMP/ICMPv6 type/code to match.

* `cidr_list` - A list of (`INGRESS`) source / (`EGRESS`) destination IP subnet (in CIDR notation) to match.
* `ports` - A list of ports or port ranges (`<start_port>-<end_port>`).
* `user_security_group_list` - A list of source (for ingress)/destination (for egress) identified by a security group.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* n/a
