---
layout: "exoscale"
page_title: "Exoscale: exoscale_instance_pool"
sidebar_current: "docs-exoscale-instance-pool"
description: |-
  Provides an Exoscale instance pool resource.
---

# exoscale\_instance\_pool

Provides an Exoscale `instance pool` resource. This can be used to create, modify, and delete instance pools.


## Example Usage

```hcl
resource "exoscale_ssh_keypair" "key" {
  name = "terraform-keypair"
}

variable "template" {
  default = "Linux Ubuntu 18.04 LTS 64-bit"
}

variable "zone" {
  default = "de-fra-1"
}

data "exoscale_compute_template" "instancepool" {
  zone = "${var.zone}"
  name = "${var.template}"
}

resource "exoscale_instance_pool" "pool" {
  zone = "${var.zone}"
  name = "terraform-instance-pool"
  template_id = "${data.exoscale_compute_template.instancepool.id}"
  size = 3
  service_offering = "Medium"
  disk_size = 50
  description = "my description"
  user_data = "#cloud-config\npackage_upgrade: true\n"
  key_pair = "${exoscale_ssh_keypair.key.name}"

  security_group_ids = ["4d388ced-209d-4be4-932c-f99d14d6e3b9"]
  network_ids = ["13ec3ed2-ec06-4061-9bd3-92bd0e3adebf"]

  timeouts {
    create = "10m"
  }
}
```

## Argument Reference

* `zone` - (Required) The name of the [zone][zone] to deploy the Compute instance into.
* `name` - (Required) The name of the instance pool.
* `template_id` - (Required) The ID of the Compute instance [template][template]. Usage of the [`compute_template`][compute_template] data source is recommended.
* `size` - (Required) The number of instances in the instance pool.
* `service_offering` - (Required) The Compute instance [size][size], e.g. `Tiny`, `Small`, `Medium`, `Large` etc.
* `disk_size` - The instances disk size from the instance pool.
* `description` - The description of the instance pool.
* `user_data` - A [cloud-init][cloudinit] configuration. Whenever possible don't base64-encode neither gzip it yourself, as this will be automatically taken care of on your behalf by the provider.
* `key_pair` - The name of the [SSH key pair][sshkeypair] to be installed on each instance.
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

An existing Compute instance can be imported as a resource by name or ID. Importing an instance pool imports the `exoscale_instance_pool` resource.

```console
# By name
$ terraform import exoscale_instance_pool.pool mypool

# By ID
$ terraform import exoscale_instance_pool.pool eb556678-ec59-4be6-8c54-0406ae0f6da6
```
