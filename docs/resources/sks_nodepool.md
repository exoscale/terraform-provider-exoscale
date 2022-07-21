---
page_title: "Exoscale: exoscale_sks_nodepool"
description: |-
  Manage Exoscale Scalable Kubernetes Service (SKS) Node Pools.
---

# exoscale\_sks\_nodepool

Manage Exoscale [Scalable Kubernetes Service (SKS)](https://community.exoscale.com/documentation/sks/) Node Pools.


## Usage

```hcl
resource "exoscale_sks_cluster" "my_sks_cluster" {
  zone = "ch-gva-2"
  name = "my-sks-cluster"
}

resource "exoscale_sks_nodepool" "my_sks_nodepool" {
  cluster_id         = exoscale_sks_cluster.my_sks_cluster.id
  zone               = exoscale_sks_cluster.my_sks_cluster.zone
  name               = "my-sks-nodepool"

  instance_type      = "standard.medium"
  size               = 3
}
```

Please refer to the [examples](../../examples/) directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/
[cli]: https://github.com/exoscale/cli/
[taints]: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/

* `cluster_id` - (Required) The parent [SKS cluster](./sks_cluster) ID.
* `zone` - (Required) The name of the [zone][zone] of the parent SKS cluster.
* `name` - (Required) The name of the SKS node pool.
* `instance_type` (Required) - The managed compute instances type (`<family>.<size>`, e.g. `standard.medium`; use the [Exoscale CLI][cli] - `exo compute instance-type list` - for the list of available types).

* `description` - A free-form text describing the SKS node pool.
* `deploy_target_id` - A deploy target ID to deploy managed compute instances to.
* `instance_prefix` - The string used to prefix the managed compute instances name (default `pool`).
* `disk_size` - The disk size of the compute instances managed by the SKS node pool (GiB; default: `50`).
* `labels` - A map of key/value labels.
* `taints` - A map of key/value Kubernetes [taints][taints] (`<value>:<effect>`).

* `anti_affinity_group_ids` - A list of [anti-affinity group](./anti_affinity_group) IDs to be attached to the compute instances managed by the SKS node pool.
* `private_network_ids` - A list of [private network](./private_network) IDs to be attached to the compute instances managed by the SKS node pool.
* `security_group_ids` - A list of [security group](./security_groups) IDs to be attached to the compute instances managed by the SKS node pool.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the SKS node pool.
* `created_at` - The creation date of the SKS node pool.
* `instance_pool_id` - The ID of the instance pool managed by the SKS node pool.
* `state` - The current state of the SKS node pool.
* `template_id` - The ID of the compute instance template used by the SKS node pool members.
* `version` - The Kubernetes version of the SKS node pool members.


## Import

An existing SKS node pool may be imported by `<cluster-ID>/<nodepool-ID>@<zone>`:

```console
$ terraform import \
  exoscale_sks_nodepool.my_sks_nodepool \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6/9ecc6b8b-73d4-4211-8ced-f7f29bb79524@ch-gva-2
```
