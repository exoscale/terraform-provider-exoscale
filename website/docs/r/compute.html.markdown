---
layout: "exoscale"
page_title: "Exoscale: exoscale_compute"
sidebar_current: "docs-exoscale-compute"
description: |-
  Manages a compute resource.
---

# exoscale_compute

Exoscale computing service allows you to create a performant
cloud virtual machine in less than 35 seconds.

## Example Usage

```hcl
resource "exoscale_compute" "mymachine" {
  display_name = "mymachine"
  template = "Linux Debian 9 64-bit"
  size = "Medium"
  disk_size = 10
  key_pair = "me@mymachine"
  state = "Running"

  affinity_groups = []
  security_groups = ["default"]

  ip6 = false

  tags {
    production = "true"
  }

  timeouts {
    create = "60m"
    delete = "2h"
  }
}
```

## Argument Reference

- `display_name` - (Required) initial `hostname`

- `template` - (Required) name from [the template](https://www.exoscale.com/templates/)

- `size` - (Required) size of [the instances](https://www.exoscale.com/pricing/#/compute/),
              e.g. Tiny, Small, Medium, Large, etc.

- `disk_size` - (Required) size of the root disk in GiB (at least 10)

- `zone` - (Required) name of [the data-center](https://www.exoscale.com/datacenters/)

- `user_data` - [cloud-init](http://cloudinit.readthedocs.io/en/latest/) configuration

- `key_pair` - (Required) name of the SSH key pair to be installed

- `keyboard` - keyboard configuration (at creation time only)

- `state` - state of the virtual machine. E.g. `Running` or `Stopped`

- `affinity_groups` - list of [Affinity Groups](affinity_group.html)

- `security_groups` - list of [Security Groups](security_group.html)

- `ip4` - activate IPv4 (only `true`)

- `ip6` - activate IPv6 (`false` by default)

- `tags` - dictionary of tags (key / value)

## Attributes Reference

- `name` - name of the machine (`hostname`)

- `username` - User to connect when using SSH

- `password` - Initial password and/or encrypted password

- `ip_address` - IP Address of the main network interface

- `ip6_address` - IPv6 Address of the main network interface

## Import

Importing Compute resource imports the compute has well as the
[`exoscale_secondary_ipaddress`](secondary_ipaddress.html) and
[`exoscale_nic`](nic.html).

```shell
# by name
$ terraform import exoscale_compute.VM-1 default

# by id
$ terraform import exoscale_compute.VM-1 eb556678-ec59-4be6-8c54-0406ae0f6da6
```
