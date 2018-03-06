---
layout: "exoscale"
page_title: "Exoscale: exoscale_domain_record"
sidebar_current: "docs-exoscale-domain-record"
description: |-
  Manages a domain record
---

# exoscale_domain_record

Defines a DNS entry linked with a domain.

## Usage example

```hcl
resource "exoscale_domain_record" "glop" {
  domain = "${exoscale_domain.exo.id}"
  name = "glap"
  record_type = "CNAME"
  content = "${exoscale_domain.exo.name}"
}
```

## Argument Reference

- `domain` - (Required) domain it's linked to

- `name` - (Required) name of the DNS record

- `record_type` - (Required) type of the DNS record. E.g. `A`, `CNAME`, `MX`, etc.

- `content` - (Required) value of the DNS record

- `ttl` - time to live

- `prio` - priority

## Attributes Reference

- `hostname` - full name, useful for linking `A` records into `CNAME`.

## Import

A record is imported with its domain resource.
