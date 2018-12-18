# Managed private network

A Managed Private Network is defined by a Private Network where lives a managed
DHCP server. A bare private network implies that each machine has to be
statically configured when it comes to network interface (NIC) on that network.

Now, it's as quick as having a dhcp client running on the eth1, eth2, ...
interfaces.

## Spawning the machines

The current example create a managed networks with four instances:

- demo-static-0 with a static DHCP lease set to 10.0.0.1
- demo-static-1 with a static DHCP lease set to 10.0.0.2
- demo-dynamic-0 and demo-dynamic-1 with dynamic leases

The managed privnet sets a dynamic IP range from 10.0.0.50 to 10.0.0.250.  The
255.255.255.0 netmask defines the network 10.0.0.0/24. The static leases may be
set (almost, see below) anywhere in the network.

```
$ terraform apply
```

Be patient and wait for the instances to be ready.

## Inspecting the private network network

Connect to any of them and use `nmap` to inspect what is in the private
network.

```
ubuntu@demo-dynamic-0 $ sudo apt install nmap
ubuntu@demo-dynamic-0 $ nmap -sP "10.0.0.*"

Starting Nmap 7.60 ( https://nmap.org ) at 2018-09-24 13:27 UTC
Nmap scan report for 10.0.0.1
Host is up (0.0017s latency).
Nmap scan report for 10.0.0.2
Host is up (0.0017s latency).
Nmap scan report for demo-dynamic-0 (10.0.0.101)
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

---

See also [the blog post announcing the feature][managed privnet].

[managed privnet]: https://www.exoscale.com/syslog/introducing-managed-private-networks/
