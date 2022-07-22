---
page_title: "Exoscale: exoscale_network"
subcategory: "Deprecated"
description: |-
  Fetch Exoscale Private Networks data.
---

# exoscale\_network

!> **WARNING:** This data source is **DEPRECATED** and will be removed in the next major version. Please use [exoscale_private_network](./private_network.md) instead.


## Arguments Reference

* `zone` - (Required) The Exoscale Zone name.

* `id` - The private network ID to match (conflicts with `name`).
* `name` - The network name to match (conflicts with `id`).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `description` - The private network description.

### For *Managed* Private Networks

* `netmask` - The network mask defining the IPv4 network allowed for static leases.
* `start_ip`/`end_ip` - The first/last IPv4 addresses used by the DHCP service for dynamic leases.
