---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "exoscale_nlb Data Source - terraform-provider-exoscale"
subcategory: ""
description: |-
  Fetch Exoscale Network Load Balancers (NLB) https://community.exoscale.com/product/networking/nlb/ data.
  Corresponding resource: exoscale_nlb ../resources/nlb.md.
---

# exoscale_nlb (Data Source)

Fetch Exoscale [Network Load Balancers (NLB)](https://community.exoscale.com/product/networking/nlb/) data.

Corresponding resource: [exoscale_nlb](../resources/nlb.md).

## Example Usage

```terraform
data "exoscale_nlb" "my_nlb" {
  zone = "ch-gva-2"
  name = "my-nlb"
}

output "my_nlb_id" {
  value = data.exoscale_nlb.my_nlb.id
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `zone` (String) The Exoscale [Zone](https://www.exoscale.com/datacenters/) name.

### Optional

- `id` (String) The Network Load Balancers (NLB) ID to match (conflicts with `name`).
- `name` (String) The NLB name to match (conflicts with `id`).

### Read-Only

- `created_at` (String) The NLB creation date.
- `description` (String) The Network Load Balancers (NLB) description.
- `ip_address` (String) The NLB public IPv4 address.
- `state` (String) The current NLB state.


