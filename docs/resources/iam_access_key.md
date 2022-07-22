---
page_title: "Exoscale: exoscale_iam_key"
description: |-
  Manage Exoscale IAM Access Keys
---

# exoscale\_iam\_access\_key

Manage Exoscale [IAM Access Keys](https://community.exoscale.com/documentation/iam/)

!> **WARNING:** This resource stores sensitive information in your Terraform state. Please be sure to correctly understand implications and how to mitigate potential risks before using it.


## Usage

```hcl
resource "exoscale_iam_access_key" "my_sos_access_key" {
  name       = "my-sos-access-key"
  operations = ["get-sos-object", "list-sos-bucket"]
  resources  = ["sos/bucket:my-bucket"]
}

resource "exoscale_iam_access_key" "my_sks_access_key" {
  name = "my-sks-access-key"
  tags = ["sks"]
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

[cli]: https://github.com/exoscale/cli/
[iam-resource]: https://community.exoscale.com/documentation/iam/quick-start/#restricting-api-access-keys-to-resources

* `name` - (Required) The IAM access key name.

* `operations` - A list of API operations to restrict the key to.
* `resources` - A list of API [resources][iam-resource] to restrict the key to (`<domain>/<type>:<name>`).
* `tags` - A list of tags to restrict the key to.

-> **NOTE:** You can retrieve the list of available operations and tags using the [Exoscale CLI][cli]: `exo iam access-key list-operations`.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `key` - The IAM access key (identifier).
* `secret` - (Sensitive) The key secret.
