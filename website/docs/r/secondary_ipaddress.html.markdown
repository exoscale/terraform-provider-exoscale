---
layout: "exoscale"
page_title: "Exoscale: exoscale_secondary_ipaddress"
sidebar_current: "docs-exoscale-secondary-ipaddress"
description: |-
  Provides an Exoscale resource for assigning an existing Elastic IP to a Compute instance.
---

# exoscale\_secondary\_ipaddress

Provides a resource for assigning an existing Exoscale [Elastic IP][eip] to a [Compute instance][compute].

~> **NOTE:** The network interfaces of the Compute instance itself still have to be configured accordingly (unless using a *managed* Elastic IP).

[eip]: ipaddress.html
[compute]: compute.html

### Secondary IP Address

```hcl
resource "exoscale_compute" "vm1" {
  ...
}

resource "exoscale_ipaddress" "vip" {
  ...
}

resource "exoscale_secondary_ipaddress" "vip" {
  compute_id = "${exoscale_compute.vm1.id}"
  ip_address = "${exoscale_ipaddress.vip.ip_address}"
}
```

## Argument Reference

* `compute_id` - (Required) The ID of the [Compute instance][compute].
* `ip_address` - (Required) The [Elastic IP][eip] address to assign.

[compute]: compute.html
[eip]: ip_address.html

## Attributes Reference

The following attributes are exported:

* `nic_id` - The ID of the NIC.
* `network_id` - The ID of the Network the Compute instance NIC is attached to.

## Import

This resource is automatically imported when importing an `exoscale_compute` resource.
