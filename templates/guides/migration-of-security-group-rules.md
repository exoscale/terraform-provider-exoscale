---
page_title: security_group_rules migration Guide
description: |-
  Migrating from security_group_rules to security_group
---

# Migrating from security_group_rules to security_group

-> This migration guide applies to Exoscale Terraform Provider **version 0.31.3 to 0.53.2**.

This page helps you migrate from an `exoscale_security_group_rules` resource (which is deprecated) to a set of `exoscale_security_group_rule`.

~> **Note:** Before migrating `exoscale_security_group_rules` resources you need to ensure you use the latest version of Terraform and have a clean configuration.

Before proceeding, please:
- Upgrade the Exoscale provider to at least v0.31.3: allow us to run [`terraform import`](https://www.terraform.io/docs/commands/import.html) on `exoscale_security_group` and `exoscale_security_group_rule` resources.
- Ensure your configuration can be successfully applied: [`terraform plan`](https://www.terraform.io/docs/commands/plan.html) must NOT output errors, changes, or moves of resources
- Have the [Exoscale cli](https://github.com/exoscale/cli/) to retrieve IDs of security groups and security group rules
- Perform a backup of your state! We are going to manipulate it, and errors can always happen. Remote states can be retrieved with [`terraform state pull` and restored with `terraform state push`](https://www.terraform.io/cli/state/recover). If you keep your state as a single local file, a regular file copy can do the job.

## Example configuration

In this guide, we will assume the following configuration as an example:

```hcl
resource "exoscale_security_group" "webapp" {
  name = "webapp"
  # ...
}

resource "exoscale_security_group" "bastion" {
  name = "bastion"
  # ...
}

resource "exoscale_security_group_rules" "webapp" {
  security_group_id = exoscale_security_group.webapp.id

  ingress {
    ports = ["22"]
    protocol = "TCP"
    user_security_group_list = [exoscale_security_group.bastion.name]
  }

  ingress {
    ports = ["80"]
    protocol = "TCP"
    cidr_list = ["0.0.0.0/0"]
  }

  ingress {
    ports = ["443"]
    protocol = "TCP"
    cidr_list = ["0.0.0.0/0"]
  }
}
```

As you can see, we have a security group with an associated `security_group_rules` resource that:
- allows access from every IPv4 address to ports 80 and 443, and
- allows access to port 22 from hosts that belongs to another security group named `bastion`.

From the [Exoscale cli](https://github.com/exoscale/cli/), we can retrieve information for every security group:

```bash
exo compute sg list
# [output]
# ┼──────────────────────────────────────┼─────────┼
# │                  ID                  │  NAME   │
# ┼──────────────────────────────────────┼─────────┼
# │ b83fe506-51e6-4933-8e70-205df6b640fa │ webapp  │
# │ e1654b58-64f6-4efc-b8d9-7bbda86a83fa │ bastion │
# │ a0191fd7-1275-4614-9fed-be6f19a770a0 │ default │
# ┼──────────────────────────────────────┼─────────┼
```

The Terraform state contains them as well:

```bash
terraform state list
# [output]
# exoscale_security_group.bastion
# exoscale_security_group.webapp
# exoscale_security_group_rules.webapp
```

We want to migrate the single `exoscale_security_group_rules.webapp` to 3 different `exoscale_security_group_rule` resources, as we have 3 different ingress rules.

~> **Note:** If in your `exoscale_security_group_rules` definition `ingress` and/or `egress` blocks define multiple ports or port ranges (as `ports` attribute is array), you need to define one `exoscale_security_group_rule` for each port or port range as multiple values per rule are no longer allowed.


## Migration plan

To achieve this migration, we have to remove from the state:
- `exoscale_security_group.webapp`
- **ALL rules** that belongs to `exoscale_security_group.webapp`. In our case: `exoscale_security_group_rules.webapp`

After these resources are removed from the state, we will have to import `exoscale_security_group.webapp` back as well as related security group rules as `exoscale_security_group_rule` resources.

## Applying the migration plan

### Removing the security group from the state

This step is pretty straightforward: we will have to issue a `terraform state rm` command for each resources we have to
remove from the state. In our example, we have to remove `exoscale_security_group.webapp`, and `exoscale_security_group_rules.webapp`:

```bash
terraform state rm exoscale_security_group_rules.webapp
# [output]
# Removed exoscale_security_group_rules.webapp
# Successfully removed 1 resource instance(s).

terraform state rm exoscale_security_group.webapp
# [output]
# Removed exoscale_security_group.webapp
# Successfully removed 1 resource instance(s).
```

Now, these resources are removed from the state.

### Update infrastructure definition

Replace the `exoscale_security_group_rules` block by new `exoscale_security_group_rule` resources:

```hcl
resource "exoscale_security_group_rule" "webapp_ssh" {
  security_group_id = exoscale_security_group.webapp.id
  type = "INGRESS"
  start_port = 22
  end_port   = 22
  protocol = "TCP"
  user_security_group_id = exoscale_security_group.bastion.id
}

resource "exoscale_security_group_rule" "webapp_public" {
  for_each = toset(["80", "443"])

  security_group_id = exoscale_security_group.webapp.id
  type = "INGRESS"
  start_port = each.value
  end_port   = each.value
  protocol = "TCP"
  cidr = "0.0.0.0/0"
}
```

Note that we can of course imagine 3 single resource blocks: `webapp_http` and `webapp_https` replacing `webapp_public` in addition to `webapp_ssh`.
In our example we tried instead to follow the semantic behind each rule, and group them:
- `webapp_ssh` allows access to SSH through a bastion security group
- `webapp_public` allows access to HTTP(S) services, on both port 80 and 443, leveraging the [`for_each` notation of Terraform](https://www.terraform.io/language/meta-arguments/for_each).

### Importing the security group and related rules into the state

Using the [Exoscale cli](https://github.com/exoscale/cli/), we can find the ID of the webapp security group:

```bash
exo compute sg list
# [output]
# ┼──────────────────────────────────────┼─────────┼
# │                  ID                  │  NAME   │
# ┼──────────────────────────────────────┼─────────┼
# │ b83fe506-51e6-4933-8e70-205df6b640fa │ webapp  │
# │ e1654b58-64f6-4efc-b8d9-7bbda86a83fa │ bastion │
# │ a0191fd7-1275-4614-9fed-be6f19a770a0 │ default │
# ┼──────────────────────────────────────┼─────────┼
```

In our case, it's `b83fe506-51e6-4933-8e70-205df6b640fa`. Let's import this resource:

```bash
terraform import exoscale_security_group.webapp b83fe506-51e6-4933-8e70-205df6b640fa
# [output]
# exoscale_security_group.webapp: Importing from ID "b83fe506-51e6-4933-8e70-205df6b640fa"...
# exoscale_security_group.webapp: Import prepared!
#   Prepared exoscale_security_group for import
# exoscale_security_group.webapp: Refreshing state... [id=b83fe506-51e6-4933-8e70-205df6b640fa]
#
# Import successful!
#
# The resources that were imported are shown above. These resources are now in
# your Terraform state and will henceforth be managed by Terraform.
```

As you can see, and as expected, the import process imported `exoscale_security_group.webapp`.
We also need to import related `exoscale_security_group_rule` resources. To find their IDs, we must use CLI again:

```bash
exo c security-group show b83fe506-51e6-4933-8e70-205df6b640fa
┼──────────────────┼─────────────────────────────────────────────────────────────────────┼
│  SECURITY GROUP  │                                                                     │
┼──────────────────┼─────────────────────────────────────────────────────────────────────┼
│ ID               │ b83fe506-51e6-4933-8e70-205df6b640fa                                │
│ Name             │ webapp                                                              │
│ Description      │                                                                     │
│ Ingress Rules    │                                                                     │
│                  │   5dafa58d-ba4d-4990-b66d-2996782ee3f3      TCP   0.0.0.0/0   22    │
│                  │   e7bbda2b-3c93-4693-a54c-465ac28bda59      TCP   0.0.0.0/0   443   │
│                  │   0a187bd0-b5dc-4a58-bab2-44bc7064b83b      TCP   0.0.0.0/0   80    │
│                  │                                                                     │
│ Egress Rules     │ -                                                                   │
│ External Sources │ -                                                                   │
┼──────────────────┼─────────────────────────────────────────────────────────────────────┼
```

Security group rule IDs can by found in `Ingress Rules`: `5dafa58d-ba4d-4990-b66d-2996782ee3f3`, `e7bbda2b-3c93-4693-a54c-465ac28bda59` and `0a187bd0-b5dc-4a58-bab2-44bc7064b83b`.
Now we can import them one by one:

```bash
terraform import exoscale_security_group_rule.webapp_ssh b83fe506-51e6-4933-8e70-205df6b640fa/5dafa58d-ba4d-4990-b66d-2996782ee3f3
# [output]
# exoscale_security_group_rule.webapp_ssh: Importing from ID "b83fe506-51e6-4933-8e70-205df6b640fa/5dafa58d-ba4d-4990-b66d-2996782ee3f3"...
# exoscale_security_group_rule.webapp_ssh: Import prepared!
#   Prepared exoscale_security_group_rule for import
# exoscale_security_group_rule.webapp_ssh: Refreshing state... [id=5dafa58d-ba4d-4990-b66d-2996782ee3f3]
#
# Import successful!
#
# The resources that were imported are shown above. These resources are now in
# your Terraform state and will henceforth be managed by Terraform.

terraform import 'exoscale_security_group_rule.webapp_public["443"]' b83fe506-51e6-4933-8e70-205df6b640fa/e7bbda2b-3c93-4693-a54c-465ac28bda59
# [output]
# exoscale_security_group_rule.webapp_public["443"]: Importing from ID "b83fe506-51e6-4933-8e70-205df6b640fa/e7bbda2b-3c93-4693-a54c-465ac28bda59"...
# exoscale_security_group_rule.webapp_public["443"]: Import prepared!
#   Prepared exoscale_security_group_rule for import
# exoscale_security_group_rule.webapp_public["443"]: Refreshing state... [id=e7bbda2b-3c93-4693-a54c-465ac28bda59]
#
# Import successful!
#
# The resources that were imported are shown above. These resources are now in
# your Terraform state and will henceforth be managed by Terraform.

terraform import 'exoscale_security_group_rule.webapp_public["80"]' b83fe506-51e6-4933-8e70-205df6b640fa/0a187bd0-b5dc-4a58-bab2-44bc7064b83b
# [output]
# exoscale_security_group_rule.webapp_public["80"]: Importing from ID "b83fe506-51e6-4933-8e70-205df6b640fa/0a187bd0-b5dc-4a58-bab2-44bc7064b83b"...
# exoscale_security_group.webapp_public["80"]: Import prepared!
#   Prepared exoscale_security_group_rule for import
# exoscale_security_group_rule.webapp_public["80"]: Refreshing state... [id=0a187bd0-b5dc-4a58-bab2-44bc7064b83b]
#
# Import successful!
#
# The resources that were imported are shown above. These resources are now in
# your Terraform state and will henceforth be managed by Terraform.
```

In order to check this result and display details on newly imported rules, we have to run [`terraform plan`](https://www.terraform.io/docs/commands/plan.html):

```bash
terraform apply
# [output]
# exoscale_security_group.webapp: Refreshing state... [id=b83fe506-51e6-4933-8e70-205df6b640fa]
# exoscale_security_group.bastion: Refreshing state... [id=e1654b58-64f6-4efc-b8d9-7bbda86a83fa]
# exoscale_security_group_rule.webapp_ssh: Refreshing state... [id=5dafa58d-ba4d-4990-b66d-2996782ee3f3]
# exoscale_security_group_rule.webapp_public["80"]: Refreshing state... [id=0a187bd0-b5dc-4a58-bab2-44bc7064b83b]
# exoscale_security_group_rule.webapp_public["443"]: Refreshing state... [id=e7bbda2b-3c93-4693-a54c-465ac28bda59]
#
# No changes. Your infrastructure matches the configuration.
#
# Apply complete! Resources: 0 added, 0 changed, 0 destroyed.
```

Here we can see that our configuration is matching the state: no resources will be changed.
Now we have finished replacing `exoscale_security_group_rules` with a set of `exoscale_security_group_rule`.
Our Terraform state matches the related configuration, so migration is completely done.
