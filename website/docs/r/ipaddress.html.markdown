---
layout: "exoscale"
page_title: "Exoscale: exoscale_ipaddress"
sidebar_current: "docs-exoscale-ipaddress"
description: |-
  Manages an elastic IP address.
---

# exoscale_ipaddress

The [Elastic IP][eip] (EIP) address is an address that belongs to a specific zone and may be
associated to one or many compute instances. See [secondary_ipaddress](secondary_ipaddress.html).
To provision a *managed* Elastic IP, use the `healthcheck_*` attributes.

### Usage example

```
resource "exoscale_ipaddress" "myip" {
  zone = "ch-dk-2"
  tags = {
    usage = "load-balancer"
  }
}
```

Managed EIP:

```
resource "exoscale_ipaddress" "myip" {
  zone                     = "ch-dk-2"
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

- `zone` - (Required) name of the [zone](https://www.exoscale.com/datacenters/)
- `healthcheck_mode` - healthcheck probing mode (must be either `tcp` or `http`)
- `healthcheck_port` - healthcheck service port to probe (must be between `1` and `65535`)
- `healthcheck_path` - healthcheck probe HTTP request path (must be specified in `http` mode)
- `healthcheck_interval` - healthcheck probing interval in seconds (must be between `5` and `300`)
- `healthcheck_timeout` - time in seconds before considering a healthcheck probing failed (must be between `2` and `60`)
- `healthcheck_strikes_ok` - number of successful healthcheck probes before considering the target healthy (must be between `1` and `20`)
- `healthcheck_strikes_fail` - number of unsuccessful healthcheck probes before considering the target unhealthy (must be between `1` and `20`)
- `tags` - dictionary of tags (key/value)

## Attributes Reference

- `ip_address` - IP address


## Import

Importing an Elastic IP resource is possible by name or id.

```shell
# by name
$ terraform import exoscale_ipaddress.myip 159.100.251.224

# by id
$ terraform import exoscale_ipaddress.myip eb556678-ec59-4be6-8c54-0406ae0f6da6
```

[eip]: https://community.exoscale.com/documentation/compute/eip/