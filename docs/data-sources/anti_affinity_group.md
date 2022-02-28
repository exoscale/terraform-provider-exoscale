---
page_title: "Exoscale: exoscale_anti_affinity_group"
description: |-
  Provides information about an Anti-Affinity Group.
---

# exoscale\_anti\_affinity\_group

Provides information on an [Anti-Affinity Group][aag-doc] for use in other resources such as a [`exoscale_compute`][r-compute] resource.


## Example Usage

```hcl
locals {
  zone = "ch-gva-2"
}

data "exoscale_anti_affinity_group" "web" {
  name = "web"
}

data "exoscale_compute_template" "ubuntu" {
  zone = local.zone
  name = "Linux Ubuntu 20.04 LTS 64-bit"
}

resource "exoscale_compute_instance" "my-server" {
  zone                   = local.zone
  name                   = "my-server"
  type                   = "standard.medium"
  template_id            = data.exoscale_compute_template.ubuntu.id
  disk_size              = 20
  anti_affinity_group_ids = [data.exoscale_anti_affinity_group.web.id]
}
```


## Arguments Reference

* `name` - The name of the Anti-Affinity Group (conflicts with `id`).
* `id` - The ID of the Anti-Affinity Group (conflicts with `name`).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `instances` - A list of Compute instance IDs belonging to the Anti-Affinity Group.


[aag-doc]: https://community.exoscale.com/documentation/compute/anti-affinity-groups/
[r-compute]: ../resources/compute
