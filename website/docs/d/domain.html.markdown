---
layout: "exoscale"
page_title: "Exoscale: exoscale_domain"
sidebar_current: "docs-exoscale-domain"
description: |-
  Provides information about a Domain.
---

# exoscale\_domain

Provides information on an [domain][domain].

[domain]: ../r/domain.html

## Example Usage

```hcl
data "exoscale_domain" "exo" {
  name = my-company.com
}
```

## Argument Reference

* `name` - (Required) The name of the [domain][domain].

[domain]: https://www.exoscale.com/dns/

## Attributes Reference

The following attributes are exported:

* `name` - Name of the Domain
* `id` - ID of the Domain