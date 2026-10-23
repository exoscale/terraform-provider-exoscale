locals {
  zone = "ch-gva-2"
}

resource "exoscale_vpc" "my_vpc" {
  zone          = local.zone
  name          = "my-vpc"
  description   = "My Virtual Private Cloud"
  dns_servers   = ["8.8.8.8", "1.1.1.1"]
  ntp_servers   = ["42.42.42.42", "43.43.43.43"]
  domain_search = ["my.domain", "their.domain"]

  labels = {
    environment = "production"
  }
}

resource "exoscale_vpc_subnet" "my_vpc_subnet" {
  zone        = local.zone
  vpc_id      = exoscale_vpc.my_vpc.id
  name        = "my-vpc-subnet"
  description = "My VPC Subnet"
  ipv4_block  = "10.0.0.0/24"

  labels = {
    environment = "production"
  }
}

resource "exoscale_vpc_route" "my_vpc_route" {
  zone        = local.zone
  vpc_id      = exoscale_vpc.my_vpc.id
  subnet_id   = exoscale_vpc_subnet.my_vpc_subnet.id
  destination = "10.0.9.0/24"
  target      = "ip=10.0.0.5"
  description = "route to 10.0.9.0/24 via 10.0.0.5"
}

resource "exoscale_vpc_subnet" "my_other_vpc_subnet" {
  zone       = local.zone
  vpc_id     = exoscale_vpc.my_vpc.id
  name       = "my-other-vpc-subnet"
  ipv4_block = "10.0.1.0/24"
}

data "exoscale_template" "my_template" {
  zone = local.zone
  name = "Linux Ubuntu 26.04 LTS 64-bit"
}

# Instance attached to the VPC subnets. Interfaces are strictly ordered:
# reordering, inserting or removing one reattaches every
# interface from that position on. Removing the vpc block detaches them all.
resource "exoscale_compute_instance" "my_instance" {
  zone = local.zone
  name = "my-instance"

  template_id = data.exoscale_template.my_template.id
  type        = "standard.medium"
  disk_size   = 10

  vpc {
    id = exoscale_vpc.my_vpc.id

    interfaces = [
      {
        subnet_id    = exoscale_vpc_subnet.my_vpc_subnet.id
        ipv4_address = "10.0.0.22"
      },
      {
        # ipv4_address defaults to "auto": the platform picks the address,
        # read it from ipv4_address_allocated.
        subnet_id = exoscale_vpc_subnet.my_other_vpc_subnet.id
      },
    ]
  }
}
