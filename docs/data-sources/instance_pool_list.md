---
page_title: "Exoscale: exoscale_instance_pool_list"
description: |-
  Shows a list of instance Pools.
---

# exoscale\_instance\_pool\_list

Lists available [Exoscale Instance Pools][pool-doc].


## Example Usage

```hcl
data "exoscale_instance_pool_list" "example" {
  zone = "ch-gva-2"
}
```

## Arguments Reference

* `zone` - (Required) The [zone][zone] of the Compute instance pool.

## Attributes Reference

* `pools` - The list of instance pools.

For `pools` items schema see [exoscale_instance_pool][d-instance_pool] data source.

[pool-doc]: https://community.exoscale.com/documentation/compute/instance-pools/
[zone]: https://www.exoscale.com/datacenters/
[d-instance_pool]: ../data-sources/instance_pool
