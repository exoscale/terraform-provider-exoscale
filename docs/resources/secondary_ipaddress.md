---
page_title: "Exoscale: exoscale_secondary_ipaddress"
subcategory: "Deprecated"
description: |-
  Associate Exoscale Elastic IPs (EIP) to Compute Instances.
---

# exoscale\_secondary\_ipaddress

!> **WARNING:** This resource is **DEPRECATED** and will be removed in the next major version. Please use [exoscale_compute_instance](./compute_instance.md) `elastic_ip_ids` list instead.


## Arguments Reference

* `compute_id` - (Required) The compute instance ID.
* `ip_address` - (Required) The Elastic IP (EIP) address.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `network_id` - The network (ID) the compute instance NIC is attached to.
* `nic_id` - The network interface (NIC) ID.
