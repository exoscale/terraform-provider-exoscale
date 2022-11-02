---
page_title: "Exoscale: exoscale_compute_instance_list"
description: |-
  List Exoscale Compute Instances.
---

# exoscale\_compute\_instance_list

List Exoscale [Compute Instances](https://community.exoscale.com/documentation/compute/).

Corresponding resource: [exoscale_compute_instance](../resources/compute_instance.md).


## Usage

```hcl
data "exoscale_compute_instance_list" "my_compute_instance_list" {
  zone = "ch-gva-2"
}

output "my_compute_instance_ids" {
  value = join("\n", formatlist(
    "%s", data.exoscale_compute_instance_list.my_compute_instance_list.instances.*.id
  ))
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/

* `zone` - (Required) The Exoscale [Zone][zone] name.


## Atributes Reference

* `instances` - The list of [exoscale_compute_instance](./compute_instance.md).
