---
page_title: "Exoscale: exoscale_instance_pool"
description: |-
  Provides information about a Instance Pool.
---

# exoscale\_instance\_pool

Provides information on an [Exoscale Instance Pool][pool-doc].


## Example Usage

```hcl
data "exoscale_instance_pool" "example" {
  zone = "ch-gva-2"
  name = "my-instance-pool"
}

output "instance_pool_state" {
  value = data.exoscale_instance_pool.example.state
}
```

## Arguments Reference

* `zone` - (Required) The [zone][zone] of the Compute instance pool.

One of the following arguments is required:

* `id` - The ID of the Compute instance pool.
* `name` - The name of the Compute instance pool.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `affinity_group_ids` - A list of [Anti-Affinity Group][r-affinity] IDs.
* `deploy_target_id` - A Deploy Target ID.
* `description` - The description of the Instance Pool.
* `disk_size` - The managed Compute instances disk size.
* `elastic_ip_ids` - A list of [Elastic IP][eip-doc] IDs.
* `instance_prefix` - The string to add as prefix to managed Compute instances name.
* `instance_type` - The managed Compute instances [type][type].
* `ipv6` - Whether IPv6 is enabled on managed Compute instances.
* `key_pair` - The name of the [SSH key pair][sshkeypair].
* `labels` - A map of key/value labels.
* `network_ids` - A list of [Private Network][privnet-doc] IDs.
* `security_group_ids` - A list of [Security Group][r-security_group] IDs.
* `size` - The number of Compute instance members the Instance Pool manages.
* `state` - Instance Pool state.
* `template_id` - The ID of the instance [template][template].
* `user_data` - A [cloud-init][cloudinit] configuration.
* `instances` - The list of Instance Pool members.

The `instances` items contains:

* `id` - The ID of the compute instance.
* `ipv6_address` - The IPv6 address of the compute instance's main network interface.
* `name` - The name of the compute instance.
* `public_ip_address` - The IPv4 address of the compute instance's main network interface.

[pool-doc]: https://community.exoscale.com/documentation/compute/instance-pools/
[zone]: https://www.exoscale.com/datacenters/
[r-affinity]: ../resources/affinity
[eip-doc]: https://community.exoscale.com/documentation/compute/eip/
[type]: https://www.exoscale.com/pricing/#/compute/
[sshkeypair]: https://community.exoscale.com/documentation/compute/ssh-keypairs/
[privnet-doc]: https://community.exoscale.com/documentation/compute/private-networks/
[r-security_group]: ../resources/security_group
[template]: https://www.exoscale.com/templates/
[cloudinit]: http://cloudinit.readthedocs.io/en/latest/
