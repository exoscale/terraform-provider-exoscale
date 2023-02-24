# Network Load Balancer (NLB)

This example demonstrates how to setup and associate a
[Network Load Balancer (NLB)](https://community.exoscale.com/documentation/compute/network-load-balancer/)
to your [Instance Pools](https://community.exoscale.com/documentation/compute/instance-pools/),
using the `exoscale_nlb` and `exoscale_nlb_service` resources.

Please refer to the [main.tf](./main.tf) Terraform configuration file.

One should note the sample web/http (TCP/80) service being setup thanks to `cloud-init`
([cloud-config.yaml](./cloud-config.yaml)).

```console
terraform init
terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET

...

Outputs:

my_nlb_ip_address = "194.182.160.241"
my_nlb_service = "http://194.182.160.241:80"
ssh_connection = <<EOT
ssh -i id_ssh ubuntu@185.19.30.210  # pool-95dde-htnmr
ssh -i id_ssh ubuntu@194.182.160.27  # pool-95dde-wvzui
ssh -i id_ssh ubuntu@159.100.240.100  # pool-95dde-ojufs
EOT

$ wget -qO- http://194.182.160.241:80
<H1>Hello World!</H1>
```
