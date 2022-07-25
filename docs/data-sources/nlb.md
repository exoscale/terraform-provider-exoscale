---
page_title: "Exoscale: exoscale_nlb"
description: |-
  Fetch Exoscale Network Load Balancers (NLB) data.
---

# exoscale\_nlb

Fetch Exoscale [Network Load Balancers (NLB)](https://community.exoscale.com/documentation/compute/network-load-balancer/) data.

Corresponding resource: [exoscale_nlb](../resources/nlb.md).


## Usage

```hcl
data "exoscale_nlb" "my_nlb" {
  zone = "ch-gva-2"
  name = "my-nlb"
}

output "my_nlb_id" {
  value = data.exoscale_nlb.my_nlb.id
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/

* `zone` - (Required) The Exoscale [Zone][zone] name.

* `id` - The Network Load Balancers (NLB) ID to match (conflicts with `name`).
* `name` - The NLB name to match (conflicts with `id`).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `description` - The Network Load Balancers (NLB) description.
* `created_at` - The NLB creation date.
* `ip_address` - The NLB public IPv4 address.
* `state` - The current NLB state.
