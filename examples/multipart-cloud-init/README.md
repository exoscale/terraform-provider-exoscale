# Multipart Cloud-Init

This example demonstrates how to use [cloud-init](http://cloudinit.readthedocs.io/)
to configure your compute instances, using
[HashiCorp cloud-init](https://registry.terraform.io/providers/hashicorp/cloudinit/) provider.

Please refer to the [main.tf](./main.tf) Terraform configuration file.

One should note:

* the `cloud-init` **multipart** configuration crafted with the `cloudinit_config` resource

* along the templated [cloud-init.yaml](./cloud-init.yaml.tpl) (aka. `#cloud-config`) file

* and the templated [x-shellscript](./x-shellscript.sh.tpl) script


```console
terraform init
terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET

...

my_instance[0] (remote-exec): Connected!
my_instance[0] (remote-exec): Setup via Terraform v1.2.5

...

my_instance[1] (remote-exec): Connected!
my_instance[1] (remote-exec): Setup via Terraform v1.2.5

...

Outputs:

ssh_connection = <<EOT
ssh -i id_ssh ubuntu@159.100.240.115  # my-instance-1
ssh -i id_ssh ubuntu@185.19.31.147  # my-instance-2
EOT
```
