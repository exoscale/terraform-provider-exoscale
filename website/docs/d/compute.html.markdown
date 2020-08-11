---
layout: "exoscale"
page_title: "Exoscale: exoscale_compute"
sidebar_current: "docs-exoscale-compute"
description: |-
  Provides information about a Compute.
---

# exoscale\_compute

Provides information on an [Exoscale Compute instance][compute-doc].


## Example Usage

```hcl
data "exoscale_compute" "my_server" {
  hostname = "my server"
}
```

## Arguments Reference

* `id` - The ID of the Compute instance.
* `hostname` - The hostname of the Compute instance.
* `tags` - The tags to find the Compute instance (key: value).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `created` - Creation date of the Compute instance.
* `zone` - Name of the zone.
* `template` - Name of the template.
* `size` - Current size of the Compute instance.
* `disk_size` - Size of the Compute instance disk.
* `cpu` - Number of cpu the Compute instance is running with.
* `memory` - Memory allocated for the Compute instance.
* `state` - State of the Compute instance.
* `ip_address` - Public IPv4 address of the Compute instance.
* `ip6_address` - Public IPv6 address of the Compute instance (if IPv6 is enabled).
* `private_network_ip_addresses` - List of Compute private IP addresses (in managed Private Networks only).


[compute-doc]: https://www.exoscale.com/compute/
