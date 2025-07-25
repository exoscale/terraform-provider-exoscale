# Examples

## [Database](./database)

This example demonstrates how to setup [DBaaS service](https://community.exoscale.com/documentation/dbaas/)
(Grafana in example) using `exoscale_database` resource.

## [Domain Name Service (DNS)](./dns)

This example demonstrates how to manage
[Domain Name Service (DNS)](https://community.exoscale.com/product/networking/dns/)
and point Resource Records (RR) to compute instances, using `exoscale_domain` and
`exoscale_domain_record` resources.

## [Elastic IP](./elastic-ip)

This example demonstrates how to setup and associate an
[Elastic IP (EIP)](https://community.exoscale.com/product/networking/eip/)
to your compute instances, using the `exoscale_elastic_ip` resource.

## [Import existing resources](./import-resources)

This example demonstrates how to start using Terraform along an already deployed infrastructure and
import existing resources - e.g. compute instance and its security groups - to match a newly-written
Terraform configuration.

## [Instance Pool](./instance-pool)

This example demonstrates how to setup an
[Instance Pool](https://community.exoscale.com/product/compute/instances/how-to/instance-pools/),
using the `exoscale_instance_pool` resource.

## [IPv6](./ipv6)

This example demonstrates how to activate
[IPv6](https://community.exoscale.com/product/networking/ip/quick-start/)
on your compute instances, thanks to the `ipv6 = true` argument.

## [Multipart Cloud-Init](./multipart-cloud-init)

This example demonstrates how to use [cloud-init](http://cloudinit.readthedocs.io/)
to configure your compute instances, using
[HashiCorp cloud-init](https://registry.terraform.io/providers/hashicorp/cloudinit/) provider.

## [Managed Private Network](./managed-private-network)

This example demonstrates how to setup and associate a
[(Managed) Private Network](https://community.exoscale.com/product/networking/private-network/)
to your compute instances, using the `exoscale_private_network` resource.
It also shows how to tweak the configuration to turn it into a _unmanaged_ private network.

## [Network Load Balancer (NLB)](./nlb)

This example demonstrates how to setup and associate a
[Network Load Balancer (NLB)](https://community.exoscale.com/product/networking/nlb/)
to your [Instance Pools](https://community.exoscale.com/product/compute/instances/how-to/instance-pools/),
using the `exoscale_nlb` and `exoscale_nlb_service` resources.

## [Scalable Kubernetes Service (SKS)](./sks)

This example demonstrates how to instantiate a
[Scalable Kubernetes Service (SKS)](https://community.exoscale.com/product/compute/containers/) cluster,
using the `exoscale_sks` and `exoscale_sks_nodepool` resource.

## [Simple Object Storage (SOS)](./sos)

This example demonstrates how to manage
[Simple Object Storage (SOS)](https://community.exoscale.com/documentation/storage/)
buckets and objects, using the stock
[S3/AWS](https://registry.terraform.io/providers/hashicorp/aws/) provider.

## [Secure Shell (SSH) Keys](./ssh-keys)

This example demonstrates how to setup
[Secure Shell (SSH) keys](https://community.exoscale.com/product/compute/instances/how-to/ssh-keypairs/)
to access your computes instances, using the `exoscale_ssh_key` resource.

## [IAM API Key SOS](./iam-bucket-access)

This example demonstrates how to create
[IAM API Key](https://community.exoscale.com/product/iam/operation/roles-policies/)
that has restricted access to an SOS bucket.

## [IAM API Key Compute](./iam-priv-instance)

This example demonstrates how to create
[IAM API Key](https://community.exoscale.com/product/iam/operation/roles-policies/)
that only allows private instances to be deployed.

## [IAM API Key DBaaS](./iam-dbaas-instance)

This example demonstrates how to create
[IAM API Key](https://community.exoscale.com/product/iam/operation/roles-policies/)
that only allows access to the specific instance of DBaaS service.

## [SOS as Terraform Backend](./sos-backend)

This example demonstrates how to configure Terraform to use a
[Simple Object Storage (SOS)](https://community.exoscale.com/documentation/storage/) bucket
as [Terraform S3 backend](https://www.terraform.io/docs/backends/types/s3.html)
to persist its state (`terraform.tfstate` object).


# External examples

- Oliver Moser's: [Prometheus Service Discovery Demo](https://github.com/olmoser/infracoders-reloaded)
