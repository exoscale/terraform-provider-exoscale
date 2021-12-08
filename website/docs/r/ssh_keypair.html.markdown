---
layout: "exoscale"
page_title: "Exoscale: exoscale_ssh_keypair"
sidebar_current: "docs-exoscale-ssh-keypair"
description: |-
  Provides an Exoscale SSH Keypair.
---

# exoscale\_ssh\_keypair

Provides an Exoscale [SSH Keypair][ssh-keypairs-doc] resource. This can be used to create and delete SSH Keypairs.

!> **WARNING:** This resource is deprecated and will be removed in the next major version.


## Example Usage

```hcl
resource "exoscale_ssh_keypair" "admin" {
  name       = "admin"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDGRY..."
}
```


## Arguments Reference

* `name` - (Required) The name of the SSH Keypair.
* `public_key` - A SSH public key that will be copied into the instances at **first** boot. If not provided, a SSH keypair is generated and the is saved locally (see the `private_key` attribute).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `fingerprint` - The unique identifier of the SSH Keypair.
* `private_key` - The SSH private key generated if no public key was provided.


## Import

An existing SSH Keypair can be imported as a resource by name:

```console
$ terraform import exoscale_ssh_keypair.mykey my-key
```


[ssh-keypairs-doc]: https://community.exoscale.com/documentation/compute/ssh-keypairs/
