# Instance Pool

This example demonstrates how to setup an
[Instance Pool](https://community.exoscale.com/documentation/compute/instance-pools/),
using the `exoscale_instance_pool` resource.

Please refer to the [main.tf](./main.tf) Terraform configuration file.

```console
terraform init
terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET

...

Outputs:

ssh_connection = <<EOT
ssh -i id_ssh ubuntu@185.19.30.210  # pool-e3c21-fbkfc
ssh -i id_ssh ubuntu@185.19.30.221  # pool-e3c21-xrkjd
ssh -i id_ssh ubuntu@159.100.240.228  # pool-e3c21-vkcal
EOT
```
