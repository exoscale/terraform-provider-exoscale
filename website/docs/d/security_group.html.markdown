---
layout: "exoscale"
page_title: "Exoscale: exoscale_security_group"
sidebar_current: "docs-exoscale-security-group"
description: |-
  Provides information about a Security Group.
---

# exoscale\_security\_group

Provides information on a [Security Group][ag] for use in other resources such as a [`exoscale_instance_pool`][instancepool] resource.

[ag]: https://community.exoscale.com/documentation/compute/security-groups/
[compute]: ../r/compute.html


## Example Usage

```hcl
locals {
  zone = "ch-gva-2"
}

data "exoscale_security_group" "web" {
  name = "web"
}

data "exoscale_compute_template" "ubuntu" {
  zone = local.zone
  name = "Linux Ubuntu 20.04 LTS 64-bit"
}

resource "exoscale_instance_pool" "webservers" {
  zone               = local.zone
  name               = "webservers"
  template_id        = data.exoscale_compute_template.ubuntu.id
  size               = 5
  service_offering   = "medium"
  security_group_ids = [data.exoscale_security_group.web.id]
}
```


## Argument Reference

* `name` - The name of the Security Group (conflicts with `id`)
* `id` - The ID of the Security Group (conflicts with `name`)


## Attributes Reference

The following attributes are exported:

* `id` - ID of the Security Group
* `name` - Name of the Security Group
