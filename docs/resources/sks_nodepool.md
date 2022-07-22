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

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/
[cli]: https://github.com/exoscale/cli/
[taints]: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/

* `cluster_id` - (Required) The parent [exoscale_sks_cluster](./sks_cluster.md) ID.
* `zone` - (Required) The Exoscale [Zone][zone] name.
* `name` - (Required) The SKS node pool name.
* `instance_type` (Required) - The managed compute instances type (`<family>.<size>`, e.g. `standard.medium`; use the [Exoscale CLI][cli] - `exo compute instance-type list` - for the list of available types).

* `description` - A free-form text describing the pool.
* `deploy_target_id` - A deploy target ID.
* `instance_prefix` - The string used to prefix the managed instances name (default `pool`).
* `disk_size` - The managed instances disk size (GiB; default: `50`).
* `labels` - A map of key/value labels.
* `taints` - A map of key/value Kubernetes [taints][taints] (`<value>:<effect>`).

* `anti_affinity_group_ids` - A list of [exoscale_anti_affinity_group](./anti_affinity_group.md) (IDs) to be attached to the managed instances.
* `private_network_ids` - A list of [exoscale_private_network](./private_network.md) (IDs) to be attached to the managed instances.
* `security_group_ids` - A list of [exoscale_security_group](./security_group.md) (IDs) to be attached to the managed instances.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The SKS node pool ID.
* `created_at` - The pool creation date.
* `instance_pool_id` - The underlying [exoscale_instance_pool](./instance_pool.md) ID.
* `state` - The current pool state.
* `template_id` - The managed instances template ID.
* `version` - The managed instances version.


## Import

An existing SKS node pool may be imported by `<cluster-ID>/<nodepool-ID>@<zone>`:

```console
$ terraform import \
  exoscale_sks_nodepool.my_sks_nodepool \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6/9ecc6b8b-73d4-4211-8ced-f7f29bb79524@ch-gva-2
```
