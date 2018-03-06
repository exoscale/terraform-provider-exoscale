---
layout: "exoscale"
page_title: "Exoscale: exoscale_domain"
sidebar_current: "docs-exoscale-domain"
description: |-
  Manages a domain.
---

# exoscale_domain

Define a domain in the sense of a website address.

## Usage example

```hcl
resource "exoscale_domain" "exo" {
  name = "exo.exo"
}
```

## Argument Reference

- `name` - (Required) name of the domain.


## Attributes Reference

The following attributes are exported:

- `token` - this token serves as an alternative way to manages the domain records

- `state`

- `auto_renew`

- `expires_on` - date of expiration, if known

## Import

Importing a domain will import all the records (but `NS` and `SOA`).

```shell
$ terraform import exoscale_domain.exoscale-ch exoscale.ch
```
