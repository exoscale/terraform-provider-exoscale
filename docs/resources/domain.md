---
page_title: "exoscale_domain Resource - terraform-provider-exoscale"
subcategory: ""
description: |-
  Manage Exoscale DNS Domains.
---

# exoscale_domain (Resource)

Manage Exoscale [DNS](https://community.exoscale.com/product/networking/dns/) Domains.

Corresponding data source: [exoscale_domain](../data-sources/domain.md).

## Example Usage

```terraform
resource "exoscale_domain" "my_domain" {
  name = "my.domain"
}
```

Next step is to attach [exoscale_domain_record](./domain_record.md)(s) to the domain.

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) ❗ The DNS domain name.

### Optional

- `timeouts` (Block, Optional) (see [below for nested schema](#nestedblock--timeouts))

### Read-Only

- `auto_renew` (Boolean, Deprecated) Whether the DNS domain has automatic renewal enabled (boolean).
- `expires_on` (String, Deprecated) The domain expiration date, if known.
- `id` (String) The ID of this resource.
- `state` (String, Deprecated) The domain state.
- `token` (String, Deprecated) A security token that can be used as an alternative way to manage DNS domains via the Exoscale API.

<a id="nestedblock--timeouts"></a>
### Nested Schema for `timeouts`

Optional:

- `create` (String)
- `delete` (String)
- `read` (String)

-> The symbol ❗ in an attribute indicates that modifying it, will force the creation of a new resource.

## Import

An existing DNS domain may be imported by `ID`:

```shell
terraform import \
  exoscale_domain.my_domain \
  89083a5c-b648-474a-0000-0000000f67bd
```
