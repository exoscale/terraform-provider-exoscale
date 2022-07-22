---
page_title: "Exoscale: exoscale_domain_record"
description: |-
  manage Exoscale DNS Domain Records.
---

# exoscale\_domain\_record

Manage Exoscale [DNS](https://community.exoscale.com/documentation/dns/) Domain Records.


## Usage

```hcl
resource "exoscale_domain" "my_domain" {
  name = "example.net"
}

resource "exoscale_domain_record" "my_host" {
  domain      = exoscale_domain.my_domain.id
  name        = "my-host"
  record_type = "A"
  content     = "1.2.3.4"
}

resource "exoscale_domain_record" "my_host_alias" {
  domain      = exoscale_domain.my_domain.id
  name        = "my-host-alias"
  record_type = "CNAME"
  content     = exoscale_domain_record.my_host.hostname
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

* `domain` - (Required) The parent [domain](./domain.md) to create the record into.
* `name` - (Required) The name of the domain record; leave blank (`""`) to create a root record (similar to using `@` in a DNS zone file).
* `content` - (Required) The value of the domain record.
* `record_type` - (Required) The type of the domain record (`A`, `AAAA`, `ALIAS`, `CAA`, `CNAME`, `HINFO`, `MX`, `NAPTR`, `NS`, `POOL`, `SPF`, `SRV`, `SSHFP`, `TXT`, `URL`).

* `prio` - The priority of the DNS domain record (for types that support it; minimum `0`).
* `ttl` - The TTL (Time-to-Live) of the domain record (seconds; minimum `0`).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `hostname` - The DNS domain record's *Fully Qualified Domain Name* (FQDN). Useful for aliasing `A`/`AAAA` records with `CNAME`.


## Import

An existing DNS domain record may be imported by `<ID>`:

```console
$ terraform import \
  exoscale_domain_record.my_host \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6
```

~> **NOTE:** importing an `exoscale_domain` resource will also import all related `exoscale_domain_record` resources (except `NS` and `SOA`).
