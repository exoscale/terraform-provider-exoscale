---
layout: "exoscale"
page_title: "Exoscale: exoscale_ipaddress"
sidebar_current: "docs-exoscale-ipaddress"
description: |-
  Provides an Exoscale Elastic IP address.
---

# exoscale\_ipaddress

Provides an Exoscale [Elastic IP][eip] resource. This can be used to create, update and delete Elastic IPs.

See [`exoscale_secondary_ipaddress`][secip] for usage with Compute instances.

[eip]: https://community.exoscale.com/documentation/compute/eip/
[secip]: secondary_ipaddress.html

### Usage example

```hcl
resource "exoscale_ipaddress" "myip" {
  zone = "ch-dk-2"
  tags = {
    usage = "load-balancer"
  }
}
```

Managed EIP:

```hcl
resource "exoscale_ipaddress" "myip" {
  zone                     = "ch-dk-2"
  description              = "My elastic IP for load balancer"
  healthcheck_mode         = "http"
  healthcheck_port         = 8000
  healthcheck_path         = "/status"
  healthcheck_interval     = 5
  healthcheck_timeout      = 2
  healthcheck_strikes_ok   = 2
  healthcheck_strikes_fail = 3
}
```

## Argument Reference

* `zone` - (Required) The name of the [zone][zone] to create the Elastic IP into.
* `description` - The description of the Elastic IP.
* `healthcheck_mode` - The healthcheck probing mode (must be either `tcp` or `http`).
* `healthcheck_port` - The healthcheck service port to probe (must be between `1` and `65535`).
* `healthcheck_path` - The healthcheck probe HTTP request path (must be specified in `http` mode).
* `healthcheck_interval` - The healthcheck probing interval in seconds (must be between `5` and `300`).
* `healthcheck_timeout` - The time in seconds before considering a healthcheck probing failed (must be between `2` and `60`).
* `healthcheck_strikes_ok` - The number of successful healthcheck probes before considering the target healthy (must be between `1` and `20`).
* `healthcheck_strikes_fail` - The number of unsuccessful healthcheck probes before considering the target unhealthy (must be between `1` and `20`).
* `tags` - A dictionary of tags (key/value).

[zone]: https://www.exoscale.com/datacenters/

## Attributes Reference

The following attributes are exported:

* `ip_address` - The Elastic IP address.

## Import

An existing Elastic IP can be imported as a resource by address or ID:

```console
# By address
$ terraform import exoscale_ipaddress.myip 159.100.251.224

# By ID
$ terraform import exoscale_ipaddress.myip eb556678-ec59-4be6-8c54-0406ae0f6da6
```
