---
layout: "exoscale"
page_title: "Exoscale: exoscale_secondary_ipaddress"
sidebar_current: "docs-exoscale-secondary-ipaddress"
description: |-
  Manages an elastic IP address assignement to a compute resource
---

# exoscale_secondary_ipaddress

A Secondary IP Address expresses the attribution of an extra IP address to a
compute resource.

~> **NOTE** The network interfaces of the compute resource itself still have
to be configured accordingly.

### Secondary IP Address

```hcl
resource "exoscale_secondary_ipaddress" "ingress_ip" {
  compute_id = "${exoscale_compute.mymachine.id}"
  ip_address = "${exoscale_ipaddress.myip.ip_address}"
}
```

## Argument Reference

- `compute_id` - (Required) id of the [compute resource](compute.html)

- `ip_address` - (Required) IP address to use, preferably this comes from an [elastic IP](ip_address.html)

## Attributes Reference

- `nic_id`: id of the NIC

- `network_id`: id of the Network (of the NIC)

## Import

This resource is automatically imported when you import a compute resource.
