---
page_title: "Exoscale: exoscale_ssh_key"
description: |-
  Manage Exoscale SSH Keys.
---

# exoscale\_ssh\_key

Manage Exoscale [SSH Keys](https://community.exoscale.com/documentation/compute/ssh-keypairs/).


## Usage

```hcl
resource "exoscale_ssh_key" "my_ssh_key" {
  name       = "my-ssh-key"
  public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDGRY..."
}
```

Should you want to _create_ an SSH keypair (including private key) with Terraform, please use the
[tls_private_key](https://registry.terraform.io/providers/hashicorp/tls/latest/docs/resources/private_key)
resource:

```hcl
resource "tls_private_key" "my_ssh_key" {}

resource "exoscale_ssh_key" "my_ssh_key" {
  name       = "my-ssh-key"
  public_key = tls_private_key.my_ssh_key.public_key_openssh
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/) directory for complete configuration examples.


## Arguments Reference

* `name` - (Required) The name of the SSH key.
* `public_key` - (Required) A SSH public key that will be authorized in compute instances.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `fingerprint` - The unique identifier of the SSH key.


## Import

An existing SSH key may be imported as a resource by `<name>`:

```console
$ terraform import \
  exoscale_ssh_key.my_ssh_key \
  my-ssh-key
```
