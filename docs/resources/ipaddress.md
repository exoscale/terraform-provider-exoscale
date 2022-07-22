---
page_title: "Exoscale: exoscale_ipaddress"
subcategory: "Deprecated"
description: |-
  Manage Exoscale Elastic IPs (EIP).
---

# exoscale\_ipaddress

!> **WARNING:** This resource is **DEPRECATED** and will be removed in the next major version. Please use [exoscale_elastic_ip](./elastic_ip.md) instead.


## Arguments Reference

* `zone` - (Required) The name of the zone to create the EIP into.

* `description` - A free-form text describing the EIP.
* `healthcheck_interval` - The healthcheck probing interval (seconds; must be between `5` and `300`).
* `healthcheck_mode` - The healthcheck probing mode (must be `tcp`, `http` or `https`).
* `healthcheck_port` - The healthcheck service port to probe (must be between `1` and `65535`).
* `healthcheck_path` - The healthcheck probe HTTP request path (must be specified in `http`/`https` modes).
* `healthcheck_strikes_fail` - The number of unsuccessful healthcheck probes before considering the target unhealthy (must be between `1` and `20`).
* `healthcheck_strikes_ok` - The number of successful healthcheck probes before considering the target healthy (must be between `1` and `20`).
* `healthcheck_timeout` - The time in seconds before considering a healthcheck probing failed (must be between `2` and `60`).
* `healthcheck_tls_skip_verify` - Disable TLS certificate validation in `https` mode (boolean; default: `false`). Note: this parameter can only be changed to `true`, it cannot be reset to `false` later on (requires a resource re-creation).
* `healthcheck_tls_sni` - The healthcheck TLS server name to specify in `https` mode. Note: this parameter can only be changed to a non-empty value, it cannot be reset to its default empty value later on (requires a resource re-creation).
* `reverse_dns` - A reverse DNS record to set for the EIP.
* `tags` - A dictionary of tags (key/value). To remove all tags, set `tags = {}`.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `ip_address` - The EIP address.
