# Domain Name Service (DNS)

This example demonstrates how to manage
[Domain Name Service (DNS)](https://community.exoscale.com/documentation/dns/)
and point Resource Records (RR) to compute instances, using `exoscale_domain` and
`exoscale_domain_record` resources.

It creates the `example.exo` domain with four records:
- the (bare) domain **A**, pointing to the instance IPv4
- its **AAAA** sibling, pointing to the instance IPv6
- a **TXT** record
- a **CNAME**, aliasing `www.example.exo` to the (bare) domain

```console
$ terraform init
$ terraform apply \
  -var exoscale_api_key=$EXOSCALE_API_KEY \
  -var exoscale_api_secret=$EXOSCALE_API_SECRET

...

Outputs:

my_instance_ipv4 = "185.19.31.186"
my_instance_ipv6 = "2a04:c43:e00:63a3:43b:eaff:fe00:3f6"
```

Which can then be seen using [Exoscale CLI](https://github.com/exoscale/cli/):

```console
$ exo dns show example.exo
┼──────────┼──────┼─────────────┼─────────────────────────────────────────────────────────────────────┼──────┼──────┼
│    ID    │ NAME │ RECORD TYPE │                               CONTENT                               │ PRIO │ TTL  │
┼──────────┼──────┼─────────────┼─────────────────────────────────────────────────────────────────────┼──────┼──────┼
│ 35726890 │      │ SOA         │ ns1.exoscale.ch admin.dnsimple.com 1658219527 86400 7200 604800 300 │ 0    │ 3600 │
│ 35726895 │      │ NS          │ ns1.exoscale.ch                                                     │ 0    │ 3600 │
│ 35726896 │      │ NS          │ ns1.exoscale.com                                                    │ 0    │ 3600 │
│ 35726897 │      │ NS          │ ns1.exoscale.io                                                     │ 0    │ 3600 │
│ 35726898 │      │ NS          │ ns1.exoscale.net                                                    │ 0    │ 3600 │
│ 35726899 │ www  │ CNAME       │ example.exo                                                         │ 0    │ 7200 │
│ 35726900 │      │ TXT         │ hello world!                                                        │ 0    │ 3600 │
│ 35726905 │      │ AAAA        │ 2a04:c43:e00:63a3:43b:eaff:fe00:3f6                                 │ 0    │ 3600 │
│ 35726906 │      │ A           │ 185.19.31.186                                                       │ 0    │ 3600 │
┼──────────┼──────┼─────────────┼─────────────────────────────────────────────────────────────────────┼──────┼──────┼
```

**WARNING:** [Exoscale DNS](https://www.exoscale.com/dns/) requires a subscription!
Please visit the account section in the [Portal](https://portal.exoscale.com/) to activate it.
