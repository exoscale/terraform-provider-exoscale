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

## Argument Reference

* `zone` - (Required) The name of the [zone][zone] to deploy the NLB into.
* `name` - (Required) The name of the NLB.
* `description` - The description of the NLB.

[zone]: https://www.exoscale.com/datacenters/

## Import

An existing NLB can be imported as a resource by ID. Importing a NLB imports the `exoscale_nlb` resource.

```console
$ terraform import exoscale_nlb.website eb556678-ec59-4be6-8c54-0406ae0f6da6

```
