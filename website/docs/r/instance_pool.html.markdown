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
resource "exoscale_ssh_keypair" "key" {
  name = "terraform-mywebapp-keypair"
}

variable "zone" {
  default = "de-fra-1"
}

resource "exoscale_security_group" "web" {
  name = "web"
  description = "Security Group for webapp production"
}

resource "exoscale_network" "web_privnet" {
  zone = var.zone
  name = "web-privnet"
}

data "exoscale_compute_template" "mywebapp" {
  zone = var.zone
  name = "mywebapp"
  filter = "mine"
}

resource "exoscale_instance_pool" "webapp" {
  zone = var.zone
  name = "webapp"
  template_id = data.exoscale_compute_template.mywebbapp.id
  size = 3
  service_offering = "Medium"
  disk_size = 50
  description = "This is the production environment for my webapp"
  user_data = "#cloud-config\npackage_upgrade: true\n"
  key_pair = exoscale_ssh_keypair.key.name

  security_group_ids = [${exoscale_security_group.web.id}]
  network_ids = [${exoscale_network.web_privnet.id}]

  timeouts {
    delete = "10m"
  }
}
```

## Argument Reference

* `zone` - (Required) The name of the [zone][zone] to deploy the Instance Pool into.
* `name` - (Required) The name of the Instance Pool.
* `template_id` - (Required) (Required) The ID of the instance [template][template] to use when creating Compute instances. Usage of the [`compute_template`][compute_template] data source is recommended.
* `size` - (Required) The number of Compute instance members the Instance Pool manages.
* `service_offering` - (Required) The managed Compute instances [size][size], e.g. `Tiny`, `Small`, `Medium`, `Large` etc.
* `disk_size` - The managed Compute instances disk size.
* `description` - The description of the Instance Pool.
* `user_data` - A [cloud-init][cloudinit] configuration to apply when creating Compute instances. Whenever possible don't base64-encode neither gzip it yourself, as this will be automatically taken care of on your behalf by the provider.
* `key_pair` - The name of the [SSH key pair][sshkeypair] to install when creating Compute instances.
* `security_group_ids` - A list of [Security Group][sg] IDs.
* `network_ids` - A list of [Private Network][net] IDs.

[template]: https://www.exoscale.com/templates/
[zone]: https://www.exoscale.com/datacenters/
[size]: https://www.exoscale.com/pricing/#/compute/
[sshkeypair]: https://community.exoscale.com/documentation/compute/ssh-keypairs/
[cloudinit]: http://cloudinit.readthedocs.io/en/latest/
[compute_template]: ../d/compute_template.html
[net]: https://community.exoscale.com/documentation/compute/private-networks/

## Import

An existing Instance Pool can be imported as a resource by name or ID. Importing an Instance Pool imports the `exoscale_instance_pool` resource.

```console
# By name
$ terraform import exoscale_instance_pool.pool mypool

# By ID
$ terraform import exoscale_instance_pool.pool eb556678-ec59-4be6-8c54-0406ae0f6da6
```
