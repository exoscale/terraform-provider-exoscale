---
layout: "exoscale"
page_title: "Exoscale: exoscale_compute"
sidebar_current: "docs-exoscale-compute"
description: |-
  Provides an Exoscale Compute instance resource.
---

# exoscale\_compute

Provides an Exoscale [Compute instance][compute-doc] resource. This can be used to create, modify, and delete Compute instances.


## Example Usage

```hcl
data "exoscale_compute_template" "ubuntu" {
  zone = "ch-gva-2"
  name = "Linux Ubuntu 18.04 LTS 64-bit"
}

resource "exoscale_compute" "mymachine" {
  zone         = "ch-gva-2"
  display_name = "mymachine"
  template_id  = data.exoscale_compute_template.ubuntu.id
  size         = "Medium"
  disk_size    = 10
  key_pair     = "me@mymachine"
  state        = "Running"

  reverse_dns = "mymachine.com."

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

## Arguments Reference

* `zone` - (Required) The name of the [zone][zone] to deploy the Compute instance into.
* `template` - (Required) The name of the Compute instance [template][template]. Only *featured* templates are available, if you want to reference *custom templates* use the `template_id` attribute instead.
* `template_id` - (Required) The ID of the Compute instance [template][template]. Usage of the [`compute_template`][d-compute_template] data source is recommended.
* `size` - (Required) The Compute instance [size][size], e.g. `Tiny`, `Small`, `Medium`, `Large` etc.
* `disk_size` - (Required) The Compute instance root disk size in GiB (at least `10`).
* `display_name` - The displayed name of the Compute instance. Note: if the `hostname` attribute is not set, this attribute is also used to set the OS' *hostname* during creation, so the value must contain only alphanumeric and hyphen ("-") characters; it can be changed to any character during a later update. If neither `display_name` or `hostname` attributes are set, a random value will be generated automatically server-side.
* `hostname` - The Compute instance hostname, must contain only alphanumeric and hyphen ("-") characters. If neither `display_name` or `hostname` attributes are set, a random value will be generated automatically server-side. Note: updating this attribute's value requires to reboot the instance.
* `key_pair` - The name of the [SSH key pair][sshkeypair-doc] to be installed.
* `reverse_dns` - The reverse DNS record of the Compute instance (must end with a `.`, e.g: `my-server.example.net.`).
* `user_data` - A [cloud-init][cloudinit] configuration. Whenever possible don't base64-encode neither gzip it yourself, as this will be automatically taken care of on your behalf by the provider.
* `keyboard` - The keyboard layout configuration (at creation time only). Supported values are: `de`, `de-ch`, `es`, `fi`, `fr`, `fr-be`, `fr-ch`, `is`, `it`, `jp`, `nl-be`, `no`, `pt`, `uk`, `us`.
* `state` - The state of the Compute instance, e.g. `Running` or `Stopped`
* `affinity_groups` - A list of [Anti-Affinity Group][r-affinity] names (at creation time only; conflicts with `affinity_group_ids`).
* `affinity_group_ids` - A list of [Anti-Affinity Group][r-affinity] IDs (at creation time only; conflicts with `affinity_groups`).
* `security_groups` - A list of [Security Group][r-security_group] names (conflicts with `security_group_ids`).
* `security_group_ids` - A list of [Security Group][r-security_group] IDs (conflicts with `security_groups`).
* `ip4` - Boolean controlling if IPv4 is enabled (only supported value is `true`).
* `ip6` - Boolean controlling if IPv6 is enabled.
* `tags` - A dictionary of tags (key/value). To remove all tags, set `tags = {}`.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the Compute instance.
* `name` - **Deprecated** The Compute instance *hostname*. Use the `hostname` attribute instead.
* `username` - The user to use to connect to the Compute instance with SSH. If you've referenced a *custom template* in the resource, use the [`compute_template`][d-compute_template] data source `username` attribute instead.
* `password` - The initial Compute instance password and/or encrypted password.
* `ip_address` - The IP address of the Compute instance main network interface.
* `ip6_address` - The IPv6 address of the Compute instance main network interface.


## `remote-exec` provisioner usage

If you wish to log to a `exoscale_compute` resource using the [`remote-exec`][remote-exec] provisioner, make sure to explicity set the SSH `user` setting to connect to the instance to the actual template username returned by the [`exoscale_compute_template`][compute_template] data source:

```hcl
data "exoscale_compute_template" "ubuntu" {
  zone = "ch-gva-2"
  name = "Linux Ubuntu 18.04 LTS 64-bit"
}

resource "exoscale_compute" "mymachine" {
  zone         = "ch-gva-2"
  display_name = "mymachine"
  template_id  = data.exoscale_compute_template.ubuntu.id
  size         = "Medium"
  disk_size    = 10
  key_pair     = "me@mymachine"
  state        = "Running"

  provisioner "remote-exec" {
    connection {
      type = "ssh"
      host = self.ip_address
      user = data.exoscale_compute_template.ubuntu.username
    }
  }
}
```


## Import

An existing Compute instance can be imported as a resource by name or ID:


```console
# By name
$ terraform import exoscale_compute.vm1 vm1

# By ID
$ terraform import exoscale_compute.vm1 eb556678-ec59-4be6-8c54-0406ae0f6da6
```

~> **NOTE:** Importing a Compute instance resource also imports related [`exoscale_secondary_ipaddress`][r-secondary_ipaddress] and [`exoscale_nic`][r-nic] resources.


[cloudinit]: http://cloudinit.readthedocs.io/en/latest/
[compute-doc]: https://community.exoscale.com/documentation/compute/
[d-compute_template]: ../d/compute_template.html
[r-affinity]: affinity.html
[r-nic]: nic.html
[r-secondary_ipaddress]: secondary_ipaddress.html
[r-security_group]: security_group.html
[remote-exec]: https://www.terraform.io/docs/provisioners/remote-exec.html
[size]: https://www.exoscale.com/pricing/#/compute/
[sshkeypair-doc]: https://community.exoscale.com/documentation/compute/ssh-keypairs/
[template]: https://www.exoscale.com/templates/
[zone]: https://www.exoscale.com/datacenters/
