---
page_title: "Exoscale: exoscale_compute_ipaddress"
subcategory: "Deprecated"
description: |-
  Fetch Exoscale Elastic IPs (EIP) data.
---

# exoscale\_compute\_ipaddress

!> **WARNING:** This data source is **DEPRECATED** and will be removed in the next major version. Please use [exoscale_elastic_ip](./elastic_ip.md) instead.


## Arguments Reference

* `zone` - (Required) The name of the zone of the EIP.

* `id` - The EIP ID to match.
* `description` - The EIP description to match.
* `ip_address` - The EIP IPv4 address to match.
* `tags` - The EIP tags to match.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* n/a
