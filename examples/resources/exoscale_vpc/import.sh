# An existing VPC may be imported by `<ID>@<zone>`:

terraform import \
  exoscale_vpc.my_vpc \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6@ch-gva-2

# An existing VPC Subnet may be imported by `<vpc-ID>@<subnet-ID>@<zone>`:

terraform import \
  exoscale_vpc_subnet.my_vpc_subnet \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6@9ecc6b8b-73d4-4211-8ced-f7f29bb79524@ch-gva-2

# An existing VPC route may be imported by `<vpc-ID>@<subnet-ID>@<route-ID>@<zone>`:

terraform import \
  exoscale_vpc_route.my_vpc_route \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6@9ecc6b8b-73d4-4211-8ced-f7f29bb79524@9ecc6b8b-73d4-4211-8ced-f7f29bb79525@ch-gva-2

# An existing attachment may be imported by `<instance-ID>@<vpc-ID>@<subnet-ID>@<zone>`:

terraform import \
  'exoscale_vpc_subnet_attachment.my_attachments["my-instance-1"]' \
  f81d4fae-7dec-11d0-a765-00a0c91e6bf6@9ecc6b8b-73d4-4211-8ced-f7f29bb79524@9ecc6b8b-73d4-4211-8ced-f7f29bb79525@ch-gva-2
