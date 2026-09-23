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
