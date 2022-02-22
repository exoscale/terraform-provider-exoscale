---
page_title: "Exoscale: exoscale_sks_nodepool"
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

resource "exoscale_security_group" "sks" {
  name = "sks"
}

resource "exoscale_security_group_rules" "sks" {
  security_group = exoscale_security_group.sks.name

  ingress {
    description              = "Calico traffic"
    protocol                 = "UDP"
    ports                    = ["4789"]
    user_security_group_list = [exoscale_security_group.sks.name]
  }

  ingress {
    description = "Nodes logs/exec"
    protocol  = "TCP"
    ports     = ["10250"]
    cidr_list = ["0.0.0.0/0", "::/0"]
  }

  ingress {
    description = "NodePort services"
    protocol    = "TCP"
    cidr_list   = ["0.0.0.0/0", "::/0"]
    ports       = ["30000-32767"]
  }
}

resource "exoscale_sks_cluster" "prod" {
  zone    = local.zone
  name    = "prod"
  version = "1.20.3"
}

resource "exoscale_sks_nodepool" "ci-builders" {
  zone               = local.zone
  cluster_id         = exoscale_sks_cluster.prod.id
  name               = "ci-builders"
  instance_type      = "standard.medium"
  size               = 3
  security_group_ids = [exoscale_security_group.sks.id]
  
  labels = {
    role = "ci-builders"
  }

  taints = {
    ci = "ci:NoSchedule"
  }
}
```


## Arguments Reference

* `zone` - (Required) The name of the [zone][zone] to deploy the SKS Nodepool into.
* `cluster_id` - (Required) The ID of the parent SKS cluster.
* `size` - (Required) The number of Compute instances the SKS Nodepool manages.
* `name` - (Required) The name of the SKS Nodepool.
* `instance_type` (Required) - The [type][type] of Compute instances managed by the SKS Nodepool (format: `FAMILY.SIZE`, e.g. `standard.medium`, `memory.huge`).
* `instance_prefix` - The string to add as prefix to managed Compute instances name (default `pool`).
* `disk_size` - The disk size of the Compute instances managed by the SKS Nodepool (default: `50`).
* `anti_affinity_group_ids` - The list of Anti-Affinity Groups (IDs) the Compute instances managed by the SKS Nodepool are member of.
* `security_group_ids` - The list of Security Groups (IDs) the Compute instances managed by the SKS Nodepool are member of.
* `private_network_ids` - The list of Private Networks (IDs) to be attached to the Compute instances managed by the SKS Nodepool.
* `description` - The description of the SKS Nodepool.
* `deploy_target_id` - A Deploy Target ID to deploy managed Compute instances to.
* `labels` - A map of key/value labels.
* `taints` - A map of key/value [Kubernetes taints][k8s-taints] (value format: `VALUE:EFFECT`).


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the SKS Nodepool.
* `state` - The current state of the SKS Nodepool.
* `created_at` - The creation date of the SKS Nodepool.
* `instance_pool_id` - The ID of the Instance Pool managed by the SKS Nodepool.
* `template_id` - The ID of the Compute instance template used by the SKS Nodepool members.
* `version` - The Kubernetes version of the SKS Nodepool members.


## Import

An existing SKS Nodepool can be imported as a resource by `<CLUSTER-ID>/<NODEPOOL-ID>@<ZONE>`:

```console
$ terraform import exoscale_sks_nodepool.ci-builders eb556678-ec59-4be6-8c54-0406ae0f6da6/8c08b92a-e673-47c7-866e-dc009f620a82@de-fra-1
```


[k8s-taints]: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/
[r-sks_cluster]: ../resources/sks_cluster
[sks-doc]: https://community.exoscale.com/documentation/sks/
[type]: https://www.exoscale.com/pricing/#/compute/
[zone]: https://www.exoscale.com/datacenters/

