---
page_title: "Exoscale: exoscale_compute_instance"
description: |-
  Fetch Exoscale Compute Instances data.
---

# exoscale\_compute\_instance

Fetch Exoscale [Compute Instances](https://community.exoscale.com/documentation/compute/) data.

Corresponding resource: [exoscale_compute_instance](../resources/compute_instance.md).


## Usage

```hcl
data "exoscale_compute_instance" "my_instance" {
  zone = "ch-gva-2"
  name = "my-instance"
}

output "my_instance_id" {
  value = data.exoscale_compute_instance.my_instance.id
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/

* `zone` - (Required) The Exoscale [Zone][zone] name.

* `id` - The compute instance ID to match (conflicts with `name`).
* `name` - The instance name to match (conflicts with `id`).


## Attributes Reference

[cloud-init]: http://cloudinit.readthedocs.io/en/latest/

In addition to the arguments listed above, the following attributes are exported:

* `created_at` - The compute instance creation date.
* `deploy_target_id` - A deploy target ID.
* `disk_size` - The instance disk size (GiB).
* `ipv6_address` - The instance (main network interface) IPv6 address.
* `ipv6` - Whether IPv6 is enabled on the instance.
* `labels` - A map of key/value labels.
* `manager_id` - The instance manager ID, if any.
* `manager_type` - The instance manager type (instance pool, SKS node pool, etc.), if any.
* `public_ip_address` - The instance (main network interface) IPv4 address.
* `ssh_key` - The [exoscale_ssh_key](../resources/ssh_key.md) (name) authorized on the instance.
* `state` - The instance state.
* `template_id` - The instance [exoscale_compute_template](./compute_template.md) ID.
* `type` - The instance type.
* `user_data` - The instance [cloud-init][cloud-init] configuration.

* `affinity_group_ids` - The list of attached [exoscale_anti_affinity_group](../resources/anti_affinity_group.md) (IDs).
* `elastic_ip_ids` - The list of attached [exoscale_elastic_ip](../resources/elastic_ip.md) (IDs).
* `network_ids` - The list of attached [exoscale_private_network](../resources/private_network.md) (IDs).
* `security_group_ids` - The list of attached [exoscale_security_group](../resources/security_group.md) (IDs).
