---
layout: "exoscale"
page_title: "Exoscale: exoscale_ipaddress"
sidebar_current: "docs-exoscale-ipaddress"
description: |-
  Manages an elastic IP address.
---

# exoscale_ipaddress

The elastic IP address is an address that belongs to a specific zone and may be
attributed to many compute. See [secondary_ipaddress](secondary_ipaddress.html).

### Usage example

```
resource "exoscale_ipaddress" "myip" {
  zone = "ch-dk-2"
  tags {
    usage = "load-balancer"
  }
}
```

## Argument Reference

- `zone` - (Required) name of [the data-center](https://www.exoscale.com/datacenters/)

- `tags` - dictionary of tags (key / value)

## Attributes Reference

- `ip_address` - IP address


## Import

Importing an Elastic IP resource is possible by name or id.

```shell
# by name
$ terraform import exoscale_ipaddress.myip 159.100.251.224

# by id
$ terraform import exoscale_ipaddress.myip eb556678-ec59-4be6-8c54-0406ae0f6da6
```
