---
page_title: "Exoscale: exoscale_instance_pool"
description: |-
  Fetch Exoscale Instance Pools data.
---

# exoscale\_instance\_pool

Fetch Exoscale [Instance Pools](https://community.exoscale.com/documentation/compute/instance-pools/) data.

Corresponding resource: [exoscale_instance_pool](../resources/instance_pool.md).


## Usage

```hcl
data "exoscale_instance_pool" "my_instance_pool" {
  zone = "ch-gva-2"
  name = "my-instance-pool"
}

output "my_instance_pool_id" {
  value = data.exoscale_instance_pool.my_instance_pool.id
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/

* `zone` - (Required) The Exoscale [Zone][zone] name.

* `id` - The instance pool ID to match (conflicts with `name`).
* `name` - The pool name to match (conflicts with `id`).


## Attributes Reference

[cloud-init]: http://cloudinit.readthedocs.io/en/latest/

In addition to the arguments listed above, the following attributes are exported:

* `description` - The instance pool description.
* `deploy_target_id` - The deploy target ID.
* `disk_size` - The managed instances disk size.
* `instance_prefix` - The string used to prefix the managed instances name.
* `instance_type` - The managed instances type.
* `ipv6` - Whether IPv6 is enabled on managed instances.
* `key_pair` - The [exoscale_ssh_key](../resources/ssh_key.md) (name) authorized on the managed instances.
* `labels` - A map of key/value labels.
* `size` - The number managed instances.
* `state` - The pool state.
* `template_id` - The managed instances [exoscale_compute_template](./compute_template.md) ID.
* `user_data` - [cloud-init][cloud-init] configuration.

* `affinity_group_ids` - The list of attached [exoscale_anti_affinity_group](../resources/anti_affinity_group.md) (IDs).
* `elastic_ip_ids` - The list of attached [exoscale_elastic_ip](../resources/elastic_ip.md) (IDs).
* `network_ids` - The list of attached [exoscale_private_network](../resources/private_network.md) (IDs).
* `security_group_ids` - The list of attached [exoscale_security_group](../resources/security_group.md) (IDs).

* `instances` - The list of managed instances. Structure is documented below.

### `instances` items

* `id` - The compute instance ID.
* `name` - The instance name.
* `ipv6_address` - The instance (main network interface) IPv6 address.
* `public_ip_address` - The instance (main network interface) IPv4 address.
