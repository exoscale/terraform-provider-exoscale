---
layout: "exoscale"
page_title: "Exoscale: exoscale_instance_pool"
sidebar_current: "docs-exoscale-instance-pool"
description: |-
  Provides an Exoscale Instance Pool resource.
---

# exoscale\_instance\_pool

Provides an Exoscale Instance Pool resource. This can be used to create, modify, and delete Instance Pools.


## Example Usage

```hcl
variable "zone" {
  default = "de-fra-1"
}

resource "exoscale_ssh_keypair" "webapp" {
  name = "my-web-app"
}

resource "exoscale_security_group" "webapp" {
  name = "webapp"
  description = "my-web-app"
}

resource "exoscale_network" "webapp" {
  zone = var.zone
  name = "my-web-app"
}

resource "exoscale_ipaddress" "webapp" {
  zone = var.zone
}

data "exoscale_compute_template" "webapp" {
  zone = var.zone
  name = "my-web-app"
  filter = "mine"
}

resource "exoscale_instance_pool" "webapp" {
  zone = var.zone
  name = "my-web-app"
  size = 3
  template_id = data.exoscale_compute_template.webapp.id
  instance_type = "medium"
  disk_size = 50
  key_pair = exoscale_ssh_keypair.webapp.name
  instance_prefix = "my-web-app"
  security_group_ids = [exoscale_security_group.webapp.id]
  network_ids = [exoscale_network.webapp.id]
  elastic_ip_ids = [exoscale_ipaddress.webapp.id]
  user_data = "#cloud-config\npackage_upgrade: true\n"

  labels = {
    app = "webapp"
    env = "prod"
  }

  timeouts {
    delete = "10m"
  }
}
```


## Argument Reference

* `zone` - (Required) The name of the [zone][zone] to deploy the Instance Pool into.
* `name` - (Required) The name of the Instance Pool.
* `template_id` - (Required) The ID of the instance [template][template] to use when creating Compute instances. Usage of the [`compute_template`][d-compute_template] data source is recommended.
* `size` - (Required) The number of Compute instance members the Instance Pool manages.
* `instance_type` - (Required) The managed Compute instances [type][type] (format: `FAMILY.SIZE`, e.g. `standard.medium`, `memory.huge`).
* `service_offering` - **Deprecated** The managed Compute instances size. Replaced by `instance_type`.
* `disk_size` - The managed Compute instances disk size.
* `description` - The description of the Instance Pool.
* `user_data` - A [cloud-init][cloudinit] configuration to apply when creating Compute instances. Whenever possible don't base64-encode neither gzip it yourself, as this will be automatically taken care of on your behalf by the provider.
* `key_pair` - The name of the [SSH key pair][sshkeypair] to install when creating Compute instances.
* `ipv6` - Enable IPv6 on managed Compute instances (default: `false`).
* `instance_prefix` - The string to add as prefix to managed Compute instances name (default: `pool`).
* `affinity_group_ids` - A list of [Anti-Affinity Group][r-affinity] IDs (at creation time only).
* `security_group_ids` - A list of [Security Group][r-security_group] IDs (at creation time only).
* `network_ids` - A list of [Private Network][privnet-doc] IDs.
* `elastic_ip_ids` - A list of [Elastic IP][eip-doc] IDs.
* `deploy_target_id` - A Deploy Target ID.
* `labels` - A map of key/value labels.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` – The ID of the Instance Pool.
* `virtual_machines` – The list of Instance Pool members (Compute instance IDs).


## Import

An existing Instance Pool can be imported as a resource by `<ID>@<ZONE>`:

```console
$ terraform import exoscale_instance_pool.example eb556678-ec59-4be6-8c54-0406ae0f6da6@de-fra-1
```


[cloudinit]: http://cloudinit.readthedocs.io/en/latest/
[d-compute_template]: ../d/compute_template.html
[eip-doc]: https://community.exoscale.com/documentation/compute/eip/
[privnet-doc]: https://community.exoscale.com/documentation/compute/private-networks/
[r-affinity]: affinity.html
[r-security_group]: security_group.html
[sshkeypair]: https://community.exoscale.com/documentation/compute/ssh-keypairs/
[template]: https://www.exoscale.com/templates/
[type]: https://www.exoscale.com/pricing/#/compute/
[zone]: https://www.exoscale.com/datacenters/
