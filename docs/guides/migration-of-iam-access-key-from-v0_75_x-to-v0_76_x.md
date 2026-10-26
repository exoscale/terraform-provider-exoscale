---
page_title: iam_access_key migration guide
description: |-
    migrating from exoscale_iam_access_key to exoscale_iam_api_key, from provider version ~> 0.75.x to ~> 0.76.x
---

# Migrating from iam_access_key to iam_api_key (v0.75.x to v0.76.x)

Version 0.76.0 removes the `exoscale_iam_access_key` resource. It relied on the legacy IAM access keys,
which are superseded by IAM API keys bound to an IAM role:
[`exoscale_iam_api_key`](../resources/iam_api_key) and [`exoscale_iam_role`](../resources/iam_role).

~> **Note:** A state that still contains an `exoscale_iam_access_key` fails with provider 0.76.0 or later.
Follow the steps below **while still on provider 0.75.x**, before you upgrade.

## Step 1: Replace the access key with a role and an API key

The `operations`, `resources` and `tags` arguments of the access key become rules of the role policy.
Example given, an access key restricted to reading one bucket:

```terraform
resource "exoscale_iam_access_key" "my_sos_access_key" {
  name       = "my-sos-access-key"
  operations = ["get-object", "list-objects"]
  resources  = ["sos/bucket:my-bucket"]
}
```

becomes:

```terraform
resource "exoscale_iam_role" "my_sos_role" {
  name     = "my-sos-role"
  editable = true

  policy = {
    default_service_strategy = "deny"
    services = {
      sos = {
        type = "rules"
        rules = [
          {
            expression = "!(parameters.bucket == 'my-bucket')"
            action     = "deny"
          },
          {
            expression = "operation in ['get-object', 'list-objects']"
            action     = "allow"
          },
        ]
      }
    }
  }
}

resource "exoscale_iam_api_key" "my_sos_api_key" {
  name    = "my-sos-api-key"
  role_id = exoscale_iam_role.my_sos_role.id
}
```

The new API key has a different `key` and `secret`: update whatever consumes them
(e.g. outputs referencing `exoscale_iam_access_key.my_sos_access_key.secret`).

See the [iam-bucket-access example](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/iam-bucket-access)
for more policy rules.

## Step 2: Remove the access key from the configuration and the state

Pick one of the following, still on provider 0.75.x:

- **Revoke the legacy key**: delete the `exoscale_iam_access_key` block and run `terraform apply`.
  The key is revoked on Exoscale and removed from the state.
- **Keep the legacy key alive** (e.g. while clients switch to the new API key): delete the
  `exoscale_iam_access_key` block and run:

  ```shell
  terraform state rm exoscale_iam_access_key.my_sos_access_key
  ```

  The key is no longer managed by Terraform; revoke it later.

## Step 3: Upgrade the provider

Run `terraform state list | grep exoscale_iam_access_key` and make sure nothing is left, then bump the
provider version to `~> 0.76` and run `terraform init -upgrade`.
