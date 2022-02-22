---
layout: "exoscale"
page_title: "Exoscale: exoscale_domain_record"
sidebar_current: "docs-exoscale-domain_record"
description: |-
  Provides information about an Exoscale DNS domain record.
---

# exoscale\_domain\_record

Provides information on [domain records][r-domain_record] hosted on [Exoscale DNS][exo-dns].


## Example Usage

The example below matches all domain records that match with name `mailserver` and Record type `MX`.

```hcl
data "exoscale_domain" "mycompany" {
  name = my-company.com
}

data "exoscale_domain_record" "mycompany_mailservers" {
  domain = data.exoscale_domain.mycompany.name
  filter {
    name   = "mailserver"
    recorde_type  = "MX"
  }
}

data "exoscale_domain_record" "mycompany_nameservers" {
  domain = data.exoscale_domain.mycompany.name
  filter {
    content_regex  = "ns.*"
  }
}

output "first_domain_record_name" {
  value = data.exoscale_domain_record.mycompany_mailservers.records.0.name
}

output "first_domain_record_content" {
  value = data.exoscale_domain_record.mycompany_nameservers.records.0.content
}
```


## Arguments Reference

* `domain` - (Required) The name of the [domain][r-domain] where to look for domain records.
* `filter`- (Required) Filter to apply when looking up domain records.

**filter**

* `name` - The name matching the domain record name to lookup.
* `id` - The ID matching the domain record ID to lookup.
* `record_type` - The record type matching the domain record type to lookup.
* `content_regex` - A regular expression matching the domain record content to lookup.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `content` - The content of the domain record.
* `prio` - The priority of the domain record.


[exo-dns]: https://www.exoscale.com/dns/
[r-domain]: ../resources/domain
[r-domain_record]: ../resources/domain_record

