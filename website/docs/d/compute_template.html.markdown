---
layout: "exoscale"
page_title: "Exoscale: exoscale_compute_template"
sidebar_current: "docs-exoscale-compute-template"
description: |-
  Provides information about a Compute template.
---

# exoscale\_compute\_template

Provides information on an Compute [template][templates] for use in other resources such as a [`exoscale_compute`][compute] resource.

[templates]: https://www.exoscale.com/templates/
[compute]: ../r/compute.html

## Example Usage

```hcl
locals {
  zone = "ch-gva-2"
}

data "exoscale_compute_template" "ubuntu" {
  zone = local.zone
  name = "Linux Ubuntu 18.04 LTS 64-bit"
}

resource "exoscale_compute" "my_server" {
  zone         = local.zone
  display_name = "my server"
  template_id  = data.exoscale_compute_template.ubuntu.id
  disk_size    = 10
  key_pair     = "my key"
}
```

## Argument Reference

* `zone` - (Required) The name of the [zone][zone] where to look for the Compute template.
* `name` - The name of the Compute template.
* `id` - The ID of the Compute template.
* `filter` - A Compute template search filter, must be either `featured` (official Exoscale templates), `community` (community-contributed templates) or `mine` (custom templates private to my organization). Default is `featured`.

[zone]: https://www.exoscale.com/datacenters/

## Attributes Reference

The following attributes are exported:

* `id` - ID of the template
* `name` - Name of the template
* `username` - Username to use to log into a Compute Instance based on this template
