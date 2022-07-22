---
page_title: "Exoscale: exoscale_instance_pool"
description: |-
  Manage Exoscale Instance Pools.
---

# exoscale\_instance\_pool

Manage Exoscale [Instance Pools](https://community.exoscale.com/documentation/compute/instance-pools/).

Corresponding data sources: [exoscale_instance_pool](../data-sources/instance_pool.md), [exoscale_instance_pool_list](../data-sources/instance_pool_list.md).


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

* `zone` - (Required) The Exoscale [Zone][zone] name.
* `name` - (Required) The instance pool name.
* `instance_type` - (Required) The managed compute instances type (`<family>.<size>`, e.g. `standard.medium`; use the [Exoscale CLI][cli] - `exo compute instance-type list` - for the list of available types).
* `size` - (Required) The number of managed instances.
* `template_id` - (Required) The [exoscale_compute_template](../data-sources/compute_template.md) (ID) to use when creating the managed instances.

* `description` - A free-form text describing the pool.
* `deploy_target_id` - A deploy target ID.
* `disk_size` - The managed instances disk size (GiB).
* `instance_prefix` - The string used to prefix managed instances name (default: `pool`).
* `ipv6` - Enable IPv6 on managed instances (boolean; default: `false`).
* `key_pair` - The [exoscale_ssh_key](./ssh_key.md) (name) to authorize in the managed instances.
* `labels` - A map of key/value labels.
* `user_data` - [cloud-init][cloud-init] configuration to apply to the managed instances (no need to base64-encode or gzip it as the provider will take care of it).

* `affinity_group_ids` - A list of [exoscale_anti_affinity_group](./anti_affinity_group.md) (IDs; may only be set at creation time).
* `elastic_ip_ids` - A list of [exoscale_elastic_ip](./elastic_ip.md) (IDs).
* `network_ids` - A list of [exoscale_private_network](./private_network.md) (IDs).
* `security_group_ids` - A list of [exoscale_security_group](./security_groups.md) (IDs).

* `service_offering` - (Deprecated) The managed instances type. Please use the `instance_type` argument instead.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` – The instance pool ID.
* `instances` - The list of managed instances. Structure is documented below.

* `virtual_machines` – (Deprecated) The list of managed instances (IDs). Please use the `instances.*.id` attribute instead.

### `instances` items

* `id` - The compute instance ID.
* `ipv6_address` - The instance (main network interface) IPv6 address.
* `name` - The instance name.
* `public_ip_address` - The instance (main network interface) IPv4 address.


## Import

An existing instance pool may be imported by `<ID>@<zone>`:

```console
$ terraform import \
  exoscale_instance_pool.my_instance_pool \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6@ch-gva-2
```
