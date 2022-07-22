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

* `zone` - (Required) The name of the [zone][zone] to create the compute instance into.
* `name` - (Required) The name of the compute instance.
* `disk_size` - (Required) The compute instance disk size (GiB; at least `10`). **WARNING**: updating this attribute stops/restarts the compute instance.
* `template_id` - (Required) The ID of the compute instance [template](../data-sources/compute_template.md) to use when creating the compute instance.
* `type` - (Required) The compute instance type (`<family>.<size>`, e.g. `standard.medium`; use the [Exoscale CLI][cli] - `exo compute instance-type list` - for the list of available types). **WARNING**: updating this attribute stops/restarts the compute instance.

* `deploy_target_id` - A deploy target ID.
* `ipv6` - Enable IPv6 on the compute instance (boolean; default: `false`).
* `labels` - A map of key/value labels.
* `ssh_key` - The name of the [SSH key](./ssh_key.md) to authorize in the compute instance (may only be set at creation time).
* `state` - The state of the compute instance (`running` or `stopped`; default: `running`).
* `user_data` - A [cloud-init][cloud-init] configuration. No need to base64-encode or gzip it as the provider will take care of it.

* `anti_affinity_group_ids` - A list of [anti-affinity group](./anti_affinity_group.md) IDs to assign the compute instance (may only be set at creation time).
* `elastic_ip_ids` - A list of [elastic IP](./elastic_ip.md) IDs to attach to the compute instance.
* `security_group_ids` - A list of [security group](./security_group.md) IDs to attach to the compute instance.

* `network_interface` - (Block) Private network interfaces (may be specified multiple times). Structure is documented below.

### `network_interface` block

* `network_id` - (Required) The [private network](./private_network.md) ID to attach to the compute instance.

* `ip_address` - The IPv4 address to request as static DHCP lease if the network interface is attached to a *managed* private network.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the compute instance.
* `created_at` - The creation date of the compute instance.
* `ipv6_address` - The IPv6 address of the compute instance main network interface.
* `public_ip_address` - The IPv4 address of the compute instance's main network interface.

* `private_network_ids` - (Deprecated) A list of private network IDs attached to the compute instance. Please use the `network_interface.*.network_id` argument instead.


## Import

An existing compute instance may be imported by `<ID>@<zone>`:

```console
$ terraform import \
  exoscale_compute_instance.my_instance \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6@ch-gva-2
```
