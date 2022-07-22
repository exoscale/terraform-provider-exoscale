---
page_title: "Exoscale: exoscale_instance_pool_list"
description: |-
  Fetch a list of Exoscale Instance Pools.
---

# exoscale\_instance\_pool\_list

List Exoscale [Instance Pools](https://community.exoscale.com/documentation/compute/instance-pools/).

Corresponding resource: [exoscale_instance_pool](../resources/instance_pool.md).


## Usage

```hcl
data "exoscale_instance_pool_list" "my_instance_pool_list" {
  zone = "ch-gva-2"
}

output "my_instance_pool_ids" {
  value = join("\n", formatlist(
    "%s", exoscale_instance_pool_list.my_instance_pool_list.pools.*.id
  ))
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/

* `zone` - (Required) The Exoscale [Zone][zone] name.


## Attributes Reference

* `pools` - The list of [exoscale_instance_pool](./instance_pool.md).
