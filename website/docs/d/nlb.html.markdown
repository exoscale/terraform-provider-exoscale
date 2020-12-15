---
layout: "exoscale"
page_title: "Exoscale: exoscale_nlb"
sidebar_current: "docs-exoscale-nlb"
description: |-
  Provides information about a Network Load Balancer.
---

# exoscale\_nlb

Provides information on a [Network Load Balancer][nlb-doc] (NLB) instance for use in other resources such as a [`exoscale_nlb_service`][r-nlb_service] resource.


## Example Usage

```hcl
data "exoscale_nlb" "prod" {
  zone = "ch-gva-2"
  name = "prod"
}

output "nlb_prod_ip_address" {
  value = data.exoscale_nlb.prod.ip_address
}
```


## Arguments Reference

* `zone` - (Required) The [zone][zone] of the NLB.
* `id` - The ID of the NLB (conflicts with `name`).
* `name` - The name of NLB (conflicts with `id`).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `description` - The description of the NLB.
* `state` - The current state of the NLB.
* `created_at` - The creation date of the NLB.
* `ip_address` - The public IP address of the NLB.


[nlb-doc]: https://community.exoscale.com/documentation/compute/network-load-balancer/
[r-instance_pool]: ../r/instance_pool.html
[zone]: https://www.exoscale.com/datacenters/

