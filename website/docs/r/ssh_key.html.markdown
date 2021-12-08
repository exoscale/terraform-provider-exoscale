---
layout: "exoscale"
page_title: "Exoscale: exoscale_ssh_key"
sidebar_current: "docs-exoscale-ssh-key"
description: |-
  Provides an Exoscale SSH Key.
---

# exoscale\_ssh\_key

Provides an Exoscale [SSH Key][ssh-keys-doc] resource. This can be used to create and delete SSH Keys.


## Example Usage

```hcl
resource "exoscale_ssh_key" "example" {
  name       = "admin"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDGRY..."
}
```


## Arguments Reference

* `name` - (Required) The name of the SSH Key.
* `public_key` - (Required) A SSH public key that will be copied into the Compute instances at **first** boot.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `fingerprint` - The unique identifier of the SSH Key.


## Import

An existing SSH Key can be imported as a resource by name:

```console
$ terraform import exoscale_ssh_key.my-key my-key
```


[ssh-keys-doc]: https://community.exoscale.com/documentation/compute/ssh-keys/
