---
layout: "exoscale"
page_title: "Exoscale: exoscale_domain_record"
sidebar_current: "docs-exoscale-domain_record"
description: |-
  Provides information about a Domain Record.
---

# exoscale\_domain\_record

Use this data source to look Domain [Records][record].

[templates]: https://community.exoscale.com/documentation/dns/api/

## Example Usage

The example below matches all Domain Records that match with name `mailserver` and Record type `MX`.

```hcl
data "exoscale_domain" "exo" {
  name = my-company.com
}

data "exoscale_domain_record" "mx" {
  domain = test.com"
  filter {
    name   = "mailserver"
    recorde_type  = "MX"
    # OR 
    # id   = 12345
    # OR 
    # record_type  = "MX"
    # OR 
    # content  = "mta*"  You can use Regex or not.
}

output "first_record_id" {
  value = $(data.exoscale_domain_record.mx.records.0.name)
  # value = $(data.exoscale_domain_record.mx.records.1.id)
}
```

## Argument Reference

* `domain` - (Required) The name of the [domain][domain] where to look for Domain Records.
* `filter`- (Required) One value is used to look up Domain Records or `name` and `record_type` together.

**filter**

* `name` - The name matching the Domain Record name to lookup.
* `id` - The ID matching the Domain Record ID to lookup.
* `record_type` - The Record type matching the Domain Record type to lookup.
* `content` - A regular expression matching the Domain Record content to lookup.


[domain]: https://www.exoscale.com/dns/

## Attributes Reference

The following attributes are exported:

**records**

* `id` - Domain Record ID
* `domain` - Domain Name where the Record is associate to.
* `name` - Domain Record name
* `content` - Domain Record content
* `create_at` - Domain Record creation date
* `update_at` - Domain Record last updated date 
* `record_type` - Domain Record type
* `prio` - Domain Record prio