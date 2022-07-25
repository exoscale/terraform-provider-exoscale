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

my_instance_ipv4 = "159.100.244.36"
my_instance_ipv6 = "2a04:c43:e00:63a3:464:e4ff:fe00:7c6"
```

Which can then be seen using [Exoscale CLI](https://github.com/exoscale/cli/):

```console
$ exo dns show example.exo
┼──────────────────────────────────────┼──────┼─────────────┼─────────────────────────────────────────────────────────────────────┼──────┼──────┼
│                  ID                  │ NAME │ RECORD TYPE │                               CONTENT                               │ PRIO │ TTL  │
┼──────────────────────────────────────┼──────┼─────────────┼─────────────────────────────────────────────────────────────────────┼──────┼──────┼
│ 89083a5c-b648-474a-0000-000002218b26 │      │ SOA         │ ns1.exoscale.ch admin.dnsimple.com 1658319961 86400 7200 604800 300 │ 0    │ 3600 │
│ 89083a5c-b648-474a-0000-000002218b2b │      │ NS          │ ns1.exoscale.ch                                                     │ 0    │ 3600 │
│ 89083a5c-b648-474a-0000-000002218b2c │      │ NS          │ ns1.exoscale.com                                                    │ 0    │ 3600 │
│ 89083a5c-b648-474a-0000-000002218b2d │      │ NS          │ ns1.exoscale.io                                                     │ 0    │ 3600 │
│ 89083a5c-b648-474a-0000-000002218b2e │      │ NS          │ ns1.exoscale.net                                                    │ 0    │ 3600 │
│ 89083a5c-b648-474a-0000-000002218b2f │      │ TXT         │ hello world!                                                        │ 0    │ 3600 │
│ 89083a5c-b648-474a-0000-000002218b30 │ www  │ CNAME       │ example.exo                                                         │ 0    │ 7200 │
│ 89083a5c-b648-474a-0000-000002218b34 │      │ A           │ 159.100.244.36                                                      │ 0    │ 3600 │
│ 89083a5c-b648-474a-0000-000002218b35 │      │ AAAA        │ 2a04:c43:e00:63a3:464:e4ff:fe00:7c6                                 │ 0    │ 3600 │
┼──────────────────────────────────────┼──────┼─────────────┼─────────────────────────────────────────────────────────────────────┼──────┼──────┼
```

**WARNING:** [Exoscale DNS](https://www.exoscale.com/dns/) requires a subscription!
Please visit the account section in the [Portal](https://portal.exoscale.com/) to activate it.
