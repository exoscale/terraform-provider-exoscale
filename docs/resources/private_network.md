---
page_title: "Exoscale: exoscale_private_network"
description: |-
  Manage Exoscale Private Networks.
---

# exoscale\_private\_network

Manage Exoscale [Private Networks](https://community.exoscale.com/documentation/compute/private-networks/).


## Usage

*Unmanaged* private network:

```hcl
resource "exoscale_private_network" "my_private_network" {
  zone = "ch-gva-2"
  name = "my-private-network"
}
```

*Managed* private network:

```hcl
resource "exoscale_private_network" "my_managed_private_network" {
  zone = "ch-gva-2"
  name = "my-managed-private-network"

  netmask  = "255.255.255.0"
  start_ip = "10.0.0.20"
  end_ip   = "10.0.0.253"
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/

* `zone` - (Required) The Exoscale [Zone][zone] name.
* `name` - (Required) The private network name.

* `description` - A free-form text describing the network.

### For *Managed* Private Networks

In addition to the arguments listed above:

* `netmask` - (Required) The network mask defining the IPv4 network allowed for static leases.
* `start_ip`/`end_ip` - (Required) The first/last IPv4 addresses used by the DHCP service for dynamic leases.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The private network ID.


## Import

An existing private network may be imported by `<ID>@<zone>`:

```console
$ terraform import \
  exoscale_private_network.my_private_network \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6@ch-gva-2
```
