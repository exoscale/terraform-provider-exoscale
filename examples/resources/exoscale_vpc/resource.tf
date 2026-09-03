locals {
  zone = "ch-gva-2"
}

resource "exoscale_vpc" "my_vpc" {
  zone        = local.zone
  name        = "my-vpc"
  description = "My Virtual Private Cloud"

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
