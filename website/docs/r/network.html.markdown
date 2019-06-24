---
layout: "exoscale"
page_title: "Exoscale: exoscale_network"
sidebar_current: "docs-exoscale-network"
description: |-
  Provides an Exoscale Private Network.
---

# exoscale\_network

Provides an Exoscale [Private Network][privnet] resource. This can be used to create, update and delete Private Networks.

See [`exoscale_nic`][nic] for usage with Compute instances.

[privnet]: https://community.exoscale.com/documentation/compute/private-networks/
[nic]: nic.html

## Usage

```hcl
resource "exoscale_network" "unmanaged" {
  zone             = "ch-gva-2"
  name             = "oob"
  display_text     = "Out-of-band network"
  network_offering = "PrivNet"

  tags = {
    ...
  }
}
```

*Managed* Private Network (~> **NOTE:** this feature is currently only available in the `ch-gva-2` zone):

```hcl
resource "exoscale_network" "managed" {
  zone             = "ch-gva-2"
  name             = "oob"
  display_text     = "Out-of-band network with DHCP"
  network_offering = "PrivNet"

  start_ip = "10.0.0.20"
  end_ip   = "10.0.0.254"
  netmask  = "255.255.255.0"
}
```

## Argument Reference

* `zone` - (Required) The name of the [zone][zone] to create the Private Network into.
* `name` - (Required) The name of the Private Network.
* `display_text` - A free-form text describing the Private Network purpose.
* `network_offering` - (Required) The Private Nnetwork offering name (`PrivNet` is the only supported value).
* `start_ip` - The first address of IP range used by the DHCP service to automatically assign. Required for *managed* Private Networks.
* `end_ip` - The last address of the IP range used by the DHCP service. Required for *managed* Private Networks.
* `netmask` - The netmask defining the IP network allowed for the static lease (see `exoscale_nic` resource). Required for *managed* Private Networks.
* `tags` - A dictionary of tags (key/value).

[zone]: https://www.exoscale.com/datacenters/

## Import

An existing Private Network can be imported as a resource by name or ID:

```console
# By name
$ terraform import exoscale_network.net myprivnet

# By ID
$ terraform import exoscale_network.net 04fb76a2-6d22-49be-8da7-f2a5a0b902e1
```
