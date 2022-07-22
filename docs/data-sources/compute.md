---
page_title: "Exoscale: exoscale_compute"
subcategory: "Deprecated"
description: |-
  Fetch Exoscale Compute Instances data.
---

# exoscale\_compute

!> **WARNING:** This data source is **DEPRECATED** and will be removed in the next major version. Please use [exoscale_compute_instance](./compute_instance.md) instead.

## Arguments Reference

* `id` - The compute instance ID to match.
* `hostname` - The instance hostname to match.
* `tags` - The instance tags to match (map of key/value).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `zone` - Exoscale Zone name.
* `cpu` - The compute instance number of CPUs.
* `created` - The instance creation date.
* `disk_size` - The instance disk size (GiB).
* `ip_address` - The instance (main network interface) IPv4 address.
* `ip6_address` - The instance (main network interface) IPv6 address (if enabled).
* `memory` - The instance allocated memory.
* `size` - The instance size.
* `state` - The current instance state.
* `template` - The instance template.

* `private_network_ip_addresses` - List of compute private IPv4 addresses (in *managed* private networks only).
