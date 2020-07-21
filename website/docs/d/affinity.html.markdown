---
layout: "exoscale"
page_title: "Exoscale: exoscale_affinity"
sidebar_current: "docs-exoscale-affinity"
description: |-
  Provides information about an Anti-Affinity Group.
---

# exoscale\_affinity

Provides information on an [Anti-Affinity Group][ag] for use in other resources such as a [`exoscale_compute`][compute] resource.

[ag]: https://community.exoscale.com/documentation/compute/anti-affinity-groups/
[compute]: ../r/compute.html


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


## Argument Reference

* `name` - The name of the Anti-Affinity Group (conflicts with `id`)
* `id` - The ID of the Anti-Affinity Group (conflicts with `name`)


## Attributes Reference

The following attributes are exported:

* `id` - ID of the Anti-Affinity Group
* `name` - Name of the Anti-Affinity Group
