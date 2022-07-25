---
page_title: "Exoscale: exoscale_compute"
subcategory: "Deprecated"
description: |-
  Manage Exoscale Compute Instances.
---

# exoscale\_compute

!> **WARNING:** This resource is **DEPRECATED** and will be removed in the next major version. Please use [exoscale_compute_instance](./compute_instance.md) instead.


## Arguments Reference

* `zone` - (Required) The Exoscale Zone name.
* `disk_size` - (Required) The instance disk size (GiB; at least `10`).
* `size` - (Required) The instance size (`Tiny`, `Small`, `Medium`, `Large`, etc.)
* `template` - (Required) The compute instance template (name). Only *featured* templates are available, if you want to reference *custom templates* use the `template_id` attribute instead.
* `template_id` - (Required) The compute instance template (ID). Usage of the `exoscale_compute_template` data source is recommended.

* `display_name` - The displayed instance name. Note: if the `hostname` attribute is not set, this attribute is also used to set the OS' *hostname* during creation, so the value must contain only alphanumeric and hyphen ("-") characters; it can be changed to any character during a later update. If neither `display_name` or `hostname` attributes are set, a random value will be generated automatically.
* `hostname` - The instance hostname, must contain only alphanumeric and hyphen (`-`) characters. If neither `display_name` or `hostname` attributes are set, a random value will be generated automatically. Note: updating this attribute's value requires to reboot the instance.
* `ip4` - Enable IPv4 on the instance (only supported value is `true`).
* `ip6` - Enable IPv6 on the instance (boolean; default: `false`).
* `key_pair` - The SSH keypair (name) to authorize in the instance.
* `keyboard` - The keyboard layout configuration (`de`, `de-ch`, `es`, `fi`, `fr`, `fr-be`, `fr-ch`, `is`, `it`, `jp`, `nl-be`, `no`, `pt`, `uk`, `us`; at creation time only).
* `reverse_dns` - The instance reverse DNS record (must end with a `.`; e.g: `my-instance.example.net.`).
* `state` - The instance state (`Running` or `Stopped`; default: `Running`)
* `tags` - A map of tags (key/value). To remove all tags, set `tags = {}`.
* `user_data` - cloud-init configuration (no need to base64-encode or gzip it as the provider will take care of it).

* `affinity_groups` - A list of anti-affinity groups (names; at creation time only; conflicts with `affinity_group_ids`).
* `affinity_group_ids` - A list of anti-affinity groups (IDs; at creation time only; conflicts with `affinity_groups`).
* `security_groups` - A list of security groups (names; conflicts with `security_group_ids`).
* `security_group_ids` - A list of security groups (IDs; conflicts with `security_groups`).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The instance ID.
* `ip_address` - The instance (main network interface) IPv4 address.
* `ip6_address` - The instance (main network interface) IPv6 address (if enabled).
* `password` - The instance initial password and/or encrypted password.
* `username` - The user to use to connect to the instance. If you've referenced a *custom template* in the resource, use the `exoscale_compute_template` data source `username` attribute instead.

* `name` - (Deprecated) The instance hostname. Please use the `hostname` argument instead.
