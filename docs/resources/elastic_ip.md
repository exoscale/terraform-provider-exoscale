---
page_title: "Exoscale: exoscale_elastic_ip"
description: |-
  Manage Exoscale Elastic IPs (EIP).
---

# exoscale\_elastic\_ip

Manage Exoscale [Elastic IPs (EIP)](https://community.exoscale.com/documentation/compute/eip/).


### Usage

*Unmanaged* EIP:

```hcl
resource "exoscale_elastic_ip" "my_elastic_ip" {
  zone = "ch-gva-2"
}
```

*Managed* EIP:

```hcl
resource "exoscale_elastic_ip" "my_managed_elastic_ip" {
  zone = "ch-gva-2"

  healthcheck {
    mode         = "https"
    port         = 443
    uri          = "/health"
    interval     = 5
    timeout      = 3
    strikes_ok   = 2
    strikes_fail = 3
    tls_sni      = "example.net"
  }
}
```

Please refer to the [examples](../../examples/) directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/

* `zone` - (Required) The name of the [zone][zone] to create the EIP into.

* `description` - A free-form text describing the EIP.

* `healthcheck` - (Block) A health checking configuration for managed EIPs. Structure is documented below.

### `healthcheck` block

* `mode` - (Required) The health checking mode (`tcp`, `http` or `https`; may only be set at creation time).
* `port` - (Required) The health checking port (must be between `1` and `65535`).

* `interval` - The health checking interval (seconds; must be between `5` and `300`; default: `10`).
* `strikes_fail` - The number of failed attempts before considering a managed EIP target unhealthy (must be between `1` and `20`; default: `2`).
* `strikes_ok` - The number of successful attempts before considering a managed EIP target healthy (must be between `1` and `20`; default: `3`).
* `timeout` - The time before considering a healthcheck probing failed (seconds; must be between `2` and `60`; default: `3`).
* `tls_skip_verify` - Disable TLS certificate verification for health checking in `https` mode (boolean; default: `false`).
* `tls_sni` - The health checking server name to present with SNI in `https` mode.
* `uri` - The health checking URI (required in `http(s)` modes).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `ip_address` - The EIP IPv4 address.


## Import

An existing EIP may be imported by `<ID>@<zone>`:

```console
$ terraform import \
  exoscale_elastic_ip.my_elastic_ip \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6@ch-gva-2
```
