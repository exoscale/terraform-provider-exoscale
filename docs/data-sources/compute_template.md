---
page_title: "Exoscale: exoscale_compute_template"
description: |-
  Fetch Exoscale Compute Instance Templates data.
---

# exoscale\_compute\_template

Fetch Exoscale [Compute Instance Templates](https://community.exoscale.com/documentation/compute/custom-templates/) data.


## Usage

```hcl
data "exoscale_compute_template" "my_template" {
  zone = "ch-gva-2"
  name = "Linux Ubuntu 22.04 LTS 64-bit"
}

output "my_template_id" {
  value = data.exoscale_compute_template.my_template.id
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/

* `zone` - (Required) The Exoscale [Zone][zone] name.

* `id` - The compute instance template ID to match (conflicts with `name`).
* `name` - The template name to match (conflicts with `id`).
* `filter` - A template category filter (default: `featured`); among:
  - `featured` - official Exoscale templates
  - `community` - community-contributed templates
  - `mine` - custom templates private to my organization


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `username` - Username to use to log into a compute instance based on this template
