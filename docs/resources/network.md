---
page_title: "Exoscale: exoscale_network"
subcategory: "Deprecated"
description: |-
  Manage Exoscale Private Networks.
---

# exoscale\_network

!> **WARNING:** This resource is **DEPRECATED** and will be removed in the next major version. Please use [exoscale_private_network](./private_network.md) instead.


## Arguments Reference

* `zone` - (Required) The Exoscale Zone name.
* `name` - (Required) The private network name.

* `display_text` - A free-form text describing the network.
* `netmask` - The network mask defining the IP network allowed for static leases (see `exoscale_nic` resource). Required for *managed* private networks.
* `start_ip`/`end_ip` - The first/last IP addresses used by the DHCP service for dynamic leases. Required for *managed* private networks.
* `tags` - A dictionary of tags (key/value). To remove all tags, set `tags = {}`.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The private network ID.
