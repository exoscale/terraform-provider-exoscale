---
page_title: "Exoscale: exoscale_nlb_service"
description: |-
  Manage Exoscale Network Load Balancer (NLB) Services.
---

# exoscale\_nlb\_service

Manage Exoscale [Network Load Balancer (NLB)](https://community.exoscale.com/documentation/compute/network-load-balancer/) Services.


## Usage

```hcl
resource "exoscale_nlb" "my_nlb" {
  zone = "ch-gva-2"
  name = "my-nlb"
}

resource "exoscale_nlb_service" "my_nlb_service" {
  nlb_id = exoscale_nlb.my_nlb.id
  zone   = exoscale_nlb.my_nlb.zone
  name   = "my-nlb-service"

  instance_pool_id = exoscale_instance_pool.my_instance_pool.id
	protocol       = "tcp"
	port           = 443
	target_port    = 8443
	strategy       = "round-robin"

  healthcheck {
    mode     = "https"
    port     = 8443
    uri      = "/healthz"
    tls_sni  = "example.net"
    interval = 5
    timeout  = 3
    retries  = 1
  }
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/

* `nlb_id` - (Required) The parent [NLB](./nlb.md) ID.
* `zone` - (Required) The name of the [zone][zone] used by the NLB.
* `name` - (Required) The name of the NLB service.
* `instance_pool_id` - (Required) The ID of the [instance pool](./instance_pool.md) to forward traffic to.
* `port` - (Required) The port of the NLB service.
* `target_port` - (Required) The port to forward traffic to (on target instance pool members).

* `description` - A free-form text describing the NLB service.
* `protocol` - The protocol (`tcp`|`udp`; default: `tcp`).
* `strategy` - The strategy (`round-robin`|`source-hash`; default: `round-robin`).

* `healthcheck` - (Block) The service health checking configuration (may only bet set at creation time). Structure is documented below.

### `healthcheck` block

* `port` - (Required) The healthcheck port.

* `interval` - The healthcheck interval in seconds (default: `10`).
* `mode` - The healthcheck mode (`tcp`|`http`|`https`; default: `tcp`).
* `retries` - The healthcheck retries (default: `1`).
* `timeout` - The healthcheck timeout in seconds (default: `5`).
* `tls_sni` - The healthcheck TLS SNI server name, only if `mode` is `https`.
* `uri` - The healthcheck URI, must be set only if `mode` is `http(s)`.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the NLB service.


## Import

An existing NLB service may be imported by `<nlb-ID>/<service-ID>@<zone>`:

```console
$ terraform import \
  exoscale_nlb_service.my_nlb_service \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6/9ecc6b8b-73d4-4211-8ced-f7f29bb79524@ch-gva-2
```
