---
page_title: "Exoscale: exoscale_instance_pool"
description: |-
  Manage Exoscale Instance Pools.
---

# exoscale\_instance\_pool

Manage Exoscale [Instance Pools](https://community.exoscale.com/documentation/compute/instance-pools/).


## Usage

```hcl
data "exoscale_compute_template" "my_template" {
  zone = "ch-gva-2"
  name = "Linux Ubuntu 22.04 LTS 64-bit"
}

resource "exoscale_instance_pool" "my_instance_pool" {
  zone = "ch-gva-2"
  name = "my-instance-pool"

  template_id   = data.exoscale_compute_template.my_template.id
  instance_type = "standard.medium"
  disk_size     = 10
  size          = 3
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Argument Reference

[zone]: https://www.exoscale.com/datacenters/
[cli]: https://github.com/exoscale/cli/
[cloud-init]: http://cloudinit.readthedocs.io/

* `zone` - (Required) The name of the [zone][zone] to create the instance pool into.
* `name` - (Required) The name of the instance pool.
* `instance_type` - (Required) The managed compute instances type (`<family>.<size>`, e.g. `standard.medium`; use the [Exoscale CLI][cli] - `exo compute instance-type list` - for the list of available types).
* `size` - (Required) The number of compute instance members the instance pool manages.
* `template_id` - (Required) The ID of the compute instance [template](../data-sources/compute_template.md) to use when creating compute instances.

* `description` - A free-form text describing the instance pool.
* `deploy_target_id` - A deploy target ID.
* `disk_size` - The managed compute instances disk size (GiB).
* `instance_prefix` - The string used to prefix managed compute instances name (default: `pool`).
* `ipv6` - Enable IPv6 on managed compute instances (boolean; default: `false`).
* `key_pair` - The name of the [SSH key](./ssh_key.md) to authorize in compute instances.
* `labels` - A map of key/value labels.
* `user_data` - A [cloud-init][cloud-init] configuration to apply when creating compute instances. No need to base64-encode or gzip it as the provider will take care of it.

* `affinity_group_ids` - A list of [anti-affinity group](./anti_affinity_group.md) IDs (may only be set at creation time).
* `elastic_ip_ids` - A list of [elastic IP](./elastic_ip.md) IDs.
* `network_ids` - A list of [private network](./private_network.md) IDs.
* `security_group_ids` - A list of [security group](./security_groups.md) IDs.

* `service_offering` - (Deprecated) The managed compute instances type. Please use the `instance_type` argument instead.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` – The ID of the instance pool.
* `instances` - The list of instance pool members.

* `virtual_machines` – (Deprecated) The list of instance pool members (compute instance IDs). Please use the `instances.*.id` attribute instead.

### `instances` items

* `id` - The ID of the compute instance.
* `ipv6_address` - The IPv6 address of the compute instance's main network interface.
* `name` - The name of the compute instance.
* `public_ip_address` - The IPv4 address of the compute instance's main network interface.


## Import

An existing instance pool may be imported by `<ID>@<zone>`:

```console
$ terraform import \
  exoscale_instance_pool.my_instance_pool \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6@ch-gva-2
```
