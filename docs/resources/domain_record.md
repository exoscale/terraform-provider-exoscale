---
layout: "exoscale"
page_title: "Exoscale: exoscale_domain_record"
sidebar_current: "docs-exoscale-domain-record"
description: |-
  Provides an Exoscale DNS domain record resource.
---

# exoscale\_domain\_record

Provides an Exoscale [DNS][dns-doc] domain record resource. This can be used to create, modify, and delete DNS domain records.


## Usage example

```hcl
resource "exoscale_domain" "example" {
  name = "example.net"
}

resource "exoscale_domain_record" "myserver" {
  domain      = exoscale_domain.example.id
  name        = "myserver"
  record_type = "A"
  content     = "1.2.3.4"
}

resource "exoscale_domain_record" "myserver_alias" {
  domain      = exoscale_domain.example.id
  name        = "myserver-new"
  record_type = "CNAME"
  content     = exoscale_domain_record.myserver.hostname
}
```


## Arguments Reference

* `domain` - (Required) The name of the [`exoscale_domain`][r-domain] to create the record into.
* `name` - (Required) The name of the domain record; leave blank (`""`) to create a root record (similar to using `@` in a DNS zone file).
* `record_type` - (Required) The type of the domain record. Supported values are: `A`, `AAAA`, `ALIAS`, `CAA`, `CNAME`, `HINFO`, `MX`, `NAPTR`, `NS`, `POOL`, `SPF`, `SRV`, `SSHFP`, `TXT`, `URL`.
* `content` - (Required) The value of the domain record.
* `ttl` - The [Time To Live][ttl] of the domain record.
* `prio` - The priority of the DNS domain record (for types that support it).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `hostname` - The DNS domain record's *Fully Qualified Domain Name* (FQDN), useful for linking `A` records into `CNAME`.


## Import

An existing DNS domain record can be imported as a resource by ID:

```console
$ terraform import exoscale_domain_record.www 12480484
```

~> **NOTE:** importing an existing [`exoscale_domain`][r-domain] resource also imports linked `exoscale_domain_record` resources.


[dns-doc]: https://community.exoscale.com/documentation/dns/
[r-domain]: domain.html
[ttl]: https://en.wikipedia.org/wiki/Time_to_live
