---
layout: "exoscale"
page_title: "Exoscale: exoscale_sks_nodepool"
sidebar_current: "docs-exoscale-sks_nodepool"
description: |-
  Provides an Exoscale SKS Nodepool resource.
---

# exoscale\_sks\_nodepool

Provides an Exoscale [SKS][sks-doc] Nodepool resource. This can be used to create, modify, and delete SKS Nodepools.


## Example Usage

```hcl
locals {
  zone = "de-fra-1"
}

resource "exoscale_sks_cluster" "prod" {
  zone    = local.zone
  name    = "prod"
  version = "1.20.0"
}

resource "exoscale_sks_nodepool" "ci-builders" {
  zone          = local.zone
  cluster_id    = exoscale_sks_cluster.prod.id
  name          = "ci-builders"
  instance_type = "medium"
  size          = 3
}
```


## Arguments Reference

* `zone` - (Required) The name of the [zone][zone] to deploy the SKS Nodepool into.
* `cluster_id` - (Required) The ID of the parent SKS cluster.
* `size` - (Required) The number of Compute instances the SKS Nodepool manages.
* `name` - (Required) The name of the SKS Nodepool.
* `instance_type` (Required) - The type of Compute instances managed by the SKS Nodepool.
* `disk_size` - The disk size of the Compute instances managed by the SKS Nodepool (default: `50`).
* `anti_affinity_group_ids` - The list of Anti-Affinity Groups (IDs) the Compute instances managed by the SKS Nodepool are member of.
* `security_group_ids` - The list of Security Groups (IDs) the Compute instances managed by the SKS Nodepool are member of.
* `description` - The description of the SKS Nodepool.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the SKS Nodepool.
* `state` - The current state of the SKS Nodepool.
* `created_at` - The creation date of the SKS Nodepool.
* `instance_pool_id` - The ID of the Instance Pool managed by the SKS Nodepool.
* `template_id` - The ID of the Compute instance template used by the SKS Nodepool members.
* `version` - The Kubernetes version of the SKS Nodepool members.


## Import

An existing SKS Nodepool can be imported as a resource by ID:

```console
$ terraform import exoscale_sks_nodepool.ci-builders eb556678-ec59-4be6-8c54-0406ae0f6da6
```


[r-sks_cluster]: sks_cluster.html
[sks-doc]: #
[zone]: https://www.exoscale.com/datacenters/

