# IPv6

This example show how to activate the IPv6 address feature `ip6 = true` as well as setting the Security Group Ingress rules to enable SSH ports.

Initializing Terraform will ask for the Exoscale provider and some credentials.

```
$ terraform init
```

Then create the machine.

```
$ terraform apply

...

Outputs:

ip6_address = 2a04:c46:c00:a07:45e:42ff:fe00:13
ip_address = 89.145.160.14
username = centos

$ ssh -6 centos@2a04:c46:c00:a07:45e:42ff:fe00:13
[centos@test-ipv6 ~] $

$ ssh -4 centos@89.145.160.14
[centos@test-ipv6 ~] $ ip addr

1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN qlen 1
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host
       valid_lft forever preferred_lft forever
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc pfifo_fast state UP qlen 1000
    link/ether 06:5e:42:00:00:13 brd ff:ff:ff:ff:ff:ff
    inet 89.145.160.14/22 brd 89.145.163.255 scope global eth0
       valid_lft forever preferred_lft forever
    inet6 2a04:c46:c00:a07:45e:42ff:fe00:13/64 scope global mngtmpaddr dynamic
       valid_lft 86388sec preferred_lft 14388sec
    inet6 fe80::45e:42ff:fe00:13/64 scope link
       valid_lft forever preferred_lft forever
```

It's even possible to activate IPv6 on existing machines without having to stop or reboot them.
