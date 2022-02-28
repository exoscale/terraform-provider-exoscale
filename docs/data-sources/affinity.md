---
page_title: "Exoscale: exoscale_affinity"
description: |-
  Provides information about an Anti-Affinity Group.
---

# exoscale\_affinity

Provides information on an [Anti-Affinity Group][aag-doc] for use in other resources such as a [`exoscale_compute`][r-compute] resource.

!> **WARNING:** This data source is deprecated and will be removed in the next major version.


## Example Usage

```hcl
locals {
  zone = "ch-gva-2"
}

data "exoscale_affinity" "web" {
  name = "web"
}

data "exoscale_compute_template" "ubuntu" {
  zone = local.zone
  name = "Linux Ubuntu 20.04 LTS 64-bit"
}

resource "exoscale_compute" "my_server" {
  zone               = local.zone
  template_id        = data.exoscale_compute_template.ubuntu.id
  disk_size          = 20
  affinity_group_ids = [data.exoscale_affinity.web.id]
}
```


## Arguments Reference

* `name` - The name of the Anti-Affinity Group (conflicts with `id`)
* `id` - The ID of the Anti-Affinity Group (conflicts with `name`)


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* n/a


[aag-doc]: https://community.exoscale.com/documentation/compute/anti-affinity-groups/
[r-compute]: ../resources/compute

