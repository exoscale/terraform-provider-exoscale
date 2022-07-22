---
page_title: "Exoscale: exoscale_nlb"
description: |-
  Manage Exoscale Network Load Balancers (NLB).
---

# exoscale\_nlb

Manage Exoscale [Network Load Balancers (NLB)](https://community.exoscale.com/documentation/compute/network-load-balancer/).

Corresponding data source: [exoscale_nlb](../data-sources/nlb.md).


## Usage

```hcl
resource "exoscale_nlb" "my_nlb" {
  zone = "ch-gva-2"
  name = "my-nlb"
}
```

Next step is to attach [exoscale_nlb_service](./nlb_service.md)(s) to the NLB.

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/

* `zone` - (Required) The Exoscale [Zone][zone] name.
* `name` - (Required) The network load balancer (NLB) name.

* `description` - A free-form text describing the NLB.
* `labels` - A map of key/value labels.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The network load balancer (NLB) ID.
* `created_at` - The NLB creation date.
* `ip_address` - The NLB IPv4 address.
* `services` - The list of the [exoscale_nlb_service](./nlb_service.md) (names).
* `state` - The current NLB state.


## Import

An existing network load balancer (NLB) may be imported by `<ID>@<zone>`:

```console
$ terraform import \
  exoscale_nlb.my_nlb \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6@ch-gva-2
```

~> **NOTE:** Importing an `exoscale_nlb` resource does _not_ import related `exoscale_nlb_service` resources.
