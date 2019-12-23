---
layout: "exoscale"
page_title: "Exoscale: exoscale_domain"
sidebar_current: "docs-exoscale-domain"
description: |-
  Provides information about a Domain.
---

# exoscale\_domain

Provides information on a domain name hosted on [Exoscale DNS][exodns].

[exodns]: https://www.exoscale.com/dns/

## Example Usage

```hcl
data "exoscale_domain" "my-company-com" {
  name = "my-company.com"
}
```

## Argument Reference

* `name` - (Required) The name of the domain.

## Attributes Reference

The following attributes are exported:

* `name` - Name of the Domain
* `id` - ID of the Domain