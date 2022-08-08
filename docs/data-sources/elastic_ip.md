---
page_title: "Exoscale: exoscale_elastic_ip"
description: |-
  Fetch Exoscale Elastic IPs (EIP) data.
---

# exoscale\_elastic\_ip

Fetch Exoscale [Elastic IPs (EIO)](https://community.exoscale.com/documentation/compute/eip/) data.

Corresponding resource: [exoscale_elastic_ip](../resources/elastic_ip.md).


## Usage

```hcl
data "exoscale_elastic_ip" "my_elastic_ip" {
  zone       = "ch-gva-2"
  ip_address = "1.2.3.4"
}

output "my_elastic_ip_id" {
  value = data.exoscale_elastic_ip.my_elastic_ip.id
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/

* `zone` - (Required) The Exocale [Zone][zone] name.

* `id` - The Elastic IP (EIP) ID to match (conflicts with `ip_address`).
* `ip_address` - The EIP IPv4 or IPv6 address to match (conflicts with `id`).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `address_family` - The Elastic IP (EIP) address family (`inet4` or `inet6`).

* `cidr` - The Elastic IP (EIP) CIDR.

* `description` - The Elastic IP (EIP) description.

* `healthcheck` - (Block) The *managed* EIP healthcheck configuration. Structure is documented below.

### `healthcheck` block

* `mode` - The healthcheck mode.
* `port` - The healthcheck target port.
* `uri` - The healthcheck URI.
* `interval` - The healthcheck interval in seconds.
* `timeout` - The time in seconds before considering a healthcheck probing failed.
* `strikes_ok` - The number of successful healthcheck attempts before considering the target healthy.
* `strikes_fail` - The number of failed healthcheck attempts before considering the target unhealthy.
* `tls_sni` - The healthcheck server name to present with SNI in `https` mode.
* `tls_skip_verify` - Disable TLS certificate verification for healthcheck in `https` mode.
