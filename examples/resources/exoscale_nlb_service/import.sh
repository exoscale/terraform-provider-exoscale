# An existing NLB service may be imported by `<nlb-ID>/<service-ID>@<zone>`:

terraform import \
  exoscale_nlb_service.my_nlb_service \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6/9ecc6b8b-73d4-4211-8ced-f7f29bb79524@ch-gva-2
