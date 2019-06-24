---
layout: "exoscale"
page_title: "Exoscale: exoscale_domain_record"
sidebar_current: "docs-exoscale-domain-record"
description: |-
  Provides an Exoscale DNS Domain Record resource.
---

# exoscale\_domain\_record

Provides an Exoscale [DNS][dns] Domain Record resource. This can be used to create, modify, and delete DNS Domain Records.

[dns]: https://community.exoscale.com/documentation/dns/

## Usage example

```hcl
resource "exoscale_domain" "example" {
  name = "example.net"
}

resource "exoscale_domain_record" "myserver" {
  domain      = "${exoscale_domain.example.id}"
  name        = "myserver"
  record_type = "A"
  content     = "1.2.3.4"
}

resource "exoscale_domain_record" "myserver_alias" {
  domain      = "${exoscale_domain.example.id}"
  name        = "myserver-new"
  record_type = "CNAME"
  content     = "${exoscale_domain_record.myserver.hostname}"
}
```

## Argument Reference

* `domain` - (Required) The name of the [`exoscale_domain`][domain] to create the record into.
* `name` - (Required) The name of the DNS Domain Record.
* `record_type` - (Required) The type of the DNS Domain Record. Supported values are: `A`, `AAAA`, `ALIAS`, `CAA`, `CNAME`, `HINFO`, `MX`, `NAPTR`, `NS`, `POOL`, `SPF`, `SRV`, `SSHFP`, `TXT`, `URL`.
* `content` - (Required) The value of the DNS Domain Record.
* `ttl` - The [Time To Live][ttl] of the DNS Domain Record.
* `prio` - The priority of the DNS Domain Record (for types that support it).

[domain]: domain.html
[ttl]: https://en.wikipedia.org/wiki/Time_to_live

## Attributes Reference

The following attributes are exported:

* `hostname` - The DNS Domain Record's *Fully Qualified Domain Name* (FQDN), useful for linking `A` records into `CNAME`.

## Import

An existing DNS Domain Record can be imported as a resource by ID:

```console
$ terraform import exoscale_domain_record.www 12480484
```

~> **NOTE:** importing an existing [`exoscale_domain`][domain] resource also imports linked `exoscale_domain_record` resources.

[domain]: domain.html
