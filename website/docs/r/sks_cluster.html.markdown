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
  
  labels = {
    env = "prod"
  }
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
* `exoscale_ccm` - Deploy the Exoscale [Cloud Controller Manager][exo-ccm] in the SKS cluster control plane (default: `true`).
* `metrics_server` - Deploy the [Kubernetes Metrics Server][k8s-ms] in the SKS cluster control plane (default: `true`).
* `auto_upgrade` - Enable automatic upgrading of the SKS cluster control plane Kubernetes version (default: `false`).
* `oidc` - An OpenID Connect configuration to provide to the Kubernetes API server. Structure is documented below.
* `labels` - A map of key/value labels.
 
The `oidc` block supports:

* `client_id` - (Required) The OpenID client ID.
* `issuer_url` - (Required) The OpenID provider URL.
* `groups_claim` - An OpenID JWT claim to use as the user's group.
* `groups_prefix` - An OpenID prefix prepended to group claims.
* `required_claim` - A `KEY=VALUE` pair that describes a required claim in the OpenID Token.
* `username_claim` - An OpenID JWT claim to use as the user name.
* `username_prefix` - An OpenID prefix prepended to username claims.

## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `id` - The ID of the SKS cluster.
* `endpoint` - The Kubernetes public API endpoint of the SKS cluster.
* `state` - The current state of the SKS cluster.
* `created_at` - The creation date of the SKS cluster.
* `nodepools` - The list of [SKS Nodepools][r-sks_nodepool] (IDs) attached to the SKS cluster.


## Import

An existing SKS cluster can be imported as a resource by specifying `ID@ZONE`:

```console
$ terraform import exoscale_sks_cluster.example eb556678-ec59-4be6-8c54-0406ae0f6da6@de-fra-1
```

~> **NOTE:** Importing a SKS cluster resource doesn't import related [`exoscale_sks_nodepool`][r-sks_nodepool] resources.


[cni]: https://www.cni.dev/
[exo-ccm]: https://github.com/exoscale/exoscale-cloud-controller-manager
[k8s-ms]: https://github.com/kubernetes-sigs/metrics-server
[r-sks_nodepool]: sks_nodepool.html
[sks-doc]: https://community.exoscale.com/documentation/sks/
[zone]: https://www.exoscale.com/datacenters/

