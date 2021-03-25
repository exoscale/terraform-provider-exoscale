---
layout: "exoscale"
page_title: "Exoscale: exoscale_sks_cluster"
sidebar_current: "docs-exoscale-sks_cluster"
description: |-
  Provides an Exoscale SKS cluster resource.
---

# exoscale\_sks\_cluster

Provides an Exoscale [SKS][sks-doc] cluster resource. This can be used to create, modify, and delete SKS clusters.


## Example Usage

```hcl
locals {
  zone = "de-fra-1"
}

resource "exoscale_sks_cluster" "prod" {
  zone    = local.zone
  name    = "prod"
  version = "1.20.2"
}

output "sks_endpoint" {
  value = exoscale_sks_cluster.prod.endpoint
}
```


## Arguments Reference

* `zone` - (Required) The name of the [zone][zone] to deploy the SKS cluster into.
* `name` - (Required) The name of the SKS cluster.
* `description` - The description of the SKS cluster.
* `service_level` - The service level of the SKS cluster control plane (default: `"pro"`).
* `version` - The Kubernetes version of the SKS cluster control plane (default: latest version available from the API).
* `cni` - The Kubernetes [CNI][cni] plugin to be deployed in the SKS cluster control plane (default: `"calico"`).
* `addons` - A list of optional add-ons to be deployed in the SKS cluster control plane (default: `["exoscale-cloud-controller"]`).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the SKS cluster.
* `endpoint` - The Kubernetes public API endpoint of the SKS cluster.
* `state` - The current state of the SKS cluster.
* `created_at` - The creation date of the SKS cluster.
* `nodepools` - The list of [SKS Nodepools][r-sks_nodepool] (IDs) attached to the SKS cluster.


## Import

An existing SKS cluster can be imported as a resource by ID:

```console
$ terraform import exoscale_sks_cluster.prod eb556678-ec59-4be6-8c54-0406ae0f6da6
```

~> **NOTE:** Importing a SKS cluster resource also imports related [`exoscale_sks_nodepool`][r-sks_nodepool] resources.


[cni]: https://www.cni.dev/
[r-sks_nodepool]: sks_nodepool.html
[sks-doc]: https://community.exoscale.com/documentation/sks/
[zone]: https://www.exoscale.com/datacenters/

