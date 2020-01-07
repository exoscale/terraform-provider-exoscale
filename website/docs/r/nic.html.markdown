---
layout: "exoscale"
page_title: "Exoscale: exoscale_nic"
sidebar_current: "docs-exoscale-nic"
description: |-
  Provides an Exoscale Compute instance Private Network Interface (NIC).
---

# exoscale\_nic

Provides an Exoscale Compute instance [Private Network][privnet] Interface (NIC) resource. This can be used to create, update and delete Compute instance NICs.

[privnet]: https://community.exoscale.com/documentation/compute/private-networks/

## Usage

```hcl
resource "exoscale_compute" "vm1" {
  ...
}

resource "exoscale_network" "oob" {
  ...
}

resource "exoscale_nic" "oob" {
  compute_id = exoscale_compute.vm1.id
  network_id = exoscale_network.oob.id
}
```

## Argument Reference

* `compute_id` - (Required) The [Compute instance][compute] ID.
* `network_id` - (Required) The [Private Network][privnet] ID.
* `ip_address` - The IP address to request as static DHCP lease if the NIC is attached to a *managed* Private Network (see the [`exoscale_network`][privnet] resource).

[compute]: compute.html
[privnet]: network.html

## Attributes Reference

The following attributes are exported:

* `mac_address` - The physical address (MAC) of the Compute instance NIC.

## Import

This resource is automatically imported when importing an `exoscale_compute` resource.
