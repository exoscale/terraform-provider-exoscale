---
layout: "exoscale"
page_title: "Exoscale: exoscale_ssh_keypair"
sidebar_current: "docs-exoscale-ssh-keypair"
description: |-
  Manages an SSH key pair.
---

# exoscale_ssh_keypair

Declare an SSH key that will be used for any compute instances.

## Example Usage

```hcl
resource "exoscale_ssh_keypair" "keylabel" {
  name = "keyname"
  public_key = "keycontents"
}
```

## Argument Reference

- `name` - (Required) Defines the label in Exoscale to identify the key

- `public_key` - the SSH public key that will be copied into the instances at **first** boot. If not `public_key` is provided, a `public_key` is saved locally

## Attributes Reference

- `fingerprint` - the unique identifier of the SSH Key Pair

- `public_key` - if no public key was provided, this has been generated

- `private_key` - if no public key was provided and a key was generated

## Import

```shell
# by name
$ terraform import exoscale_ssh_keypair.mykeypair name
```
