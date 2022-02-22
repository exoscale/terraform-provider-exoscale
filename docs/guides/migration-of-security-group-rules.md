---
page_title: security_group_rules migration Guide
description: Migrating from security_group_rules to security_group
---

# Migrating from security_group_rules to security_group

This page helps you migrate from an `exoscale_security_group_rules` resource (which is deprecated) to a set of `exoscale_security_group_rule`.

~> **Note:** Before migrating `exoscale_security_group_rules` resources you need to ensure you use the latest version of Terraform and the provider, and have a clean configuration.

Before proceeding, please:
- Upgrade Terraform to at least v1.1.x: allows us to easily refactor definitions thanks to `moved {}` blocks.
- Upgrade the Exoscale provider to at least v0.31.3: allow us to run [`terraform import`](https://www.terraform.io/docs/commands/import.html) on `exoscale_security_group` resources. 
- Ensure your configuration can be successfully applied: [`terraform plan`](https://www.terraform.io/docs/commands/plan.html) must NOT output errors, changes, or moves of resources
- Have the [Exoscale cli](https://github.com/exoscale/cli/) to retrieve IDs of security groups (optional, as you can also retrieve this information from the [`Exoscale portal`](https://portal.exoscale.com/login)).
- Perform a backup of your state! We are going to manipulate it, and errors can always happen. Remote states can be retrieved with [`terraform state pull` and restored with `terraform state push`](https://www.terraform.io/cli/state/recover). If you keep your state as a single local file, a regular file copy can do the job.

Update your provider version to latest `0.31.x` version:

```hcl
terraform {
    required_providers {
        exoscale = {
            source = "exoscale/exoscale"
            version = "~> 0.31.3"
        }
    }
}
```

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


## Migration plan

To achieve this migration, we have to remove from the state:
- `exoscale_security_group.webapp`
- **ALL rules** that belongs to `exoscale_security_group.webapp`. In our case: `exoscale_security_group_rules.webapp`

After these resources are removed from the state, we will have to import `exoscale_security_group.webapp` back.
The import process will also automatically import related security group rules as `exoscale_security_group_rule` resources.
The name for rules will be the same as the security group, suffixed with a number except for the first one.
In our case, we have 3 rules, and the security group has `webapp` as a name, so we will have `exoscale_security_group_rule` 
resources imported as `webapp`, `webapp-1`, and `webapp-2`.

Just after the import process, we will have to adjust the configuration to reflect the real infrastructure.
Thanks to `moved {}` blocks we will be able to easily move `security_group_rule` resources to whatever name we want. 

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

### Importing the security group and related rules into the state

Using the [Exoscale cli](https://github.com/exoscale/cli/) or the [`Exoscale portal`](https://portal.exoscale.com/login), we can find the ID of the webapp security group:

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
#   Prepared exoscale_security_group_rule for import
#   Prepared exoscale_security_group_rule for import
#   Prepared exoscale_security_group_rule for import
# exoscale_security_group.webapp: Refreshing state... [id=b83fe506-51e6-4933-8e70-205df6b640fa]
# exoscale_security_group_rule.webapp-1: Refreshing state... [id=e7bbda2b-3c93-4693-a54c-465ac28bda59]
# exoscale_security_group_rule.webapp: Refreshing state... [id=5dafa58d-ba4d-4990-b66d-2996782ee3f3]
# exoscale_security_group_rule.webapp-2: Refreshing state... [id=0a187bd0-b5dc-4a58-bab2-44bc7064b83b]
#
# Import successful!
#
# The resources that were imported are shown above. These resources are now in
# your Terraform state and will henceforth be managed by Terraform.
```

As you can see, and as expected, the import process imported not only `exoscale_security_group.webapp` but also related rules as `exoscale_security_group_rule` resources: `webapp`, `webapp-1`, and `webapp-2`.
In order to check this result and display details on newly imported rules, we have to run [`terraform plan`](https://www.terraform.io/docs/commands/plan.html):

```bash
terraform plan
# [output]                                                                      
# exoscale_security_group.bastion: Refreshing state... [id=e1654b58-64f6-4efc-b8d9-7bbda86a83fa]
# exoscale_security_group.webapp: Refreshing state... [id=b83fe506-51e6-4933-8e70-205df6b640fa]
# exoscale_security_group_rule.webapp-1: Refreshing state... [id=e7bbda2b-3c93-4693-a54c-465ac28bda59]
# exoscale_security_group_rule.webapp: Refreshing state... [id=5dafa58d-ba4d-4990-b66d-2996782ee3f3]
# exoscale_security_group_rule.webapp-2: Refreshing state... [id=0a187bd0-b5dc-4a58-bab2-44bc7064b83b]
#
# Terraform used the selected providers to generate the following execution plan. Resource actions are indicated with the following symbols:
#   + create
#   - destroy
#
# Terraform will perform the following actions:
#
#   # exoscale_security_group_rule.webapp will be destroyed
#   # (because exoscale_security_group_rule.webapp is not in configuration)
#   - resource "exoscale_security_group_rule" "webapp" {
#       - end_port               = 22 -> null
#       - id                     = "5dafa58d-ba4d-4990-b66d-2996782ee3f3" -> null
#       - protocol               = "TCP" -> null
#       - security_group         = "webapp" -> null
#       - security_group_id      = "b83fe506-51e6-4933-8e70-205df6b640fa" -> null
#       - start_port             = 22 -> null
#       - type                   = "INGRESS" -> null
#       - user_security_group    = "bastion" -> null
#       - user_security_group_id = "e1654b58-64f6-4efc-b8d9-7bbda86a83fa" -> null
#
#       - timeouts {}
#     }
#
#   # exoscale_security_group_rule.webapp-1 will be destroyed
#   # (because exoscale_security_group_rule.webapp-1 is not in configuration)
#   - resource "exoscale_security_group_rule" "webapp-1" {
#       - cidr              = "0.0.0.0/0" -> null
#       - end_port          = 443 -> null
#       - id                = "e7bbda2b-3c93-4693-a54c-465ac28bda59" -> null
#       - protocol          = "TCP" -> null
#       - security_group    = "webapp" -> null
#       - security_group_id = "b83fe506-51e6-4933-8e70-205df6b640fa" -> null
#       - start_port        = 443 -> null
#       - type              = "INGRESS" -> null
#
#       - timeouts {}
#     }
#
#   # exoscale_security_group_rule.webapp-2 will be destroyed
#   # (because exoscale_security_group_rule.webapp-2 is not in configuration)
#   - resource "exoscale_security_group_rule" "webapp-2" {
#       - cidr              = "0.0.0.0/0" -> null
#       - end_port          = 80 -> null
#       - id                = "0a187bd0-b5dc-4a58-bab2-44bc7064b83b" -> null
#       - protocol          = "TCP" -> null
#       - security_group    = "webapp" -> null
#       - security_group_id = "b83fe506-51e6-4933-8e70-205df6b640fa" -> null
#       - start_port        = 80 -> null
#       - type              = "INGRESS" -> null
#
#       - timeouts {}
#     }
#
#   # exoscale_security_group_rules.webapp will be created
#   + resource "exoscale_security_group_rules" "webapp" {
#       + id                = (known after apply)
#       + security_group    = (known after apply)
#       + security_group_id = "b83fe506-51e6-4933-8e70-205df6b640fa"
#
#       + ingress {
#           + cidr_list                = [
#               + "0.0.0.0/0",
#             ]
#           + ids                      = (known after apply)
#           + ports                    = [
#               + "443",
#             ]
#           + protocol                 = "TCP"
#           + user_security_group_list = []
#         }
#       + ingress {
#           + cidr_list                = [
#               + "0.0.0.0/0",
#             ]
#           + ids                      = (known after apply)
#           + ports                    = [
#               + "80",
#             ]
#           + protocol                 = "TCP"
#           + user_security_group_list = []
#         }
#       + ingress {
#           + cidr_list                = []
#           + ids                      = (known after apply)
#           + ports                    = [
#               + "22",
#             ]
#           + protocol                 = "TCP"
#           + user_security_group_list = [
#               + "bastion",
#             ]
#         }
#     }
#
# Plan: 1 to add, 0 to change, 3 to destroy.
```

For the time being, terraform tries to remove imported `exoscale_security_group_rule` resources, and re-create the `exoscale_security_group_rules` resource.
We have to update our definition in such a way that the code reflects the actual Terraform state.

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

### Move imported resources to match definitions

At this moment, we still have some migration to do. Indeed, we defined `webapp_ssh`, and `webapp_public` but imported resources are `webapp`, `webapp-1` and `webapp-2`.
We must tell Terraform how to match new definitions with resources that were imported into the state.

We can use [`moved` blocks](https://www.terraform.io/language/modules/develop/refactoring#moved-block-syntax) for this purpose (see also [Hashicorp learn](https://learn.hashicorp.com/tutorials/terraform/move-config)).

In the latest plan that we asked from Terraform (see above), we can see that:
- `exoscale_security_group_rule` for port 22 was imported as `exoscale_security_group_rule.webapp` (instead of `exoscale_security_group_rule.webapp_ssh`)
- `exoscale_security_group_rule` for port 80 was imported as `exoscale_security_group_rule.webapp-2` (instead of `exoscale_security_group_rule.webapp_public["80"]`)
- `exoscale_security_group_rule` for port 443 was imported as `exoscale_security_group_rule.webapp-1` (instead of `exoscale_security_group_rule.webapp_public["443"]`)

We have to add `moved {}` blocks to update the state according to these observations:

```hcl
moved {
  from = exoscale_security_group_rule.webapp
  to = exoscale_security_group_rule.webapp_ssh
}

moved {
  from = exoscale_security_group_rule.webapp-1
  to = exoscale_security_group_rule.webapp_public["443"]
}

moved {
  from = exoscale_security_group_rule.webapp-2
  to = exoscale_security_group_rule.webapp_public["80"]
}
```

Once our definitions are updated, we can ask Terraform a new plan, to check what will be done:

```bash
terraform plan
# [output]
# exoscale_security_group.webapp: Refreshing state... [id=b83fe506-51e6-4933-8e70-205df6b640fa]
# exoscale_security_group.bastion: Refreshing state... [id=e1654b58-64f6-4efc-b8d9-7bbda86a83fa]
# exoscale_security_group_rule.webapp_public["80"]: Refreshing state... [id=0a187bd0-b5dc-4a58-bab2-44bc7064b83b]
# exoscale_security_group_rule.webapp_ssh: Refreshing state... [id=5dafa58d-ba4d-4990-b66d-2996782ee3f3]
# exoscale_security_group_rule.webapp_public["443"]: Refreshing state... [id=e7bbda2b-3c93-4693-a54c-465ac28bda59]
#
# Terraform will perform the following actions:
#
#   # exoscale_security_group_rule.webapp-1 has moved to exoscale_security_group_rule.webapp_public["443"]
#     resource "exoscale_security_group_rule" "webapp_public" {
#         id                = "e7bbda2b-3c93-4693-a54c-465ac28bda59"
#         # (7 unchanged attributes hidden)
#
#         # (1 unchanged block hidden)
#     }
#
#   # exoscale_security_group_rule.webapp-2 has moved to exoscale_security_group_rule.webapp_public["80"]
#     resource "exoscale_security_group_rule" "webapp_public" {
#         id                = "0a187bd0-b5dc-4a58-bab2-44bc7064b83b"
#         # (7 unchanged attributes hidden)
#
#         # (1 unchanged block hidden)
#     }
#
#   # exoscale_security_group_rule.webapp has moved to exoscale_security_group_rule.webapp_ssh
#     resource "exoscale_security_group_rule" "webapp_ssh" {
#         id                     = "5dafa58d-ba4d-4990-b66d-2996782ee3f3"
#         # (8 unchanged attributes hidden)
#
#         # (1 unchanged block hidden)
#     }
#
# Plan: 0 to add, 0 to change, 0 to destroy.
#
# ────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────
#
# Note: You didn't use the -out option to save this plan, so Terraform can't guarantee to take exactly these actions if you run "terraform apply" now.
```

Here we can see that our configuration is matching the state: no resources will be created, only moves will occur. We can apply moves, using `terraform apply`:

```bash
terraform apply
# [output]
# exoscale_security_group.webapp: Refreshing state... [id=b83fe506-51e6-4933-8e70-205df6b640fa]
# exoscale_security_group.bastion: Refreshing state... [id=e1654b58-64f6-4efc-b8d9-7bbda86a83fa]
# exoscale_security_group_rule.webapp_ssh: Refreshing state... [id=5dafa58d-ba4d-4990-b66d-2996782ee3f3]
# exoscale_security_group_rule.webapp_public["80"]: Refreshing state... [id=0a187bd0-b5dc-4a58-bab2-44bc7064b83b]
# exoscale_security_group_rule.webapp_public["443"]: Refreshing state... [id=e7bbda2b-3c93-4693-a54c-465ac28bda59]
#
# Terraform will perform the following actions:
#
#   # exoscale_security_group_rule.webapp-1 has moved to exoscale_security_group_rule.webapp_public["443"]
#     resource "exoscale_security_group_rule" "webapp_public" {
#         id                = "e7bbda2b-3c93-4693-a54c-465ac28bda59"
#         # (7 unchanged attributes hidden)
#
#         # (1 unchanged block hidden)
#     }
#
#   # exoscale_security_group_rule.webapp-2 has moved to exoscale_security_group_rule.webapp_public["80"]
#     resource "exoscale_security_group_rule" "webapp_public" {
#         id                = "0a187bd0-b5dc-4a58-bab2-44bc7064b83b"
#         # (7 unchanged attributes hidden)
#
#         # (1 unchanged block hidden)
#     }
#
#   # exoscale_security_group_rule.webapp has moved to exoscale_security_group_rule.webapp_ssh
#     resource "exoscale_security_group_rule" "webapp_ssh" {
#         id                     = "5dafa58d-ba4d-4990-b66d-2996782ee3f3"
#         # (8 unchanged attributes hidden)
#
#         # (1 unchanged block hidden)
#     }
#
# Plan: 0 to add, 0 to change, 0 to destroy.
#
# Do you want to perform these actions?
#   Terraform will perform the actions described above.
#   Only 'yes' will be accepted to approve.
#
#   Enter a value: yes
#
#
# Apply complete! Resources: 0 added, 0 changed, 0 destroyed.
```

Now we have finished replacing `exoscale_security_group_rules` with a set of `exoscale_security_group_rule`.
Our Terraform state matches the related configuration, so migration is completely done, and `moved {}`
blocks can now be removed from the configuration.
