---
layout: "exoscale"
page_title: "Exoscale: exoscale_domain_record"
sidebar_current: "docs-exoscale-domain_record"
description: |-
  Provides information about a Domain Record.
---

# exoscale\_domain\_record

Provides information on an Domain [Record][record].

[templates]: https://community.exoscale.com/documentation/dns/api/
[compute]: ../r/compute.html

## Example Usage

```hcl
resource "exoscale_domain" "exo" {
  name = my-company.com
}

data "exoscale_domain_record" "mx" {
  domain = "${exoscale_domain.exo.id}"
  name   = "mail"
}
```

## Argument Reference

* `domain` - (Required) The name of the [domain][domain] where to look for a Record.
* `name` - The name of the Record.
* `id` - The ID of the Record.

[domain]: https://www.exoscale.com/dns/

## Attributes Reference

The following attributes are exported:

* `domain` - Domain of the Record
* `name` - Name of the Record
* `id` - ID of the Record
