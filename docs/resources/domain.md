---
page_title: "Exoscale: exoscale_domain"
description: |-
  Manage Exoscale DNS Domains.
---

# exoscale\_domain

Manage Exoscale [DNS](https://community.exoscale.com/documentation/dns/) Domains.


## Usage

```hcl
resource "exoscale_domain" "my_domain" {
  name = "my.domain"
}
```

Next step is to attach [domain records](./domain_record) to the domain.

Please refer to the [examples](../../examples/) directory for complete configuration examples.


## Arguments Reference

* `name` - (Required) The name of the DNS domain.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `auto_renew` - Boolean indicating that the DNS domain has automatic renewal enabled.
* `expires_on` - The date of expiration of the DNS domain, if known.
* `state` - The state of the DNS domain.
* `token` - A security token that can be used as an alternative way to manage DNS domains via the Exoscale API.


## Import

An existing DNS domain may be imported by `<name>`:

```console
$ terraform import \
  exoscale_domain.my_domain \
  my.domain
```

~> **NOTE:** importing an `exoscale_domain` resource will also import all related `exoscale_domain_record` resources (except `NS` and `SOA`).
