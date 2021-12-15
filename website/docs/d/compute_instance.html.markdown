---
layout: "exoscale"
page_title: "Exoscale: exoscale_compute_instance"
sidebar_current: "docs-exoscale-compute-instance"
description: |-
  Provides information about a Compute instance.
---

# exoscale\_compute\_instance

Provides information on an [Exoscale Compute instance][compute-doc].


## Example Usage

```hcl
data "exoscale_compute_instance" "example" {
  zone = "ch-gva-2"
  name = "my-instance"
}
```

## Arguments Reference

* `zone` - (Required) The [zone][zone] of the Compute instance.
* `id` - The ID of the Compute instance.
* `name` - The name of the Compute instance.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `anti_affinity_group_ids` - A list of [Anti-Affinity Group][r-anti_affinity_group] IDs.
* `created_at` - The creation date of the Compute instance.
* `deploy_target_id` - A Deploy Target ID.
* `disk_size` - The Compute instance disk size in GiB.
* `elastic_ip_ids` - A list of [Elastic IP][r-elastic_ip] IDs attached to the Compute instance.
* `ipv6_address` - The IPv6 address of the Compute instance main network interface.
* `ipv6` - Whether IPv6 is enabled on the Compute instance.
* `labels` - A map of key/value labels.
* `manager_id` - The ID of the Compute instance manager, if any.
* `manager_type` - The type of Compute instance manager, if any.
* `private_network_ids` - A list of [Private Network][r-private_network] IDs attached to the Compute instance.
* `public_ip_address` - The IPv4 address of the Compute instance's main network interface.
* `security_group_ids` - A list of [Security Group][r-security_group] IDs attached to the Compute instance.
* `ssh_key` - The name of the [SSH key pair][sshkeypair] installed to the Compute instance's user account during creation.
* `state` - The state of the Compute instance.
* `template_id` - The ID of the instance [template][template] used when creating the Compute instance.
* `type` - The Compute instance [type][type].
* `user_data` - A [cloud-init][cloudinit] configuration.


[cloudinit]: http://cloudinit.readthedocs.io/en/latest/
[compute-doc]: https://community.exoscale.com/documentation/compute/
[r-anti_affinity_group]: anti_affinity_group.html
[r-elastic_ip]: ../r/elastic_ip.html
[r-private_network]: ../r/private_network.html
[r-security_group]: security_group.html
[sshkeypair-doc]: https://community.exoscale.com/documentation/compute/ssh-keypairs/
[template]: https://www.exoscale.com/templates/
[type]: https://www.exoscale.com/pricing/#/compute/
[zone]: https://www.exoscale.com/datacenters/
