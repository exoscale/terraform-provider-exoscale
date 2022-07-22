---
page_title: "Exoscale: exoscale_security_group_rule"
description: |-
  Manage Exoscale Security Group Rules.
---

# exoscale\_security\_group\_rule

Manage Exoscale [Security Group](https://community.exoscale.com/documentation/compute/security-groups/) Rules.


## Usage

```hcl
resource "exoscale_security_group" "my_security_group" {
  name = "my-security-group"
}

resource "exoscale_security_group_rule" "my_security_group_rule" {
  security_group_id = exoscale_security_group.my_security_group.id
  type              = "INGRESS"
  protocol          = "TCP"
  cidr              = "0.0.0.0/0" # "::/0" for IPv6
  start_port        = 80
  end_port          = 80
}
```


## Arguments Reference

[cidr]: https://en.wikipedia.org/wiki/Classless_Inter-Domain_Routing#CIDR_notation
[icmp]: https://en.wikipedia.org/wiki/Internet_Control_Message_Protocol#Control_messages

* `security_group_id` - (Required) The parent [security group](./security_group.md) ID.
* `type` - (Required) The traffic direction to match (`INGRESS` or `EGRESS`).
* `protocol` - (Required) The network protocol to match (`TCP`, `UDP`, `ICMP`, `ICMPv6`, `AH`, `ESP`, `GRE`, `IPIP` or `ALL`)

* `description` - A free-form text describing the security group rule.
* `cidr` - An (`INGRESS`) source / (`EGRESS`) destination IP subnet (in [CIDR notation][cidr]) to match (conflicts with `user_security_group`/`user_security_group_id`).
* `start_port`/`end_port` - A `TCP`/`UDP` port range to match.
* `icmp_type`/`icmp_code` - An ICMP/ICMPv6 [type/code][icmp] to match.
* `user_security_group_id` - An (`INGRESS`) source / (`EGRESS`) destination security group ID to match (conflicts with `cidr`/`user_security_group)`).

* `security_group` - (Deprecated) The parent security group name. Please use the `security_group_id` argument along the [exoscale_security_group](../data-sources/security_group.md) data source instead.
* `user_security_group` - (Deprecated) An (`INGRESS`) source / (`EGRESS`) destination security group name to match (conflicts with `cidr`/`user_security_group_id`). Please use the `user_security_group_id` argument along the [exoscale_security_group](../data-sources/security_group.md) data source instead.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the security group rule.


## Import

An existing security group rule may be imported by `<security-group-ID>/<security-group-rule-ID>`:

```console
$ terraform import \
  exoscale_security_group_rule.my_security_group_rule \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6/9ecc6b8b-73d4-4211-8ced-f7f29bb79524
```

~> **NOTE:** This resource is automatically imported when importing an `exoscale_security_group` resource.
