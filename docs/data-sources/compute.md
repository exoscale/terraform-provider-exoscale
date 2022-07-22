---
page_title: "Exoscale: exoscale_compute"
subcategory: "Deprecated"
description: |-
  Fetch Exoscale Compute Instances data.
---

# exoscale\_compute

!> **WARNING:** This data source is **DEPRECATED** and will be removed in the next major version. Please use [exoscale_compute_instance](./compute_instance) instead.

## Arguments Reference

* `id` - The compute instance ID to match.
* `hostname` - The compute instance hostname to match.
* `tags` - The compute instance tags to match (map of key/value).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `zone` - Name of the zone.
* `cpu` - Number of cpu the compute instance is running with.
* `created` - Creation date of the compute instance.
* `disk_size` - Size of the compute instance disk.
* `ip_address` - Public IPv4 address of the compute instance.
* `ip6_address` - Public IPv6 address of the compute instance (if IPv6 is enabled).
* `memory` - Memory allocated for the compute instance.
* `size` - Current size of the compute instance.
* `state` - State of the compute instance.
* `template` - Name of the template.

* `private_network_ip_addresses` - List of compute private IP addresses (in *managed* private networks only).
