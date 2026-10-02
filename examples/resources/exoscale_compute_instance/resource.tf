data "exoscale_template" "my_template" {
  zone = "ch-gva-2"
  name = "Linux Ubuntu 22.04 LTS 64-bit"
}

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

resource "exoscale_vpc_subnet" "my_vpc_subnet2" {
  zone       = "ch-gva-2"
  vpc_id     = exoscale_vpc.my_vpc.id
  name       = "my-vpc-subnet2"
  ipv4_block = "10.0.1.0/24"
}

resource "exoscale_compute_instance" "my_instance" {
  zone = "ch-gva-2"
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
        subnet_id    = exoscale_vpc_subnet.my_vpc_subnet2.id
        ipv4_address = "auto"
      },
    ]
  }
}
