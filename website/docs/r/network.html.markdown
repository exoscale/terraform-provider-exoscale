---
layout: "exoscale"
page_title: "Exoscale: exoscale_network"
sidebar_current: "docs-exoscale-network"
description: |-
  Manages a private network.
---

# exoscale_network

The `exoscale_network` resource manages a [Private Network][privnet], a
virtual L2 network segment shared only among Compute instances attached to it.

[privnet]: https://community.exoscale.com/documentation/compute/private-networks/

## Usage

```hcl
resource "exoscale_network" "unmanaged" {
  name = "myPrivNet"
  display_text = "description"
  zone = "ch-gva-2"
  network_offering = "PrivNet"

  tags = {
    # ...
  }
}
```

*Managed* Private Network (note: this feature is currently only available in
the `CH-GVA-2` zone):

```hcl
resource "exoscale_network" "managed" {
  name = "myPrivNet"
  display_text = "description"
  zone = "ch-gva-2"
  network_offering = "PrivNet"

  start_ip = "10.0.0.20"
  end_ip = "10.0.0.254"
  netmask = "255.255.255.0"
}
```

## Argument Reference

- `zone` - (Required) name of the zone

- `name` - (Required) name of the network

- `network_offering` - (Required) network offering name

- `display_text` - Description of the network

- `start_ip` - First address of IP range used by the DHCP service to automatically assign.
  Required for *managed* Private Networks.

- `end_ip` - Last address of the IP range used by the DHCP service.
  Required for *managed* Private Networks.

- `netmask` - Netmask defining the IP network allowed for the static lease (see `exoscale_nic` resource).
  Required for *managed* Private Networks.

- `tags` - dictionary of tags (key/value)


## Import

```shell
# by name
$ terraform import exoscale_network.net myPrivNet

# by id
$ terraform import exoscale_network.net ID
```
