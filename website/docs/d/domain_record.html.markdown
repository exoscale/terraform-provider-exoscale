---
layout: "exoscale"
page_title: "Exoscale: exoscale_domain_record"
sidebar_current: "docs-exoscale-domain_record"
description: |-
  Provides information about an Exoscale DNS domain record.
---

# exoscale\_domain\_record

Provides information on [domain records][record] hosted on [Exoscale DNS][exodns].

[exodns]: https://www.exoscale.com/dns/
[record]: ../r/domain_record.html

## Example Usage

The example below matches all Domain Records that match with name `mailserver` and Record type `MX`.

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
  value = $(data.exoscale_domain_record.mycompany_mailservers.records.0.name)
}

output "first_domain_record_content" {
  value = $(data.exoscale_domain_record.mycompany_nameservers.records.0.content)
}
```

## Argument Reference

* `domain` - (Required) The name of the [domain][domain] where to look for Domain Records.
* `filter`- (Required) One value is used to look up Domain Records or `name` and `record_type` together.

**filter**

* `name` - The name matching the Domain Record name to lookup.
* `id` - The ID matching the Domain Record ID to lookup.
* `record_type` - The Record type matching the Domain Record type to lookup.
* `content_regex` - A regular expression matching the Domain Record content to lookup.


[domain]: ../r/domain.html

## Attributes Reference

The following attributes are exported:

**records**

* `id` - Domain Record ID
* `domain` - Domain Name where the Record is associate to.
* `name` - Domain Record name
* `content` - Domain Record content
* `record_type` - Domain Record type
* `prio` - Domain Record prio