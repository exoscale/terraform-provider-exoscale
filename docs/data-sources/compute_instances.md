---
page_title: "Exoscale: exoscale_compute_instances"
description: |-
  Shows a list of Compute instances.
---

# exoscale\_compute\_instance

Lists available [Exoscale Compute instances][compute-doc].


## Example Usage

```hcl
data "exoscale_compute_instances" "example" {
  zone = "ch-gva-2"
}
```

## Arguments Reference

* `zone` - (Required) The [zone][zone] of the Compute instance.
* `instances` - The list of instances.

The `instances` items contains:

* `created_at` - The creation date of the Compute instance.
* `id` - The ID of the compute instance.
* `ipv6_address` - The IPv6 address of the compute instance's main network interface.
* `labels` - A map of key/value labels.
* `name` - The name of the compute instance.
* `private_network_ids` - A list of [Private Network][r-private_network] IDs attached to the Compute instance.
* `public_ip_address` - The IPv4 address of the compute instance's main network interface.
* `ssh_key` - The name of the [SSH key pair][sshkeypair] installed to the Compute instance's user account during creation.
* `security_group_ids` - A list of [Security Group][r-security_group] IDs attached to the Compute instance.
* `state` - The state of the Compute instance.
* `template_id` - The ID of the instance [template][template] used when creating the Compute instance.
* `type` - The Compute instance [type][type].


[compute-doc]: https://community.exoscale.com/documentation/compute/
[r-private_network]: ../resources/private_network
[r-security_group]: ../resources/security_group
[sshkeypair-doc]: https://community.exoscale.com/documentation/compute/ssh-keypairs/
[template]: https://www.exoscale.com/templates/
[type]: https://www.exoscale.com/pricing/#/compute/
[zone]: https://www.exoscale.com/datacenters/
