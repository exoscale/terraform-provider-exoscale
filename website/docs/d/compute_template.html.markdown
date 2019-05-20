---
layout: "exoscale"
page_title: "Exoscale: exoscale_compute_template"
sidebar_current: "docs-exoscale-compute-template"
description: |-
  Retrieve information about a compute template.
---

# exoscale_compute_template

Get information on an Compute [template](https://www.exoscale.com/templates/)
for use in other resources such as a [Compute Instance](../r/compute.html).

## Example Usage

```hcl
locals {
  zone = "ch-gva-2"
}

data "exoscale_compute_template" "ubuntu" {
  zone = "${local.zone}"
  name = "Linux Ubuntu 18.04 LTS 64-bit"
}

resource "exoscale_compute" "my_server" {
  zone         = "${local.zone}"
  display_name = "my server"
  template     = "${data.exoscale_compute_template.ubuntu.id}"
  disk_size    = 10
  key_pair     = "my key"
}
```

## Argument Reference

- `zone` - (Required) Name of the [zone](https://www.exoscale.com/datacenters/)

- `name` - Name of the template

- `id` - ID of the template

- `filter` - Template search filter, must be either `featured` (official
  Exoscale templates), `community` (community-contributed templates) or `mine`
  (custom templates private to my organization). Default is `featured`.

## Attributes Reference

- `id` - ID of the template

- `name` - Name of the template

- `username` - Username to use to log into a Compute Instance based on this template
