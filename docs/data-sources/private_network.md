---
page_title: "Exoscale: exoscale_private_network"
description: |-
  Fetch Exoscale Private Networks data.
---

# exoscale\_private\_network

Fetch Exoscale [Private Networks](https://community.exoscale.com/documentation/compute/private-networks/) data.

Corresponding resource: [exoscale_private_network](../resources/private_network.md).


## Usage

```hcl
data "exoscale_private_network" "my_private_network" {
  zone = "ch-gva-2"
  name = "my-private-network"
}

output "my_private_network_id" {
  value = data.exoscale_private_network.my_private_network.id
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/

* `zone` - (Required) The Exoscale [Zone][zone] name.

* `id` - The private network ID to match (conflicts with `name`).
* `name` - The network name to match (conflicts with `id`).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `description` - The private network description.

### For *Managed* Private Networks

* `netmask` - The network mask defining the IPv4 network allowed for static leases.
* `start_ip`/`end_ip` - The first/last IPv4 addresses used by the DHCP service for dynamic leases.
