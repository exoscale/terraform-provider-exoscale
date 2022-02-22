---
page_title: "Exoscale: exoscale_private_network"
description: |-
  Provides an Exoscale Private Network.
---

# exoscale\_private\_network

Provides an Exoscale [Private Network][privnet-doc] resource. This can be used to create, update and delete Private Networks.


## Usage

```hcl
resource "exoscale_private_network" "example" {
  zone        = "ch-gva-2"
  name        = "oob"
  description = "Out-of-band network"
}
```

*Managed* Private Network:

```hcl
resource "exoscale_private_network" "example-managed" {
  zone        = "ch-gva-2"
  name        = "oob"
  description = "Out-of-band network with DHCP"
  start_ip    = "10.0.0.20"
  end_ip      = "10.0.0.253"
  netmask     = "255.255.255.0"
}
```


## Arguments Reference

* `zone` - (Required) The name of the [zone][zone] to create the Private Network into.
* `name` - (Required) The name of the Private Network.
* `description` - A description for the Private Network.
* `start_ip` - The first address of IP range used by the DHCP service to automatically assign. Required for *managed* Private Networks.
* `end_ip` - The last address of the IP range used by the DHCP service. Required for *managed* Private Networks.
* `netmask` - The netmask defining the IP network allowed for the static lease. Required for *managed* Private Networks.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the Private Network.


## Import

An existing Private Network can be imported as a resource by specifying `ID@ZONE`:

```console
$ terraform import exoscale_private_network.net 04fb76a2-6d22-49be-8da7-f2a5a0b902e1@ch-gva-2
```


[privnet-doc]: https://community.exoscale.com/documentation/compute/private-networks/
[zone]: https://www.exoscale.com/datacenters/

