---
layout: "exoscale"
page_title: "Exoscale: exoscale_nlb"
sidebar_current: "docs-exoscale-nlb"
description: |-
  Provides an Exoscale Network Load Balancer resource.
---

# exoscale\_nlb

Provides an Exoscale Network Load Balancer (NLB) resource. This can be used to create, modify, and delete NLBs.


## Example Usage

```hcl
variable "zone" {
  default = "de-fra-1"
}

resource "exoscale_nlb" "website" {
  zone = var.zone
  name = "website"
  description = "This is the Network Load Balancer for my website"
}
```


## Arguments Reference

* `zone` - (Required) The name of the [zone][zone] to deploy the NLB into.
* `name` - (Required) The name of the NLB.
* `description` - The description of the NLB.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `ip_address` - The public IP address of the NLB.
* `state` - The current state of the NLB.
* `created_at` - The creation date of the NLB.
* `services` - The list of the NLB service names.


## Import

An existing NLB can be imported as a resource by ID:

```console
$ terraform import exoscale_nlb.website eb556678-ec59-4be6-8c54-0406ae0f6da6
```

~> **NOTE:** Importing a NLB resource also imports related [`exoscale_nlb_service`][r-nlb_service] resources.


[r-nlb_service]: nlb_service.html
[zone]: https://www.exoscale.com/datacenters/

