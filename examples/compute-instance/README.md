# Compute Instance

This example demonstrates how to setup an
[Instance Pool](https://community.exoscale.com/documentation/compute/),
using the `exoscale_compute_instance` resource.

Please refer to the [main.tf](./main.tf) Terraform configuration file.

```console
$ terraform init
$ terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET

...

Outputs:


ssh_connection = <<EOT
ssh -i id_ssh ubuntu@159.100.241.199  # my-big-instance
ssh -i id_ssh ubuntu@159.100.241.252  # my-small-instance
EOT
```
