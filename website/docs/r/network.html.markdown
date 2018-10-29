---
layout: "exoscale"
page_title: "Exoscale: exoscale_network"
sidebar_current: "docs-exoscale-network"
description: |-
  Manages a private network.
---

# exoscale_network


## Usage

```hcl
resource "exoscale_network" "privNet" {
  name = "myPrivNet"
  display_text = "description"
  zone = "ch-gva-2"
  network_offering = "PrivNet"

  // Optional and only available at zone: CH-GVA-2
  start_ip = "10.0.0.20"
  end_ip = "10.0.0.254"
  netmask = "255.255.255.0"

  tags {
    # ...
  }
}
```

## Argument Reference

- `name` - (Required) name of the network

- `display_text` - description of the network

- `network_offering` - (Required) network offering name

- `zone` - (Required) name of the zone

- `start_ip` - First IP address of IP range used by the DHCP service to automatically assign

- `end_ip` - Last IP address of the IP range used by the DHCP service

- `netmask` - Netmask defining the IP network allowed for the static lease (see `exoscale_nic` resource)

- `tags` - dictionary of tags (key / value)


## Import

```shell
# by name
$ terraform import exoscale_network.net myPrivNet

# by id
$ terraform import exoscale_network.net ID
```
