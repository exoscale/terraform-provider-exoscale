---
page_title: ssh_keypair migration Guide
description: |-
  Migrating from ssh_keypair to ssh_key
---

# Migrating from ssh_keypair to ssh_key

-> This migration guide applies to Exoscale Terraform Provider **version 0.31.0 or above**.

This page helps you migrate from an `exoscale_ssh_keypair` resource (deprecated) to the new
`exoscale_ssh_key`.

Unlike its predecessor, the `exoscale_ssh_key` resource doesn't support generating private keys
and only allows the registration of an existing (public) key in your Exoscale account.

Should you need to generate a key _pair_ (public and private key), we invite you to use the generic
[tls_private_key][tls_private_key] and the resource's `public_key_openssh` output along Exoscale
`exoscale_ssh_key`. Example given:

```hcl
resource "tls_private_key" "my_ssh_key" {
  algorithm = "ED25519"
}

resource "exoscale_ssh_key" "my_ssh_key" {
  name       = "my-ssh-key"
  public_key = tls_private_key.my_ssh_key.public_key_openssh
}
```

[tls_private_key]: https://registry.terraform.io/providers/hashicorp/tls/latest/docs/resources/private_key

**WARNING:** Should you generate an `RSA` key pair, make sure your SSH _client_ supports SHA2 -
`rsa-sha2-256` or `rsa-sha2-512` - when talking to SSH servers which might have disabled support
for SHA1 (example given [OpenSSH 8.8 and above](https://www.openssh.com/txt/release-8.8))!
