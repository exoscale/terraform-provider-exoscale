---
page_title: "Exoscale: exoscale_domain"
description: |-
  Fetch Exoscale DNS Domains data.
---

# exoscale\_domain

Fetch Exoscale [DNS](https://community.exoscale.com/documentation/dns/) Domains data.

Corresponding resource: [exoscale_domain](../resources/domain.md).


## Usage

```hcl
data "exoscale_domain" "my_domain" {
  name = "my.domain"
}

output "my_domain_id" {
  value = data.exoscale_domain.my_domain.id
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

* `name` - (Required) The DNS domain name to match.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The DNS domain ID
