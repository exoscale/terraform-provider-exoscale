---
page_title: "Exoscale: exoscale_sks_kubeconfig"
description: |-
  Manage Exoscale Scalable Kubernetes Service (SKS) Credentials (Kubeconfig).
---

# exoscale\_sks\_kubeconfig

Manage Exoscale [Scalable Kubernetes Service (SKS)](https://community.exoscale.com/documentation/sks/) Credentials (*Kubeconfig*).

!> **WARNING:** This resource stores sensitive information in your Terraform state. Please be sure to correctly understand implications and how to mitigate potential risks before using it.


## Usage

```hcl
resource "exoscale_sks_cluster" "my_sks_cluster" {
  zone = "ch-gva-2"
  name = "my-sks-cluster"
}

resource "exoscale_sks_kubeconfig" "my_sks_kubeconfig" {
  cluster_id = exoscale_sks_cluster.my_sks_cluster.id
  zone       = exoscale_sks_cluster.my_sks_cluster.zone

  user   = "kubernetes-admin"
  groups = ["system:masters"]
}
```

Please refer to the [examples](https://github.com/exoscale/terraform-provider-exoscale/tree/master/examples/)
directory for complete configuration examples.


## Arguments Reference

[zone]: https://www.exoscale.com/datacenters/

* `cluster_id` - (Required) The parent [exoscale_sks_cluster](./sks_cluster.md) ID.
* `zone` - (Required) The Exoscale [Zone][zone] name.
* `groups` - (Required) Group names in the generated Kubeconfig. The certificate present in the Kubeconfig will have these roles set in the Organization field.
* `user` - (Required) User name in the generated Kubeconfig. The certificate present in the Kubeconfig will also have this name set for the CN field.

* `ttl_seconds` - The Time-to-Live of the Kubeconfig, after which it will expire / become invalid (seconds; default: 2592000 = 30 days).
* `early_renewal_seconds` - If set, the resource will consider the Kubeconfig to have expired the given number of seconds before its actual CA certificate or client certificate expiry time. This can be useful to deploy an updated Kubeconfig in advance of the expiration of its internal current certificate. Note however that the old certificate remains valid until its true expiration time since this resource does not (and cannot) support revocation. Also note this advance update can only take place if the Terraform configuration is applied during the early renewal period (seconds; default: 0).

## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `kubeconfig` - The generated Kubeconfig (YAML content).


## Automatic Renewal

This resource considers its instances to have been deleted after either their validity period ends or the early renewal period is reached. Past this period, applying the Terraform configuration will cause a new Kubeconfig to be generated.

Therefore in a development environment with frequent deployments, it may be convenient to set a relatively-short expiration time and use early renewal to automatically provision a new Kubeconfig when the current one is about to expire.

The creation of a new Kubeconfig may of course cause dependent resources to be updated or replaced, depending on the lifecycle rules applying to those resources.
