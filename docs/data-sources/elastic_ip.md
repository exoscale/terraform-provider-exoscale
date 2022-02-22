---
page_title: "Exoscale: exoscale_elastic_ip"
description: |-
  Provides information about an Elastic IP.
---

# exoscale\_elastic\_ip

Provides information on an [Elastic IP][eip-doc].


## Example Usage

```hcl
locals {
  zone = "ch-gva-2"
}

data "exoscale_elastic_ip" "vip" {
  zone       = local.zone
  ip_address = "1.2.3.4"
}

data "exoscale_compute_template" "ubuntu" {
  zone = local.zone
  name = "Linux Ubuntu 20.04 LTS 64-bit"
}

resource "exoscale_compute_instance" "example" {
  zone           = local.zone
  name           = "my-instance"
  type           = "standard.medium"
  template_id    = data.exoscale_compute_template.ubuntu.id
  elastic_ip_ids = [data.exoscale_elastic_ip.vip.id]
}
```


## Arguments Reference

* `zone` - (Required) The [zone][zone] of the Elastic IP.
* `id` - The ID of the Elastic IP (conflicts with `ip_address`).
* `ip_address` - The IP address of the Elastic IP (conflicts with `id`).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `description` - The description of the Elastic IP.
* `healthcheck` - A health checking configuration for managed Elastic IPs. Structure is documented below.

The `healthcheck` block contains:

* `mode` - (Required) The health checking mode.
* `port` - (Required) The health checking port.
* `uri` - The health checking URI.
* `interval` - The health checking interval in seconds.
* `timeout` - The time in seconds before considering a healthcheck probing failed.
* `strikes_ok` - The number of successful attempts before considering a managed Elastic IP target healthy.
* `strikes_fail` - The number of failed attempts before considering a managed Elastic IP target unhealthy.
* `tls_sni` - The health checking server name to present with SNI in `https` mode.
* `tls_skip_verify` - Disable TLS certificate verification for health checking in `https` mode.


[eip-doc]: https://community.exoscale.com/documentation/compute/eip/
[zone]: https://www.exoscale.com/datacenters/

