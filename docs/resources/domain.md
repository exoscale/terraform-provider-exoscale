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

Next step is to attach [exoscale_domain_record](./domain_record.md)(s) to the domain.

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

* `name` - (Required) The DNS domain name.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `auto_renew` - Whether the DNS domain has automatic renewal enabled (boolean).
* `expires_on` - The domain expiration date, if known.
* `state` - The domain state.
* `token` - A security token that can be used as an alternative way to manage DNS domains via the Exoscale API.


## Import

An existing DNS domain may be imported by `<name>`:

```console
$ terraform import \
  exoscale_domain.my_domain \
  my.domain
```

~> **NOTE:** importing an `exoscale_domain` resource will also import all related `exoscale_domain_record` resources (except `NS` and `SOA`).
