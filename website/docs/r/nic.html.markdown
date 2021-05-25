---
layout: "exoscale"
page_title: "Exoscale: exoscale_nic"
sidebar_current: "docs-exoscale-nic"
description: |-
  Provides an Exoscale Compute instance Private Network Interface (NIC).
---

# exoscale\_nic

Provides an Exoscale Compute instance [Private Network][privnet-doc] Interface (NIC) resource. This can be used to create, update and delete Compute instance NICs.


## Usage

```hcl
resource "exoscale_compute" "vm1" {
  # ...
}

resource "exoscale_network" "oob" {
  # ...
}

resource "exoscale_nic" "oob" {
  compute_id = exoscale_compute.vm1.id
  network_id = exoscale_network.oob.id
}
```


## Arguments Reference

* `compute_id` - (Required) The [Compute instance][r-compute] ID.
* `network_id` - (Required) The [Private Network][r-network] ID.
* `ip_address` - The IP address to request as static DHCP lease if the NIC is attached to a *managed* Private Network (see the [`exoscale_network`][r-network] resource).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the Compute instance NIC.
* `mac_address` - The physical address (MAC) of the Compute instance NIC.


## Import

This resource is automatically imported when importing an `exoscale_compute` resource.


[privnet-doc]: https://community.exoscale.com/documentation/compute/private-networks/
[r-compute]: compute.html
[r-network]: network.html
