data "exoscale_vpc" "my_vpc" {
  zone = "ch-gva-2"
  name = "my-vpc"
}

output "my_vpc_id" {
  value = data.exoscale_vpc.my_vpc.id
}
