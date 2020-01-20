---
layout: "exoscale"
page_title: "Exoscale: exoscale_compute"
sidebar_current: "docs-exoscale-compute"
description: |-
  Provides information about a Compute.
---

# exoscale\_compute

Provides information on an compute hosted on [Exoscale Compute][exocompute].

[exocompute]: https://www.exoscale.com/compute/

## Example Usage

```hcl
data "exoscale_compute" "my_server" {
  name = "my server"
}
```

## Argument Reference

* `name` - The name of the Compute.
* `id` - The ID of the Compute.
* `tags` - The tags to find the Compute.

## Attributes Reference

The following attributes are exported:

* `id` - ID of the compute.
* `name` - Name of the compute.
* `tags` - Map of tags (key: value).
* `created` - Date when the compute was created.
* `zone` - Name of the zone.
* `template` - Name of the template.
* `size` - Current size of the compute.
* `disk_size` - Size of the compute disk.
* `cpu` - Number of cpu the compute is running with.
* `memory` - Memory allocated for the Compute.
* `state` - State of the compute.
* `ip_address` - IP Address.
* `ip6_address` - IPv6 Address.
* `privnet_ip_address` - Privet Network IP Address.
* `privnet_ip6_address` - Privet Network IPv6 Address.