# Secure Shell (SSH) Keys

This example demonstrates how to setup
[Secure Shell (SSH) keys](https://community.exoscale.com/documentation/compute/ssh-keypairs/)
to access your compute instances, using the `exoscale_ssh_key` resource.

Please refer to the [main.tf](./main.tf) Terraform configuration file.

```console
$ terraform init
$ terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET

...

my_instance (remote-exec): Connected!
my_instance (remote-exec): Linux my-instance 5.15.0-41-generic #44-Ubuntu SMP Wed Jun 22 14:20:53 UTC 2022 x86_64 x86_64 x86_64 GNU/Linux

...

Outputs:

ssh_connection = "ssh -i id_ssh ubuntu@194.182.160.162"
```
