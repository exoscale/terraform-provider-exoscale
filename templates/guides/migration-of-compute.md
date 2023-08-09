---
page_title: compute migration Guide
description: |-
  Migrating compute and nic to a compute_instance
---

# Migrating from compute with nic to a compute_instance

-> This migration guide applies to Exoscale Terraform Provider **version 0.34.0 or above**.

This page helps you migrate from an `exoscale_compute`, `exoscale_network` and `exoscale_nic` resources (which are deprecated) to the `exoscale_compute_instance` and `exoscale_private_network` resources.

~> **Note:** Before migrating resources you need to ensure you use the latest version of Terraform and have a clean configuration.

Before proceeding, please:
- Upgrade the Exoscale provider to at least v0.34.0: allow us to run [`terraform import`](https://www.terraform.io/docs/commands/import.html) on `exoscale_compute_instance` resource;
- Ensure your configuration can be successfully applied: [`terraform plan`](https://www.terraform.io/docs/commands/plan.html) must NOT output errors, changes, or moves of resources;
- Have the [Exoscale cli](https://github.com/exoscale/cli/) to retrieve IDs of compute instance and private network;
- Perform a backup of your state! We are going to manipulate it, and errors can always happen. Remote states can be retrieved with [`terraform state pull` and restored with `terraform state push`](https://www.terraform.io/cli/state/recover). If you keep your state as a single local file, a regular file copy can do the job.

## Example configuration

In this guide, we will assume the following configuration as an example:

```hcl
resource "exoscale_compute" "my_instance" {
  disk_size = 10
  display_name = "my-instance"
  security_group_ids = ["493ac3c0-c8ba-415a-a038-6578194a6d36"]
  size = "Tiny"
  template_id = "ee73810b-0245-43e0-8b15-0632473d56ba"
  zone = "ch-gva-2"
}

resource "exoscale_network" "my_network" {
  name = "privnet"
  display_text = "Private Network"
  zone = "ch-gva-2"
}

resource "exoscale_nic" "my_nic" {
  compute_id = exoscale_compute.my_instance.id
  network_id = exoscale_network.my_network.id
}
```

As you can see, we have a single `exoscale_compute` (instance) attached to the `exoscale_network` (private network) using `exoscale_nic` resource.

The Terraform state contains them as well:

```bash
$ terraform state list
exoscale_compute.my_instance
exoscale_network.my_network
exoscale_nic.my_nic
```

We want to migrate `exoscale_compute` and `exoscale_nic` to the `exoscale_compute_instance` resource and `exoscale_network` to the `exoscale_private_network` resource.


## Migration plan

To achieve this migration, we have to remove all 3 deprecated resources from the state and then import them as `exoscale_compute_instance` and `exoscale_private_network`.

## Applying the migration plan

### Removing the security group from the state 

This step is pretty straightforward: we will have to issue a `terraform state rm` command for each resources we have to remove from the state. In our example, we have to remove `exoscale_compute.my_instance`, `exoscale_network.my_network` and `exoscale_nic.my_nic`:

```bash
$ terraform state rm exoscale_compute.my_instance
Removed exoscale_compute.my_instance
Successfully removed 1 resource instance(s).

q$ terraform state rm exoscale_network.my_network
Removed exoscale_network.my_network
Successfully removed 1 resource instance(s).

$ terraform state rm exoscale_nic.my_nic
Removed exoscale_nic.my_nic
Successfully removed 1 resource instance(s).
```

Now, these resources are removed from the state.

### Update infrastructure definition

Replace the `exoscale_network` block by the new `exoscale_private_network` resource:

```hcl
resource "exoscale_private_network" "my_network" {
  name = "privnet"
  description = "Private Network"
  zone = "ch-gva-2"
}
```

In this example we are using unmanaged private network, for managed network you can copy values from `exoscale_network` definition with the following changes:
- `display_text` must be renamed to `description`;
- `tags` must be renamed to `labels`;
- `network_offering` is deprecated and should be removed.

Now replace `exoscale_compute` and `exoscale_nic` blocks with `exoscale_compute_instance`:

```hcl
resource "exoscale_compute_instance" "my_instance" {
	disk_size = 10
  name = "my-instance"
  security_group_ids = ["493ac3c0-c8ba-415a-a038-6578194a6d36"]
  template_id = "ee73810b-0245-43e0-8b15-0632473d56ba"
  type = "standard.tiny"
  zone = "ch-gva-2"

  network_interface {
    network_id = exoscale_private_network.my_network.id
  }
}
```

Again for other attributes that are not in the example you can copy them with the following rules:
- `affinity_group_ids` must be renamed to `anti_affinity_group_ids`;
- `affinity_groups` is unsupported and must be replaced with `affinity_group_ids` (we recommend using `exoscale_anti_affinity_group` datasource for anti-affinity group name->id mapping);
- `display_name` must be renamed as `name`;
- `hostname` is deprecated (`name` is used instead) and should be removed;
- `ip4` is deprecated and should be removed;
- `key_pair` must be renamed to `ssh_key`;
- `keyboard` is deprecated and should be removed;
- `security_groups` is unsupported and must be replaced with `security_group_ids` (we recommend using `exoscale_security_group` datasource for security group name->id mapping);
- `size` must be replaced with `type` and format changed to `<family>.<size>` (check [docs](https://registry.terraform.io/providers/exoscale/exoscale/latest/docs/resources/compute_instance#type) for details)
- `tags` must be renamed to `labels`;
- `template` is unsupported and must be replaced with `template_id` (we recommend using `exoscale_template` datasource for template name->id mapping);

### Importing the `compute_instance` and `private_network` into the state

Using the [Exoscale cli](https://github.com/exoscale/cli/), we can find the ID of the `privnet` private network:

```bash
$ exo compute private-network show privnet
┼─────────────────┼──────────────────────────────────────┼
│ PRIVATE NETWORK │                                      │
┼─────────────────┼──────────────────────────────────────┼
│ ID              │ cbdb7256-30d2-8dc4-288c-d7f58bc9e308 │
│ Name            │ privnet                              │
│ Description     │ Private Network                      │
│ Zone            │ ch-gva-2                             │
│ Type            │ manual                               │
┼─────────────────┼──────────────────────────────────────┼
```

In our case, it's `cbdb7256-30d2-8dc4-288c-d7f58bc9e308`. Let's import this resource:

```bash
$ terraform import exoscale_private_network.my_network cbdb7256-30d2-8dc4-288c-d7f58bc9e308@ch-gva-2
exoscale_private_network.my_network: Importing from ID "cbdb7256-30d2-8dc4-288c-d7f58bc9e308@ch-gva-2"...
exoscale_private_network.my_network: Import prepared!
  Prepared exoscale_private_network for import
exoscale_private_network.my_network: Refreshing state... [id=cbdb7256-30d2-8dc4-288c-d7f58bc9e308]

Import successful!

The resources that were imported are shown above. These resources are now in
your Terraform state and will henceforth be managed by Terraform.
```

As you can see, and as expected, the import process imported `exoscale_private_network.my_network`.
We also import  `exoscale_compute_instance` the same way. To find instance ID, we must use CLI again:

```bash
$ exo c i show my-instance
┼──────────────────────┼──────────────────────────────────────┼
│   COMPUTE INSTANCE   │                                      │
┼──────────────────────┼──────────────────────────────────────┼
│ ID                   │ 0e28952c-1ae9-45fc-9f6d-0a9710175fd8 │
│ Name                 │ my-instance                          │
│ Creation Date        │ 2023-07-21 16:29:28 +0000 UTC        │
│ Instance Type        │ standard.tiny                        │
│ Template             │ Linux Ubuntu 22.04 LTS 64-bit        │
│ Zone                 │ ch-gva-2                             │
│ Anti-Affinity Groups │ n/a                                  │
│ Deploy Target        │ -                                    │
│ Security Groups      │ default                              │
│ Private Instance     │ No                                   │
│ Private Networks     │ privnet                              │
│ Elastic IPs          │ n/a                                  │
│ IP Address           │ 194.182.161.116                      │
│ IPv6 Address         │ -                                    │
│ SSH Key              │ -                                    │
│ Disk Size            │ 10 GiB                               │
│ State                │ running                              │
│ Labels               │ n/a                                  │
│ Reverse DNS          │                                      │
┼──────────────────────┼──────────────────────────────────────┼
```

Instance ID is `0e28952c-1ae9-45fc-9f6d-0a9710175fd8` and we can import it:

```bash
$ terraform import exoscale_compute_instance.my_instance 0e28952c-1ae9-45fc-9f6d-0a9710175fd8@ch-gva-2

```

In order to check this result and display details on newly imported rules, we have to run [`terraform plan`](https://www.terraform.io/docs/commands/plan.html):

```bash
$ terraform apply
exoscale_private_network.my_network: Refreshing state... [id=cbdb7256-30d2-8dc4-288c-d7f58bc9e308]
exoscale_compute_instance.my_instance: Refreshing state... [id=0e28952c-1ae9-45fc-9f6d-0a9710175fd8]

No changes. Your infrastructure matches the configuration.

Terraform has compared your real infrastructure against your configuration and found no differences, so no changes are needed.

Apply complete! Resources: 0 added, 0 changed, 0 destroyed.
```

Here we can see that our configuration is matching the state: no resources will be changed.
Our Terraform state matches the related configuration, so migration is completely done.
