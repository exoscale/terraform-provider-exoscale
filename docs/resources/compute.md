---
page_title: "Exoscale: exoscale_compute"
subcategory: "Deprecated"
description: |-
  Manage Exoscale Compute Instances.
---

# exoscale\_compute

!> **WARNING:** This resource is **DEPRECATED** and will be removed in the next major version. Please use [exoscale_compute_instance](./compute_instance) instead.


## Arguments Reference

* `zone` - (Required) The name of the zone to create the compute instance into.
* `disk_size` - (Required) The compute instance root disk size (GiB; at least `10`).
* `size` - (Required) The compute instance size (`Tiny`, `Small`, `Medium`, `Large`, etc.)
* `template` - (Required) The name of the compute instance template. Only *featured* templates are available, if you want to reference *custom templates* use the `template_id` attribute instead.
* `template_id` - (Required) The ID of the compute instance template. Usage of the `compute_template` data source is recommended.

* `display_name` - The displayed name of the compute instance. Note: if the `hostname` attribute is not set, this attribute is also used to set the OS' *hostname* during creation, so the value must contain only alphanumeric and hyphen ("-") characters; it can be changed to any character during a later update. If neither `display_name` or `hostname` attributes are set, a random value will be generated automatically.
* `hostname` - The compute instance hostname, must contain only alphanumeric and hyphen (`-`) characters. If neither `display_name` or `hostname` attributes are set, a random value will be generated automatically. Note: updating this attribute's value requires to reboot the instance.
* `ip4` - Boolean controlling if IPv4 is enabled (only supported value is `true`).
* `ip6` - Boolean controlling if IPv6 is enabled (default: `false`).
* `key_pair` - The name of the SSH keypair to be installed.
* `keyboard` - The keyboard layout configuration (`de`, `de-ch`, `es`, `fi`, `fr`, `fr-be`, `fr-ch`, `is`, `it`, `jp`, `nl-be`, `no`, `pt`, `uk`, `us`; at creation time only).
* `reverse_dns` - The reverse DNS record of the compute instance (must end with a `.`; e.g: `my-server.example.net.`).
* `state` - The state of the compute instance (`Running` or `Stopped`; default: `Running`)
* `tags` - A dictionary of tags (key/value). To remove all tags, set `tags = {}`.
* `user_data` - A cloud-init configuration. Whenever possible don't base64-encode neither gzip it yourself, as this will be automatically taken care of by the provider.

* `affinity_groups` - A list of anti-affinity group names (at creation time only; conflicts with `affinity_group_ids`).
* `affinity_group_ids` - A list of anti-affinity group IDs (at creation time only; conflicts with `affinity_groups`).
* `security_groups` - A list of security group names (conflicts with `security_group_ids`).
* `security_group_ids` - A list of security group IDs (conflicts with `security_groups`).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the compute instance.
* `ip_address` - The IP address of the compute instance main network interface.
* `ip6_address` - The IPv6 address of the compute instance main network interface.
* `password` - The initial compute instance password and/or encrypted password.
* `username` - The user to use to connect to the compute instance with SSH. If you've referenced a *custom template* in the resource, use the `compute_template` data source `username` attribute instead.

* `name` - (Deprecated) The compute instance *hostname*. Please use the `hostname` argument instead.
