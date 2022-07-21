---
page_title: "Exoscale: exoscale_secondary_ipaddress"
subcategory: "Deprecated"
description: |-
  Associate Exoscale Elastic IPs (EIP) to Compute Instances.
---

# exoscale\_secondary\_ipaddress

!> **WARNING:** This resource is **DEPRECATED** and will be removed in the next major version. Please use [exoscale_compute_instance][./compute_instance] `elastic_ip_ids` list instead.


## Arguments Reference

* `compute_id` - (Required) The ID of the compute instance.
* `ip_address` - (Required) The EIP address to assign.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `network_id` - The ID of the network the compute instance NIC is attached to.
* `nic_id` - The ID of the NIC.
