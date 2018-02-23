# Multiple private network

It creates a _Private network_ with two machines having two interfaces. `eth0` is
the provided NIC and a second one is also present, called `eth1`. It is configured
via the [Cloud-Init] description file `cloud-config.yaml` (`data.tf`).

```console
$ terraform apply
```

Then log into one machine. It has both eth0 and eth1 set up and you can ping the
other machine from there on the non-routable network (`network.tf`).

```
$ ssh ubuntu@...
ubuntu@demo-machine-0 $ ip addr
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
    inet6 ::1/128 scope host 
       valid_lft forever preferred_lft forever
2: eth0: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc pfifo_fast state UP group default qlen 1000
    link/ether 06:2c:20:00:00:15 brd ff:ff:ff:ff:ff:ff
    inet 159.100.251.208/22 brd 159.100.251.255 scope global eth0
       valid_lft forever preferred_lft forever
    inet6 fe80::42c:20ff:fe00:15/64 scope link
       valid_lft forever preferred_lft forever
3: eth1: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc pfifo_fast state UP group default qlen 1000
    link/ether 0a:24:aa:00:05:9b brd ff:ff:ff:ff:ff:ff
    inet 192.168.0.1/24 brd 192.168.0.255 scope global eth1
       valid_lft forever preferred_lft forever
    inet6 fe80::824:aaff:fe00:59b/64 scope link
       valid_lft forever preferred_lft forever


ubuntu@demo-machine-0 $ ping -c 3 192.168.0.2
PING 192.168.0.2 (192.168.0.2) 56(84) bytes of data.
64 bytes from 192.168.0.2: icmp_seq=1 ttl=64 time=0.767 ms
64 bytes from 192.168.0.2: icmp_seq=2 ttl=64 time=0.590 ms
64 bytes from 192.168.0.2: icmp_seq=3 ttl=64 time=0.632 ms

--- 192.168.0.2 ping statistics ---
3 packets transmitted, 3 received, 0% packet loss, time 2011ms
rtt min/avg/max/mdev = 0.590/0.663/0.767/0.075 ms
```

Profit!

## Documentation

To better understand the concept of [Private network][] in
the CloudStack world, consult [our documentation][Community].

[Community]: https://community.exoscale.com/
[Private network]: https://community.exoscale.com/documentation/compute/privnet/
[Cloud-Init]: https://community.exoscale.com/documentation/compute/cloud-init/
