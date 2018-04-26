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
  zone = "ch-dk-2"
  network_offering = "PrivNet"

  tags {
    # ...
  }
}
```

## Argument Reference

- `name` - (Required) name of the network

- `display_text` - (Required) description of the network

- `network_offering` - (Required) network offering name

- `zone` - (Required) name of the zone

- `tags` - dictionary of tags (key / value)


## Import

```shell
# by name
$ terraform import exoscale_network.net myPrivNet

# by id
$ terraform import exoscale_network.net ID
```
