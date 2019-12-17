---
layout: "exoscale"
page_title: "Exoscale: exoscale_compute_ipaddress"
sidebar_current: "docs-exoscale-compute-ipaddress"
description: |-
  Provides information about a Compute template.
---

# exoscale\_compute\_template

Provides information on an Compute [IP Address][ip].

[ip]: https://community.exoscale.com/documentation/compute/eip/

## Example Usage

```hcl
locals {
  zone = "ch-gva-2"
}

data "exoscale_compute_ipaddress" "eip" {
  zone = "${local.zone}"
  ip_address = "159.162.3.4"
}
```

## Argument Reference

* `zone` - (Required) The name of the [zone][zone] where to look for the IP Address.
* `ip_address` - The IP Address of the EIP.
* `id` - The ID of the IP Address.
* `description` - The Description to find the IP Address.
* `tags` - The tags to find the IP Address.

[zone]: https://www.exoscale.com/datacenters/

## Attributes Reference

The following attributes are exported:

* `zone` - Name of the zone.
* `ip_address` - IP Address.
* `id` - ID of the IP Address.
* `description` - Description of the IP.
* `tags` - Map of tags (key: value).
