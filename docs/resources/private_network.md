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

Please refer to the [examples](../../examples/) directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/

* `zone` - (Required) The name of the [zone][zone] to create the private network into.
* `name` - (Required) The name of the private network.

* `description` - A free-form text describing the private network.
* `netmask` - The network mask defining the IPv4 network allowed for static leases (required for *managed* private networks).
* `start_ip`/`end_ip` - The first/last IPv4 addresses used by the DHCP service for dynamic leases (required for *managed* private networks).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the private network.


## Import

An existing private network may be imported by `<ID>@<zone>`:

```console
$ terraform import \
  exoscale_private_network.my_private_network \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6@ch-gva-2
```
