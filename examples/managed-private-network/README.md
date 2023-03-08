# Managed Private Network

This example demonstrates how to setup and associate a
[(Managed) Private Network](https://community.exoscale.com/documentation/compute/private-networks/)
to your compute instances, using the `exoscale_private_network` resource.

Please refer to the [main.tf](./main.tf) Terraform configuration file.

One should note:

* the network setup:
  - network mask `255.255.255.0` <-> `/24` (256 IP addresses, out of which 253 are usable: `1` to `253`)
  - dynamic IP range from `10.0.0.50` to `10.0.0.250`

* the _static_ DHCP lease - IP address `10.0.0.1` attributed to the `my_instance_static` instance

* the _dynamic_ IP address which is granted by DHCP to the `my_instance_dynamic` instance

* the `eth1` (private network) interface being configured thanks to `cloud-init`
  ([cloud-config.yaml](./cloud-config.yaml))


```console
terraform init
terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET

...

exoscale_compute_instance.my_instance_static (remote-exec): 3: eth1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq state UP group default qlen 1000
exoscale_compute_instance.my_instance_static (remote-exec):     inet 10.0.0.1/24 metric 100 brd 10.0.0.255 scope global dynamic eth1

...

exoscale_compute_instance.my_instance_dynamic (remote-exec): 3: eth1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc mq state UP group default qlen 1000
exoscale_compute_instance.my_instance_dynamic (remote-exec):     inet 10.0.0.138/24 metric 100 brd 10.0.0.255 scope global dynamic eth1

...

Outputs:

ssh_connection_dynamic = "ssh -i id_ssh ubuntu@185.19.30.33"
ssh_connection_static = "ssh -i id_ssh ubuntu@185.19.30.210"
```

## Inspecting the private network

Connect to any of them and use `nmap` to inspect what is in the private
network.

```
ubuntu@my-instance-dynamic $ sudo apt install nmap
ubuntu@my-instance-dynamic $ nmap -sP "10.0.0.*"

Starting Nmap 7.60 ( https://nmap.org ) at 2018-09-24 13:27 UTC
Nmap scan report for 10.0.0.1
Host is up (0.0017s latency).
Nmap scan report for 10.0.0.2
Host is up (0.0017s latency).
Nmap scan report for my-instance-dynamic (10.0.0.101)
Host is up (0.000055s latency).
Nmap scan report for 10.0.0.231
Host is up (0.0012s latency).
Nmap done: 256 IP addresses (4 hosts up) scanned in 2.51 seconds
```

More inspection shows us where the DHCP server lives, the `.254` address in our
case.

```
$ arp -a
? (10.0.0.254) at 0e:00:00:00:00:00 [ether] on eth1
? (10.0.0.2) at 0a:89:90:00:2c:ca [ether] on eth1
? (10.0.0.1) at 0a:0e:38:00:2c:ca [ether] on eth1
_gateway (159.100.241.1) at fe:6e:f0:00:00:71 [ether] on eth0
? (10.0.0.231) at 0a:f6:92:00:2c:ca [ether] on eth1
```

## Unmanaged private network

Should you want to _not_ use DHCP and attribute IP addresses _manually_ to the private network interface:

* remove the `netmask`, `start_ip` and `end_ip` from the `exoscale_private_network` resource

* replace `dhcp4: true` by `addresses: ["<actual-IP-address>"]` in the `cloud-init.yaml` configuration
