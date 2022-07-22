---
page_title: "Exoscale: exoscale_security_group"
description: |-
  Fetch Exoscale Security Groups data.
---

# exoscale\_security\_group

Fetch Exoscale [Security Groups](https://community.exoscale.com/documentation/compute/security-groups/) data.

Corresponding resource: [exoscale_security_group](../resources/security_group.md).


## Usage

```hcl
data "exoscale_security_group" "my_security_group" {
  name = "my-security-group"
}

output "my_security_group_id" {
  value = data.exoscale_security_group.my_security_group.id
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

* `id` - The security group ID to match (conflicts with `name`)
* `name` - The name to match (conflicts with `id`)


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* n/a
