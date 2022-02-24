---
page_title: "Exoscale: exoscale_secondary_ipaddress"
subcategory: "Deprecated"
description: |-
  Provides an Exoscale resource for assigning an existing Elastic IP to a Compute instance.
---

# exoscale\_secondary\_ipaddress

Provides a resource for assigning an existing Exoscale [Elastic IP][r-ipaddress] to a [Compute instance][r-compute].

~> **NOTE:** The network interfaces of the Compute instance itself still have to be configured accordingly (unless using a *managed* Elastic IP).

!> **WARNING:** This resource is deprecated and will be removed in the next major version. Please migrate your [exoscale_compute][r-compute] resources to [exoscale_compute_instance][r-compute-instance], which support attaching elastic IPs directly.

### Secondary IP Address

```hcl
resource "exoscale_compute" "vm1" {
  # ...
}

resource "exoscale_ipaddress" "vip" {
  # ...
}

resource "exoscale_secondary_ipaddress" "vip" {
  compute_id = exoscale_compute.vm1.id
  ip_address = exoscale_ipaddress.vip.ip_address
}
```


## Arguments Reference

* `compute_id` - (Required) The ID of the [Compute instance][r-compute].
* `ip_address` - (Required) The [Elastic IP][r-ipaddress] address to assign.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `nic_id` - The ID of the NIC.
* `network_id` - The ID of the Network the Compute instance NIC is attached to.


## Import

This resource is automatically imported when importing an `exoscale_compute` resource.


[r-compute]: ../resources/compute
[r-compute-instance]: ../resources/compute_instance
[r-ipaddress]: ../resources/ipaddress
