---
page_title: "Exoscale: exoscale_compute_instance"
description: |-
  Manage Exoscale Compute Instances.
---

# exoscale\_compute\_instance

Manage Exoscale [Compute Instances](https://community.exoscale.com/documentation/compute/).


## Usage

```hcl
data "exoscale_compute_template" "my_template" {
  zone = "ch-gva-2"
  name = "Linux Ubuntu 22.04 LTS 64-bit"
}

resource "exoscale_compute_instance" "my_instance" {
  zone = "ch-gva-2"
  name = "my-instance"

  template_id = data.exoscale_compute_template.my_template.id
  type        = "standard.medium"
  disk_size   = 10
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/
[cli]: https://github.com/exoscale/cli/
[cloud-init]: https://cloudinit.readthedocs.io/

* `zone` - (Required) The Exoscale [Zone][zone] name.
* `name` - (Required) The compute instance name.
* `disk_size` - (Required) The instance disk size (GiB; at least `10`). **WARNING**: updating this attribute stops/restarts the instance.
* `template_id` - (Required) The [exoscale_compute_template](../data-sources/compute_template.md) (ID) to use when creating the instance.
* `type` - (Required) The instance type (`<family>.<size>`, e.g. `standard.medium`; use the [Exoscale CLI][cli] - `exo compute instance-type list` - for the list of available types). **WARNING**: updating this attribute stops/restarts the instance.

* `deploy_target_id` - A deploy target ID.
* `ipv6` - Enable IPv6 on the instance (boolean; default: `false`).
* `labels` - A map of key/value labels.
* `ssh_key` - The [exoscale_ssh_key](./ssh_key.md) (name) to authorize in the instance (may only be set at creation time).
* `state` - The instance state (`running` or `stopped`; default: `running`).
* `user_data` - [cloud-init][cloud-init] configuration (no need to base64-encode or gzip it as the provider will take care of it).

* `anti_affinity_group_ids` - A list of [exoscale_anti_affinity_group](./anti_affinity_group.md) (IDs) to attach to the instance (may only be set at creation time).
* `elastic_ip_ids` - A list of [exoscale_elastic_ip](./elastic_ip.md) (IDs) to attach to the instance.
* `security_group_ids` - A list of [exoscale_security_group](./security_group.md) (IDs) to attach to the instance.

* `network_interface` - (Block) Private network interfaces (may be specified multiple times). Structure is documented below.

### `network_interface` block

* `network_id` - (Required) The [exoscale_private_network](./private_network.md) (ID) to attach to the instance.

* `ip_address` - The IPv4 address to request as static DHCP lease if the network interface is attached to a *managed* private network.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The compute instance ID.
* `created_at` - The instance creation date.
* `ipv6_address` - The instance (main network interface) IPv6 address.
* `public_ip_address` - The instance (main network interface) IPv4 address.

* `private_network_ids` - (Deprecated) A list of private networks (IDs) attached to the instance. Please use the `network_interface.*.network_id` argument instead.


## Import

An existing compute instance may be imported by `<ID>@<zone>`:

```console
$ terraform import \
  exoscale_compute_instance.my_instance \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6@ch-gva-2
```
