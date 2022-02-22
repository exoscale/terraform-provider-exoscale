---
page_title: "Exoscale: exoscale_network"
subcategory: "Deprecated"
description: |-
  Provides an Exoscale Private Network.
---

# exoscale\_network

Provides an Exoscale [Private Network][privnet-doc] resource. This can be used to create, update and delete Private Networks.

See [`exoscale_nic`][r-nic] for usage with Compute instances.

!> **WARNING:** This resource is deprecated and will be removed in the next major version.


## Usage

```hcl
resource "exoscale_network" "unmanaged" {
  zone             = "ch-gva-2"
  name             = "oob"
  display_text     = "Out-of-band network"

  tags = {
    # ...
  }
}
```

*Managed* Private Network:

```hcl
resource "exoscale_network" "managed" {
  zone             = "ch-gva-2"
  name             = "oob"
  display_text     = "Out-of-band network with DHCP"
  start_ip = "10.0.0.20"
  end_ip   = "10.0.0.253"
  netmask  = "255.255.255.0"
}
```


## Arguments Reference

* `zone` - (Required) The name of the [zone][zone] to create the Private Network into.
* `name` - (Required) The name of the Private Network.
* `display_text` - A free-form text describing the Private Network purpose.
* `start_ip` - The first address of IP range used by the DHCP service to automatically assign. Required for *managed* Private Networks.
* `end_ip` - The last address of the IP range used by the DHCP service. Required for *managed* Private Networks.
* `netmask` - The netmask defining the IP network allowed for the static lease (see `exoscale_nic` resource). Required for *managed* Private Networks.
* `tags` - A dictionary of tags (key/value). To remove all tags, set `tags = {}`.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the Private Network.


## Import

An existing Private Network can be imported as a resource by name or ID:

```console
# By name
$ terraform import exoscale_network.net myprivnet

# By ID
$ terraform import exoscale_network.net 04fb76a2-6d22-49be-8da7-f2a5a0b902e1
```


[r-nic]: ../resources/nic
[privnet-doc]: https://community.exoscale.com/documentation/compute/private-networks/
[zone]: https://www.exoscale.com/datacenters/

