---
page_title: "Exoscale: exoscale_nlb"
description: |-
  Manage Exoscale Network Load Balancers (NLB).
---

# exoscale\_nlb

Manage Exoscale [Network Load Balancers (NLB)](https://community.exoscale.com/documentation/compute/network-load-balancer/).


## Usage

```hcl
resource "exoscale_nlb" "my_nlb" {
  zone = "ch-gva-2"
  name = "my-nlb"
}
```

Next step is to attach [NLB services](./nlb_service) to the network load balancer.

Please refer to the [examples](../../examples/) directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/

* `zone` - (Required) The name of the [zone][zone] to create the NLB into.
* `name` - (Required) The name of the NLB.

* `description` - A free-form text describing the NLB.
* `labels` - A map of key/value labels.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the NLB.
* `created_at` - The creation date of the NLB.
* `ip_address` - The public IPv4 address of the NLB.
* `services` - The list of the NLB service names.
* `state` - The current state of the NLB.


## Import

An existing NLB may be imported by `<ID>@<zone>`:

```console
$ terraform import \
  exoscale_nlb.my_nlb \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6@ch-gva-2
```

~> **NOTE:** Importing an NLB resource does _not_ import related `exoscale_nlb_service` resources.
