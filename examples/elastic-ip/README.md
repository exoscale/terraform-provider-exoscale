# Elastic IP (EIP)

This example demonstrates how to setup and associate an
[Elastic IP (EIP)](https://community.exoscale.com/documentation/compute/eip/)
to your compute instances, using the `exoscale_elastic_ip` resource.

Please refer to the [main.tf](./main.tf) Terraform configuration file.

One should note the Elastic IP address being associated to the loopbak (`lo`) interface thanks
to `cloud-init` ([cloud-config.yaml](./cloud-config.yaml.tpl)).

```console
$ terraform init
$ terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET

...

my_instance (remote-exec): Connected!
my_instance (remote-exec): 1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
my_instance (remote-exec):     inet 127.0.0.1/8 scope host lo
my_instance (remote-exec):     inet 194.182.162.134/32 scope global lo

...

Outputs:

my_elastic_ip = "194.182.162.134"
ssh_connection = "ssh -i id_ssh ubuntu@194.182.161.4"
```
