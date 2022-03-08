---
page_title: "Exoscale: exoscale_sks_kubeconfig"
description: |-
  Provides an Exoscale SKS cluster kubeconfig.
---

# exoscale\_sks\_kubeconfig

Provides an Exoscale [SKS][sks-doc] Kubeconfig resource. This can be used to create a configuration file (Kubeconfig) to interact with SKS clusters.


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

resource "exoscale_sks_kubeconfig" "prod_admin" {
  zone = local.zone

  ttl_seconds = 3600
  early_renewal_seconds = 300
  cluster_id = exoscale_sks_cluster.prod.id
  user = "kubernetes-admin"
  groups = ["system:masters"]
}

output "kubeconfig" {
  value = exoscale_sks_kubeconfig.prod_admin.kubeconfig
  sensitive = true
}
```


## Arguments Reference

* `zone` - (Required) The name of the [zone][zone] of the target SKS cluster.
* `cluster_id` - (Required) The ID of the target SKS cluster.
* `ttl_seconds` - The number of seconds after initial issuing that the Kubeconfig will become invalid.
* `early_renewal_seconds` - If set, the resource will consider the Kubeconfig to have expired the given number of seconds before its actual ca certificate or client certificate expiry time. This can be useful to deploy an updated Kubeconfig in advance of the expiration of its internal current certificate. Note however that the old certificate remains valid until its true expiration time since this resource does not (and cannot) support certificate revocation. Note also that this advance update can only be performed should the Terraform configuration be applied during the early renewal period.
* `user` - User name in the generated Kubeconfig. The certificate present in the Kubeconfig will also have this name set for the CN field.
* `groups` - Group names in the generated Kubeconfig. The certificate present in the Kubeconfig will have these roles set in the Organization field.

## Attributes Reference

In addition to the arguments listed above, the following attributes are exported:

* `kubeconfig` - The generated Kubeconfig for use to interact with the SKS cluster.

## Automatic Renewal

This resource considers its instances to have been deleted after either their validity period ends or the early renewal period is reached. At this time, applying the Terraform configuration will cause a new certificate to be generated for the instance.

Therefore in a development environment with frequent deployments, it may be convenient to set a relatively-short expiration time and use early renewal to automatically provision a new certificate when the current one is about to expire.

The creation of a new certificate may of course cause dependent resources to be updated or replaced, depending on the lifecycle rules applying to those resources.

[sks-doc]: https://community.exoscale.com/documentation/sks/
[zone]: https://www.exoscale.com/datacenters/
