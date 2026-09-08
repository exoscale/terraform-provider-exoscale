locals {
  zone = "ch-gva-2"
}

data "exoscale_template" "my_template" {
  zone = local.zone
  name = "Linux Ubuntu 22.04 LTS 64-bit"
}

resource "exoscale_compute_instance" "my_instances" {
  for_each = toset(["my-instance-1", "my-instance-2"])

  zone = local.zone
  name = each.key

  template_id = data.exoscale_template.my_template.id
  type        = "standard.medium"
  disk_size   = 10
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

resource "exoscale_vpc_route" "my_vpc_route" {
  zone        = local.zone
  vpc_id      = exoscale_vpc.my_vpc.id
  subnet_id   = exoscale_vpc_subnet.my_vpc_subnet.id
  destination = "10.0.9.0/24"
  target      = "ip=10.0.0.5"
  description = "route to 10.0.9.0/24 via 10.0.0.5"
}

# Attach both instances to the same Subnet using `for_each`.
resource "exoscale_vpc_subnet_attachment" "my_attachments" {
  for_each = exoscale_compute_instance.my_instances

  zone        = local.zone
  instance_id = each.value.id
  vpc_id      = exoscale_vpc.my_vpc.id
  subnet_id   = exoscale_vpc_subnet.my_vpc_subnet.id
}
