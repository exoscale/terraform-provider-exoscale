---
page_title: "Exoscale: exoscale_compute_template"
description: |-
  Provides information about a Compute template.
---

# exoscale\_compute\_template

Provides information on a Compute [template][templates] for use in other resources such as a [`exoscale_compute`][r-compute] resource.


## Example Usage

```hcl
locals {
  zone = "ch-gva-2"
}

data "exoscale_compute_template" "ubuntu" {
  zone = local.zone
  name = "Linux Ubuntu 20.04 LTS 64-bit"
}

resource "exoscale_compute_instance" "my-server" {
  zone         = local.zone
  name         = "my-server"
  type         = "standard.medium"
  template_id  = data.exoscale_compute_template.ubuntu.id
  disk_size    = 20
}
```


## Arguments Reference

* `zone` - (Required) The name of the [zone][zone] where to look for the Compute template.
* `name` - The name of the Compute template (conflicts with `id`).
* `id` - The ID of the Compute template (conflicts with `name`).
* `filter` - A Compute template search filter, must be either `featured` (official Exoscale templates), `community` (community-contributed templates) or `mine` (custom templates private to my organization). Default is `featured`.



## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `username` - Username to use to log into a Compute Instance based on this template


[r-compute]: ../resources/compute
[templates]: https://www.exoscale.com/templates/
[zone]: https://www.exoscale.com/datacenters/

