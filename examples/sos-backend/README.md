# SOS as Terraform Backend

This example demonstrates how to configure Terraform to use a
[Simple Object Storage (SOS)](https://community.exoscale.com/documentation/storage/) bucket
as [Terraform S3 backend](https://www.terraform.io/docs/backends/types/s3.html)
to persist its state (`terraform.tfstate` object).

Please refer to the [providers.tf](./providers.tf) Terraform configuration file.

One should note:

* the S3 backend is coupled to some AWS-specific nomenclature which requires
  `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY` credentials to match your
  Exoscale API credentials.

* the specified `bucket` _must_ already exist; example given using the
  [Exoscale CLI](https://github.com/exoscale/cli/):
  `exo storage mb -z <zone> sos://<bucket>`

```console
export AWS_ACCESS_KEY_ID=$EXOSCALE_API_KEY AWS_SECRET_ACCESS_KEY=$EXOSCALE_API_SECRET
terraform init

Output

Initializing the backend...

Successfully configured the backend "s3"! Terraform will automatically
use this backend unless the backend configuration changes.

Initializing provider plugins...
- Finding latest version of exoscale/exoscale...
- Installing exoscale/exoscale v0.39.0...
- Installed exoscale/exoscale v0.39.0 (signed by a HashiCorp partner, key ID 81426F034A3D05F7)

...

Terraform has been successfully initialized!
```
