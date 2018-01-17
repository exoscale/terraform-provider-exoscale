# Multiple private network

It creates a _Private network_ with two machines having two interfaces.

```console
$ terraform apply
```

Then log into one machine and ping the other.

```
$ ssh ubuntu@...
ubuntu@demo-machine-0 $ ping 192.168.0.2
PING 192.168.0.2 (192.168.0.2) 56(84) bytes of data.
64 bytes from 192.168.0.2: icmp_seq=1 ttl=64 time=1.50 ms
...
```

Profit!
