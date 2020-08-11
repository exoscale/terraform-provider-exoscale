---
layout: "exoscale"
page_title: "Exoscale: exoscale_compute_ipaddress"
sidebar_current: "docs-exoscale-compute-ipaddress"
description: |-
  Provides information about a Compute template.
---

# exoscale\_compute\_ipaddress

Provides information on an Compute [Elastic IP address][eip-doc].


## Example Usage

```hcl
locals {
  zone = "ch-gva-2"
}

data "exoscale_compute_ipaddress" "eip" {
  zone = local.zone
  ip_address = "159.162.3.4"
}
```


## Arguments Reference

* `zone` - (Required) The name of the [zone][zone] where to look for the IP Address.
* `ip_address` - The IP Address of the EIP.
* `id` - The ID of the IP Address.
* `description` - The Description to find the IP Address.
* `tags` - The tags to find the IP Address.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* n/a


[eip-doc]: https://community.exoscale.com/documentation/compute/eip/
[zone]: https://www.exoscale.com/datacenters/
