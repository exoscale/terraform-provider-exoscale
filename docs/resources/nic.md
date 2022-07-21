---
page_title: "Exoscale: exoscale_nic"
subcategory: "Deprecated"
description: |-
  Manage Exoscale Compute Instance Private Network Interfaces (NIC).
---

# exoscale\_nic

!> **WARNING:** This resource is **DEPRECATED** and will be removed in the next major version. Please use [exoscale_compute_instance][./compute_instance] `network_interface` block instead.


## Arguments Reference

* `compute_id` - (Required) The compute instance ID.
* `network_id` - (Required) The private network ID.

* `ip_address` - The IP address to request as static DHCP lease if the NIC is attached to a *managed* private network.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the compute instance NIC.
* `mac_address` - The physical address (MAC) of the compute instance NIC.
