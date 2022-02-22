---
layout: "exoscale"
page_title: "Exoscale: exoscale_domain"
sidebar_current: "docs-exoscale-domain"
description: |-
  Provides an Exoscale DNS Domain resource.
---

# exoscale\_domain

Provides an Exoscale [DNS][dns-doc] Domain resource. This can be used to create and delete DNS Domains.


## Usage example

```hcl
resource "exoscale_domain" "example" {
  name = "example.net"
}
```


## Arguments Reference

* `name` - (Required) The name of the DNS Domain.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `token` - A security token that can be used as an alternative way to manage DNS Domains via the Exoscale API.
* `state` - The state of the DNS Domain.
* `auto_renew` - Boolean indicating that the DNS Domain has automatic renewal enabled.
* `expires_on` - The date of expiration of the DNS Domain, if known.


## Import

An existing DNS Domain can be imported as a resource by name:

```console
$ terraform import exoscale_domain.example example.net
```

~> **NOTE:** importing a `exoscale_domain` resource will also import all related [`exoscale_domain_records`][r-domain_record] resources (except `NS` and `SOA`).


[dns-doc]: https://community.exoscale.com/documentation/dns/
[r-domain_record]: domain_record.html
