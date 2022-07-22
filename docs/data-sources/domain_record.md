---
page_title: "Exoscale: exoscale_domain_record"
description: |-
  Fetch Exoscale DNS Domain Records data.
---

# exoscale\_domain\_record

Fetch Exoscale [DNS](https://community.exoscale.com/documentation/dns/) Domain Records data.

Corresponding resource: [exoscale_domain_record](../resources/domain_record.md).


## Usage

```hcl
data "exoscale_domain" "my_domain" {
  name = "my.domain"
}

data "exoscale_domain_record" "my_exoscale_domain_A_records" {
  domain = data.exoscale_domain.my_domain.name
  filter {
    name        = "my-host"
    record_type = "A"
  }
}

data "exoscale_domain_record" "my_exoscale_domain_NS_records" {
  domain = data.exoscale_domain.my_domain.name
  filter {
    content_regex = "ns.*"
  }
}

output "my_exoscale_domain_A_records" {
  value = join("\n", formatlist(
    "%s", data.exoscale_domain_record.my_exoscale_domain_A_records.records.*.name
  ))
}

output "my_exoscale_domain_NS_records" {
  value = join("\n", formatlist(
    "%s", data.exoscale_domain_record.my_exoscale_domain_NS_records.records.*.content
  ))
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

* `domain` - (Required) The [exoscale_domain](./domain.md) name to match.
* `filter`- (Required, Block) Filter to apply when looking up domain records. Structure is documented below.

### `filter` block

* `name` - The domain record name to match.
* `id` - The record ID to match.
* `record_type` - The record type to match.
* `content_regex` - A regular expression to match the record content.


## Attributes Reference

* `records` - The list of matching records. Structure is documented below.

### `records` items

In addition to the arguments listed above, the following attributes are exported:

* `content` - The domain record content.
* `prio` - The record priority.
* `ttl` - The record TTL.
