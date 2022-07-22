---
page_title: "Exoscale: exoscale_ssh_keypair"
subcategory: "Deprecated"
description: |-
  Manage Exoscale SSH Keypairs.
---

# exoscale\_ssh\_keypair

!> **WARNING:** This resource is **DEPRECATED** and will be removed in the next major version. Please use [exoscale_ssh_key](./ssh_key.md) instead (or refer to the ad-hoc [migration guide](../guides/migration-of-ssh-keypair.md)).

!> **WARNING:** This resource stores sensitive information in your Terraform state. Please be sure to correctly understand implications and how to mitigate potential risks before using it.


## Arguments Reference

* `name` - (Required) The SSH keypair name.

* `public_key` - A SSH *public* key that will be authorized in compute instances. If not provided, an SSH keypair (including the *private* key) is generated and saved locally (see the `private_key` attribute).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `fingerprint` - The SSH keypair unique identifier.
* `private_key` - The SSH *private* key generated if no public key was provided.
