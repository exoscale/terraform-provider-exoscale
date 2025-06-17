# IPv6

This example demonstrates how to activate
[IPv6](https://community.exoscale.com/product/networking/ip/quick-start/)
on your compute instances, thanks to the `ipv6 = true` argument.

Please refer to the [main.tf](./main.tf) Terraform configuration file.

One should note the IPv6-specific security groups rules set up for the SSH access
(see the [ssh.tf](./ssh.tf) file).

```console
terraform init
terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET

...

my_instance (remote-exec): Connected!
my_instance (remote-exec): 2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq state UP group default qlen 1000
my_instance (remote-exec):     link/ether 06:0d:e0:00:03:0e brd ff:ff:ff:ff:ff:ff
my_instance (remote-exec):     inet 185.19.30.210/22 metric 100 brd 185.19.31.255 scope global eth0
my_instance (remote-exec):     inet6 2a04:c43:e00:63a3:40d:e0ff:fe00:30e/64 scope global dynamic mngtmpaddr noprefixroute
my_instance (remote-exec):     inet6 fe80::40d:e0ff:fe00:30e/64 scope link

...

Outputs:

my_instance_ipv4 = "185.19.30.210"
my_instance_ipv6 = "2a04:c43:e00:63a3:40d:e0ff:fe00:30e"
ssh_connection = "ssh -i id_ssh ubuntu@185.19.30.210"
```

It's also possible to activate IPv6 on existing instances _without_ having to stop or reboot them.
