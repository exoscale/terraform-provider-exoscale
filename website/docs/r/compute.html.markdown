---
layout: "exoscale"
page_title: "Exoscale: exoscale_compute"
sidebar_current: "docs-exoscale-compute"
description: |-
  Provides an Exoscale Compute instance resource.
---

# exoscale\_compute

Provides an Exoscale [Compute instance][compute] resource. This can be used to create, modify, and delete Compute instances.

[compute]: https://community.exoscale.com/documentation/compute/

## Example Usage

```hcl
resource "exoscale_compute" "mymachine" {
  zone         = "ch-gva-2"
  display_name = "mymachine"
  template     = "Linux Debian 9 64-bit"
  size         = "Medium"
  disk_size    = 10
  key_pair     = "me@mymachine"
  state        = "Running"

  affinity_groups = []
  security_groups = ["default"]

  ip6 = false

  user_data = <<EOF
#cloud-config
manage_etc_hosts: localhost
EOF

  tags = {
    production = "true"
  }

  timeouts {
    create = "60m"
    delete = "2h"
  }
}
```

## Argument Reference

* `zone` - (Required) The name of the [zone][zone] to deploy the Compute instance into.
* `display_name` - (Required) The displayed name of the Compute instance. Note: This value is also used to set the OS' *hostname* during creation, so the value can only contain alphanumeric and hyphen ("-") characters; it can be changed to any character during a later update.
* `template` - (Required) The name or ID of the Compute instance [template][template]. If a name is provided, only *featured* templates are available.
* `size` - (Required) The Compute instance [size][size], e.g. `Tiny`, `Small`, `Medium`, `Large` etc.
* `disk_size` - (Required) The Compute instance root disk size in GiB (at least `10`).
* `key_pair` - (Required) The name of the [SSH key pair][sshkeypair] to be installed.
* `user_data` - A [cloud-init][cloudinit] configuration. Whenever possible don't base64-encode neither gzip it yourself, as this will be automatically taken care of on your behalf by the provider.
* `keyboard` - The keyboard layout configuration (at creation time only). Supported values are: `de`, `de-ch`, `es`, `fi`, `fr`, `fr-be`, `fr-ch`, `is`, `it`, `jp`, `nl-be`, `no`, `pt`, `uk`, `us`.
* `state` - The state of the Compute instance, e.g. `Running` or `Stopped`
* `affinity_groups` - A list of [Anti-Affinity Group][aag] names (at creation time only; conflicts with `affinity_group_ids`).
* `affinity_group_ids` - A list of [Anti-Affinity Group][aag] IDs (at creation time only; conflicts with `affinity_groups`).
* `security_groups` - A list of [Security Group][sg] names (conflicts with `security_group_ids`).
* `security_group_ids` - A list of [Security Group][sg] IDs (conflicts with `security_groups`).
* `ip4` - Boolean controlling if IPv4 is enabled (only supported value is `true`).
* `ip6` - Boolean controlling if IPv6 is enabled.
* `tags` - A dictionary of tags (key/value).

[template]: https://www.exoscale.com/templates/
[zone]: https://www.exoscale.com/datacenters/
[size]: https://www.exoscale.com/pricing/#/compute/
[sshkeypair]: https://community.exoscale.com/documentation/compute/ssh-keypairs/
[cloudinit]: http://cloudinit.readthedocs.io/en/latest/
[aag]: affinity.html
[sg]: security_group.html

## Attributes Reference

The following attributes are exported:

* `name` - The name of the Compute instance (*hostname*).
* `username` - The user to use to connect to the Compute instance with SSH.
* `password` - The initial Compute instance password and/or encrypted password.
* `ip_address` - The IP address of the Compute instance main network interface.
* `ip6_address` - The IPv6 address of the Compute instance main network interface.

## Import

An existing Compute instance can be imported as a resource by name or ID. Importing a Compute instance imports the `exoscale_compute` resource as well as related [`exoscale_secondary_ipaddress`][secip] and [`exoscale_nic`][nic] resources.

[secip]: secondary_ipaddress.html
[nic]: nic.html

```console
# By name
$ terraform import exoscale_compute.vm1 vm1

# By ID
$ terraform import exoscale_compute.vm1 eb556678-ec59-4be6-8c54-0406ae0f6da6
```
