data "exoscale_template" "my_template" {
  zone = "ch-gva-2"
  name = "Linux Ubuntu 22.04 LTS 64-bit"
}

# Attaching an instance to VPC Subnets: repeat the `vpc_interface` block per
# Subnet. All blocks must reference the same VPC, as an instance can only be
# attached to one. The Subnets are attached in the order the blocks are written
# in, and they are kept in that order. Reordering them afterwards reattaches
# nothing: only added, removed and readdressed blocks are acted on.

resource "exoscale_vpc" "my_vpc" {
  zone = "ch-gva-2"
  name = "my-vpc"
}

resource "exoscale_vpc_subnet" "my_vpc_subnet" {
  zone       = "ch-gva-2"
  vpc_id     = exoscale_vpc.my_vpc.id
  name       = "my-vpc-subnet"
  ipv4_block = "10.0.0.0/24"
}

resource "exoscale_compute_instance" "my_instance" {
  zone = "ch-gva-2"
  name = "my-instance"

  template_id = data.exoscale_template.my_template.id
  type        = "standard.medium"
  disk_size   = 10

  vpc_interface {
    vpc_id    = exoscale_vpc.my_vpc.id
    subnet_id = exoscale_vpc_subnet.my_vpc_subnet.id

    # Optional; automatically allocated by the platform if not set.
    ipv4_address = "10.0.0.11"
  }
}
