data "exoscale_vpc" "my_vpc" {
  zone = "ch-gva-2"
  name = "my-vpc"
}

data "exoscale_vpc_subnet" "my_vpc_subnet" {
  zone   = "ch-gva-2"
  vpc_id = data.exoscale_vpc.my_vpc.id
  name   = "my-vpc-subnet"
}

output "my_vpc_id" {
  value = data.exoscale_vpc.my_vpc.id
}

output "my_vpc_subnet_id" {
  value = data.exoscale_vpc_subnet.my_vpc_subnet.id
}
