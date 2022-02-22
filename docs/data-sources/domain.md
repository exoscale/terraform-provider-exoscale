---
page_title: "Exoscale: exoscale_domain"
description: |-
  Provides information about a Domain.
---

# exoscale\_domain

Provides information on a domain name hosted on [Exoscale DNS][exo-dns].


## Example Usage

```hcl
data "exoscale_domain" "my-company-com" {
  name = "my-company.com"
}
```


## Arguments Reference

* `name` - (Required) The name of the domain.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the domain


[exo-dns]: https://www.exoscale.com/dns/
