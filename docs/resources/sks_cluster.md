---
page_title: "Exoscale: exoscale_sks_cluster"
description: |-
  Manage Exoscale Scalable Kubernetes Service (SKS) Clusters.
---

# exoscale\_sks\_cluster

Manage Exoscale [Scalable Kubernetes Service (SKS)](https://community.exoscale.com/documentation/sks/) Clusters.


## Usage

```hcl
resource "exoscale_sks_cluster" "my_sks_cluster" {
  zone = "ch-gva-2"
  name = "my-sks-cluster"
}

output "my_sks_cluster_endpoint" {
  value = exoscale_sks_cluster.my_sks_cluster.endpoint
}
```

Next step is to attach [exoscale_sks_nodepool](./sks_nodepool.md)(s) to the cluster.

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/
[ccm]: https://github.com/exoscale/exoscale-cloud-controller-manager/
[cni]: https://www.cni.dev/
[ms]: https://github.com/kubernetes-sigs/metrics-server/

* `zone` - (Required) The Exoscale [Zone][zone] name.
* `name` - (Required) The SKS cluster name.

* `description` - A free-form text describing the cluster.
* `auto_upgrade` - Enable automatic upgrading of the control plane version (boolean; default: `false`).
* `exoscale_ccm` - Deploy the Exoscale [Cloud Controller Manager][ccm] in the control plane (boolean; default: `true`; may only be set at creation time).
* `metrics_server` - Deploy the [Kubernetes Metrics Server][ms] in the control plane (boolean; default: `true`; may only be set at creation time).
* `service_level` - The service level of the control plane (`pro` or `starter`; default: `pro`; may only be set at creation time).
* `version` - The version of the control plane (default: latest version available from the API; see `exo compute sks versions` for reference; may only be set at creation time).
* `labels` - A map of key/value labels.

* `oidc` - (Block) An OpenID Connect configuration to provide to the Kubernetes API server (may only be set at creation time). Structure is documented below.

### `oidc` block

* `client_id` - (Required) The OpenID client ID.
* `issuer_url` - (Required) The OpenID provider URL.

* `groups_claim` - An OpenID JWT claim to use as the user's group.
* `groups_prefix` - An OpenID prefix prepended to group claims.
* `required_claim` - A map of key/value pairs that describes a required claim in the OpenID Token.
* `username_claim` - An OpenID JWT claim to use as the user name.
* `username_prefix` - An OpenID prefix prepended to username claims.


## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The SKS cluster ID.
* `aggregation_ca` - The CA certificate (in PEM format) for TLS communications between the control plane and the aggregation layer (e.g. `metrics-server`).
* `control_plane_ca` - The CA certificate (in PEM format) for TLS communications between control plane components.
* `created_at` - The cluster creation date.
* `endpoint` - The cluster API endpoint.
* `kubelet_ca` - The CA certificate (in PEM format) for TLS communications between kubelets and the control plane.
* `nodepools` - The list of [exoscale_sks_nodepool](./sks_nodepool.md) (IDs) attached to the cluster.
* `state` - The cluster state.


## Import

An existing SKS cluster may be imported by `<ID>@<zone>`:

```console
$ terraform import \
  exoscale_sks_cluster.my_sks_cluster \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6@ch-gva-2
```

~> **NOTE:** Importing an `exoscale_sks_cluster` resource does _not_ import related `exoscale_sks_nodepool` resources.
