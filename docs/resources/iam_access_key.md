---
page_title: "Exoscale: exoscale_iam_key"
description: |-
  Provides an Exoscale IAM Access Key and Secret
---

# exoscale\_iam\_access\_key

Provides an Exoscale [IAM Access Key][exo-iam] resource. This can be used to create, update and delete an IAM Access Key and its associated Secret.
You can retrieve the list of available operations or tags, using the Exoscale CLI, running `exo iam access-key list-operations`.

!> **WARNING:** This resource stores sensitive information in your Terraform state. Please be sure to correctly understand implications and how to mitigate potential risks before using it.

## Usage

```hcl
resource "exoscale_iam_access_key" "example" {
  name       = "backup-read-only"
  operations = ["get-sos-object", "list-sos-bucket"]
  resources  = ["sos/bucket:example-backup-bucket"]
}
```

```hcl
resource "exoscale_iam_access_key" "example" {
  name        = "sks-management"
  tags        = ["sks"]
}
```

## Arguments Reference

* `name` - (Required) The name of the Access Key.
* `operations` - A list of API operations to restrict the access key to.
* `resources` - A list of API resource to restrict the access key to (format: DOMAIN/TYPE:NAME). This format is the same as [in the CLI][exo-cli-resources]
* `tags` - A list of tags to restrict the access key to.

## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `key` - The IAM key.
* `secret` - (Sensitive) The IAM key secret.

[exo-iam]: https://community.exoscale.com/documentation/iam/quick-start/
[exo-cli-resources]: https://community.exoscale.com/documentation/iam/quick-start/#restricting-api-access-keys-to-resources