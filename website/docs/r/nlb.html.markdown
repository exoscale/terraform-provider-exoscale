---
layout: "exoscale"
page_title: "Exoscale: exoscale_nlb"
sidebar_current: "docs-exoscale-nlb"
description: |-
  Provides an Exoscale Network Load Balancer resource.
---

# exoscale\_nlb

Provides an Exoscale [Network Load Balancer][nlb-doc] (NLB) resource. This can be used to create, modify, and delete NLBs.


## Example Usage

```hcl
variable "zone" {
  default = "de-fra-1"
}

resource "exoscale_nlb" "website" {
  zone = var.zone
  name = "website"
  description = "This is the Network Load Balancer for my website"

  labels = {
    env = "prod"
  }
}
```


## Arguments Reference

* `zone` - (Required) The name of the [zone][zone] to deploy the NLB into.
* `name` - (Required) The name of the NLB.
* `description` - The description of the NLB.
* `labels` - A map of key/value labels.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the NLB.
* `ip_address` - The public IP address of the NLB.
* `state` - The current state of the NLB.
* `created_at` - The creation date of the NLB.
* `services` - The list of the NLB service names.


## Import

An existing NLB can be imported as a resource by `<ID>@<ZONE>`:

```console
$ terraform import exoscale_nlb.example eb556678-ec59-4be6-8c54-0406ae0f6da6@de-fra-1
```

~> **NOTE:** Importing a NLB resource doesn't import related [`exoscale_nlb_service`][r-nlb_service] resources.


[nlb-doc]: https://community.exoscale.com/documentation/compute/network-load-balancer/
[r-nlb_service]: nlb_service.html
[zone]: https://www.exoscale.com/datacenters/

