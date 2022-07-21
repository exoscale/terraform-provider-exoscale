---
page_title: "Exoscale: exoscale_network"
subcategory: "Deprecated"
description: |-
  Manage Exoscale Private Networks.
---

# exoscale\_network

!> **WARNING:** This resource is **DEPRECATED** and will be removed in the next major version. Please use [exoscale_private_network](./private_network) instead.


## Arguments Reference

* `zone` - (Required) The name of the zone to create the private network into.
* `name` - (Required) The name of the private network.

* `display_text` - A free-form text describing the private network.
* `netmask` - The network mask defining the IP network allowed for static leases (see `exoscale_nic` resource). Required for *managed* private networks.
* `start_ip`/`end_ip` - The first/last IP addresses used by the DHCP service for dynamic leases. Required for *managed* private networks.
* `tags` - A dictionary of tags (key/value). To remove all tags, set `tags = {}`.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the private network.
