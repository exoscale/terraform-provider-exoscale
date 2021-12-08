---
layout: "exoscale"
page_title: "Exoscale: exoscale_elastic_ip"
sidebar_current: "docs-exoscale-elastic_ip"
description: |-
  Provides an Exoscale Elastic IP address.
---

# exoscale\_elastic_ip

Provides an Exoscale [Elastic IP address][eip-doc] resource. This can be used to create, update and delete Elastic IPs.


### Usage example

```hcl
resource "exoscale_elastic_ip" "example" {
  zone = "ch-dk-2"
}
```

Managed EIP:

```hcl
resource "exoscale_elastic_ip" "example-managed" {
  zone        = "ch-dk-2"
  description = "My elastic IP for load balancer"
  
  healthcheck {
    mode         = "https"
    port         = 443
    uri         = "/health"
    interval     = 5
    timeout      = 3
    strikes_ok   = 2
    strikes_fail = 3
    tls_sni      = "example.net"
  }
}
```


## Arguments Reference

* `zone` - (Required) The name of the [zone][zone] to create the Elastic IP into.
* `description` - The description of the Elastic IP.
* `healthcheck` - A health checking configuration for managed Elastic IPs. Structure is documented below.

The `healthcheck` block supports:

* `mode` - (Required) The health checking mode (`supported values: `tcp`, `http`, `https`).
* `port` - (Required) The health checking port (must be between `1` and `65535`).
* `uri` - The health checking URI (required in `http(s)` modes).
* `interval` - The health checking interval in seconds (must be between `5` and `300`; default: `10`).
* `timeout` - The time in seconds before considering a healthcheck probing failed (must be between `2` and `60`; default: `3`).
* `strikes_ok` - The number of successful attempts before considering a managed Elastic IP target healthy (must be between `1` and `20`).
* `strikes_fail` - The number of failed attempts before considering a managed Elastic IP target unhealthy (must be between `1` and `20`).
* `tls_sni` - The health checking server name to present with SNI in `https` mode.
* `tls_skip_verify` - Disable TLS certificate verification for health checking in `https` mode.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `ip_address` - The Elastic IP address.


## Import

An existing Elastic IP can be imported as a resource by specifying `ID@ZONE`:

```console
$ terraform import exoscale_elastic_ip.web eb556678-ec59-4be6-8c54-0406ae0f6da6@ch-dk-2
```


[eip-doc]: https://community.exoscale.com/documentation/compute/eip/
[zone]: https://www.exoscale.com/datacenters/
