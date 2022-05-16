---
page_title: "Exoscale: exoscale_compute_instance_list"
description: |-
  Shows a list of Compute instances.
---

# exoscale\_compute\_instance_list

Lists available [Exoscale Compute instances][compute-doc].


## Example Usage

```hcl
data "exoscale_compute_instance_list" "example" {
  zone = "ch-gva-2"
}
```

## Arguments Reference

* `zone` - (Required) The [zone][zone] of the Compute instance.

## Atributes Reference

* `instances` - The list of instances.

For `instances` items schema see [exoscale_compute_instance][d-compute_instance] data source.

[compute-doc]: https://community.exoscale.com/documentation/compute/
[d-compute_instance]: ./exoscale_compute_instance
[zone]: https://www.exoscale.com/datacenters/
