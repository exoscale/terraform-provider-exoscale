---
layout: "exoscale"
page_title: "Exoscale: exoscale_nic"
sidebar_current: "docs-exoscale-nic"
description: |-
  Manages a compute NIC
---

# exoscale_nic


## Usage

```hcl
resource "exoscale_nic" "eth1" {
  compute_id = "${exoscale_compute.mymachine.id}"
  network_id = "${exoscale_network.privNet.id}"
}
```

## Argument Reference

- `compute_id` - (Required) identifier of the compute resource

- `network_id` - (Required) identifier of the private network

- `ip_address` - IP address to use as a static DHCP lease (see the `exoscale_network` resource)

## Attributes Reference

- `mac_address` - physical address of the network interface

## Import

This resource is automatically imported when you import a compute resource.
